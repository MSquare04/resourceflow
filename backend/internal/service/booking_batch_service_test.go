package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
	"resourceflow/backend/internal/service"
)

func TestBookingService_PreviewBatchAndCreateBatch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 23, 9, 0, 0, 0, time.UTC)
	location := time.UTC
	baseReq := dto.BatchBookingRequest{
		ResourceID: 10,
		Dates:      []string{"2026-06-24", "2026-06-25"},
		StartTime:  "15:00",
		EndTime:    "16:00",
	}

	newService := func(
		t *testing.T,
		repo *bookingRepoMock,
		unavailabilityFn func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error),
		maxActive int32,
		requiresApproval bool,
	) *service.BookingService {
		t.Helper()

		resources := &resourceRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
				return model.Resource{ID: id, TypeID: 20, IsActive: true, IsBookable: true, Name: "Room 1"}, nil
			},
		}
		users := &userRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.User, error) {
				return model.User{ID: id, FullName: "Tester", IsActive: true}, nil
			},
		}
		rules := &bookingRuleRepoMock{
			findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
				workdayStart, _ := time.Parse("15:04", "09:00")
				workdayEnd, _ := time.Parse("15:04", "18:00")
				return model.BookingRule{
					ID:                       1,
					ResourceTypeID:           resourceTypeID,
					MinDurationMinutes:       30,
					MaxDurationMinutes:       180,
					MaxActiveBookingsPerUser: maxActive,
					RequiresApproval:         requiresApproval,
					BookingHorizonDays:       30,
					WorkdayStart:             workdayStart,
					WorkdayEnd:               workdayEnd,
					IsActive:                 true,
				}, nil
			},
		}

		return service.NewBookingService(repo, resources, users, rules).
			WithTimeLocation(location).
			WithUnavailabilityChecker(&unavailabilityCheckerMock{hasConflictFn: unavailabilityFn})
	}

	t.Run("preview for multiple free dates", func(t *testing.T) {
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 0, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
		}
		svc := newService(t, repo, func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
			return false, nil
		}, 5, false)

		result, err := svc.PreviewBatchAt(context.Background(), 77, baseReq, now)
		if err != nil {
			t.Fatalf("PreviewBatchAt returned error: %v", err)
		}
		if !result.CanCreate || len(result.Items) != 2 {
			t.Fatalf("unexpected preview result: %+v", result)
		}
		for _, item := range result.Items {
			if !item.Valid || item.ErrorCode != nil || item.Status == nil || *item.Status != model.BookingStatusConfirmed {
				t.Fatalf("unexpected preview item: %+v", item)
			}
		}
	})

	t.Run("one conflicting date is marked in preview", func(t *testing.T) {
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 0, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return startAt.Equal(time.Date(2026, time.June, 25, 15, 0, 0, 0, time.UTC)), nil
			},
		}
		svc := newService(t, repo, nil, 5, false)

		result, err := svc.PreviewBatchAt(context.Background(), 77, baseReq, now)
		if err != nil {
			t.Fatalf("PreviewBatchAt returned error: %v", err)
		}
		if result.CanCreate {
			t.Fatalf("expected preview to be invalid: %+v", result)
		}
		if !result.Items[0].Valid || result.Items[1].Valid {
			t.Fatalf("unexpected preview item states: %+v", result.Items)
		}
		if result.Items[1].ErrorCode == nil || *result.Items[1].ErrorCode != dto.ErrorCodeBookingConflict {
			t.Fatalf("unexpected preview conflict code: %+v", result.Items[1])
		}
	})

	t.Run("batch with conflict creates nothing", func(t *testing.T) {
		createdCount := 0
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 0, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return startAt.Equal(time.Date(2026, time.June, 25, 15, 0, 0, 0, time.UTC)), nil
			},
			createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
				createdCount++
				return model.Booking{}, nil
			},
		}
		svc := newService(t, repo, nil, 5, false)

		_, err := svc.CreateBatchAt(context.Background(), 77, baseReq, now)
		var batchErr *service.BookingBatchValidationError
		if !errors.As(err, &batchErr) {
			t.Fatalf("expected BookingBatchValidationError, got %v", err)
		}
		if createdCount != 0 {
			t.Fatalf("expected no bookings to be created, got %d", createdCount)
		}
	})

	t.Run("valid series is created completely", func(t *testing.T) {
		var created []repository.CreateBookingParams
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return int64(len(created)), nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
				created = append(created, params)
				return model.Booking{
					ID:         int64(len(created)),
					ResourceID: params.ResourceID,
					UserID:     params.UserID,
					StartAt:    params.StartAt,
					EndAt:      params.EndAt,
					Status:     params.Status,
					CreatedAt:  now,
					UpdatedAt:  now,
				}, nil
			},
		}
		svc := newService(t, repo, nil, 5, true)

		result, err := svc.CreateBatchAt(context.Background(), 77, baseReq, now)
		if err != nil {
			t.Fatalf("CreateBatchAt returned error: %v", err)
		}
		if result.CreatedCount != 2 || len(result.Items) != 2 {
			t.Fatalf("unexpected created result: %+v", result)
		}
		for _, item := range result.Items {
			if item.Status != model.BookingStatusPending {
				t.Fatalf("expected pending status, got %+v", item)
			}
		}
	})

	t.Run("duplicate dates are rejected", func(t *testing.T) {
		svc := newService(t, &bookingRepoMock{}, nil, 5, false)
		_, err := svc.PreviewBatchAt(context.Background(), 77, dto.BatchBookingRequest{
			ResourceID: 10,
			Dates:      []string{"2026-06-24", "2026-06-24"},
			StartTime:  "15:00",
			EndTime:    "16:00",
		}, now)
		if !errors.Is(err, service.ErrBookingBatchDuplicateDate) {
			t.Fatalf("expected ErrBookingBatchDuplicateDate, got %v", err)
		}
	})

	t.Run("more than 31 dates is rejected", func(t *testing.T) {
		dates := make([]string, 0, 32)
		for day := 1; day <= 32; day++ {
			dates = append(dates, time.Date(2026, time.July, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
		}
		svc := newService(t, &bookingRepoMock{}, nil, 40, false)
		_, err := svc.PreviewBatchAt(context.Background(), 77, dto.BatchBookingRequest{
			ResourceID: 10,
			Dates:      dates,
			StartTime:  "15:00",
			EndTime:    "16:00",
		}, now)
		if !errors.Is(err, service.ErrBookingBatchTooLarge) {
			t.Fatalf("expected ErrBookingBatchTooLarge, got %v", err)
		}
	})

	t.Run("total active limit is applied to the whole series", func(t *testing.T) {
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 2, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
		}
		svc := newService(t, repo, nil, 3, false)

		result, err := svc.PreviewBatchAt(context.Background(), 77, baseReq, now)
		if err != nil {
			t.Fatalf("PreviewBatchAt returned error: %v", err)
		}
		if result.Items[0].ErrorCode != nil {
			t.Fatalf("first item should still be valid: %+v", result.Items[0])
		}
		if result.Items[1].ErrorCode == nil || *result.Items[1].ErrorCode != dto.ErrorCodeBookingLimitExceeded {
			t.Fatalf("second item should fail by limit: %+v", result.Items[1])
		}
	})

	t.Run("workday and resource unavailability are respected", func(t *testing.T) {
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 0, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
		}
		svc := newService(t, repo, func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
			return startAt.Equal(time.Date(2026, time.June, 25, 15, 0, 0, 0, time.UTC)), nil
		}, 5, false)

		result, err := svc.PreviewBatchAt(context.Background(), 77, dto.BatchBookingRequest{
			ResourceID: 10,
			Dates:      []string{"2026-06-24", "2026-06-25"},
			StartTime:  "08:00",
			EndTime:    "09:00",
		}, now)
		if err != nil {
			t.Fatalf("PreviewBatchAt returned error: %v", err)
		}
		if result.Items[0].ErrorCode == nil || *result.Items[0].ErrorCode != dto.ErrorCodeBookingOutsideWorkday {
			t.Fatalf("expected first item to fail by workday: %+v", result.Items[0])
		}
	})

	t.Run("resource unavailability is reflected in preview", func(t *testing.T) {
		repo := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 0, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
		}
		svc := newService(t, repo, func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
			return startAt.Equal(time.Date(2026, time.June, 25, 15, 0, 0, 0, time.UTC)), nil
		}, 5, false)

		result, err := svc.PreviewBatchAt(context.Background(), 77, baseReq, now)
		if err != nil {
			t.Fatalf("PreviewBatchAt returned error: %v", err)
		}
		if result.Items[1].ErrorCode == nil || *result.Items[1].ErrorCode != dto.ErrorCodeBookingInUnavailability {
			t.Fatalf("expected unavailability code, got %+v", result.Items[1])
		}
	})

	t.Run("concurrent conflict during insert rolls back the whole transaction", func(t *testing.T) {
		state := make([]model.Booking, 0)
		repo := &bookingRepoMock{}
		repo.withTransactionFn = func(ctx context.Context, fn func(repo repository.BookingRepository) error) error {
			snapshot := append([]model.Booking(nil), state...)
			txRepo := &bookingRepoMock{
				countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
					var count int64
					for _, booking := range snapshot {
						if booking.UserID == userID && (booking.Status == model.BookingStatusPending || booking.Status == model.BookingStatusConfirmed) {
							count++
						}
					}
					return count, nil
				},
				hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
					return false, nil
				},
				createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
					if params.StartAt.Equal(time.Date(2026, time.June, 25, 15, 0, 0, 0, time.UTC)) {
						return model.Booking{}, &pq.Error{Code: "23P01"}
					}
					booking := model.Booking{
						ID:         int64(len(snapshot) + 1),
						ResourceID: params.ResourceID,
						UserID:     params.UserID,
						StartAt:    params.StartAt,
						EndAt:      params.EndAt,
						Status:     params.Status,
						CreatedAt:  now,
						UpdatedAt:  now,
					}
					snapshot = append(snapshot, booking)
					return booking, nil
				},
			}
			if err := fn(txRepo); err != nil {
				return err
			}
			state = snapshot
			return nil
		}
		repo.countByStatusesFn = func(ctx context.Context, userID int64, statuses []string) (int64, error) { return 0, nil }
		repo.hasConflictFn = func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
			return false, nil
		}

		svc := newService(t, repo, nil, 5, false)
		_, err := svc.CreateBatchAt(context.Background(), 77, baseReq, now)
		var batchErr *service.BookingBatchValidationError
		if !errors.As(err, &batchErr) {
			t.Fatalf("expected BookingBatchValidationError, got %v", err)
		}
		if len(state) != 0 {
			t.Fatalf("expected transaction rollback, got persisted bookings: %+v", state)
		}
	})
}
