package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
	"resourceflow/backend/internal/service"
)

func TestBookingService_Create_StatusByRequiresApproval(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	req := dto.CreateBookingRequest{
		ResourceID: 10,
		StartAt:    now.Add(2 * time.Hour),
		EndAt:      now.Add(3 * time.Hour),
	}

	tests := []struct {
		name            string
		requiresApprove bool
		wantStatus      string
	}{
		{name: "pending when approval required", requiresApprove: true, wantStatus: model.BookingStatusPending},
		{name: "confirmed when approval not required", requiresApprove: false, wantStatus: model.BookingStatusConfirmed},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bookings := &bookingRepoMock{
				isCoveredFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
					return true, nil
				},
				countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
					return 0, nil
				},
				hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
					return false, nil
				},
				createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
					if params.Status != tc.wantStatus {
						t.Fatalf("unexpected status in create params: got %q want %q", params.Status, tc.wantStatus)
					}
					return model.Booking{
						ID:         1,
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
			resources := &resourceRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
					return model.Resource{ID: id, TypeID: 20, IsActive: true, IsBookable: true, Name: "Room 1"}, nil
				},
			}
			users := &userRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.User, error) {
					return model.User{ID: id, FullName: "Test User", IsActive: true}, nil
				},
			}
			rules := &bookingRuleRepoMock{
				findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
					return model.BookingRule{
						ID:                       100,
						ResourceTypeID:           resourceTypeID,
						MinDurationMinutes:       15,
						MaxDurationMinutes:       180,
						MaxActiveBookingsPerUser: 5,
						RequiresApproval:         tc.requiresApprove,
						BookingHorizonDays:       30,
						IsActive:                 true,
					}, nil
				},
			}

			svc := service.NewBookingService(bookings, resources, users, rules)
			got, err := svc.Create(context.Background(), 55, req)
			if err != nil {
				t.Fatalf("Create returned error: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Fatalf("unexpected status: got %q want %q", got.Status, tc.wantStatus)
			}
		})
	}
}

func TestBookingService_Create_Errors(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)

	baseReq := dto.CreateBookingRequest{
		ResourceID: 10,
		StartAt:    now.Add(2 * time.Hour),
		EndAt:      now.Add(3 * time.Hour),
	}

	baseResource := &resourceRepoMock{
		findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
			return model.Resource{ID: id, TypeID: 20, IsActive: true, IsBookable: true}, nil
		},
	}
	baseUsers := &userRepoMock{
		findByIDFn: func(ctx context.Context, id int64) (model.User, error) {
			return model.User{ID: id, IsActive: true}, nil
		},
	}

	t.Run("overlap returns conflict", func(t *testing.T) {
		bookings := &bookingRepoMock{
			isCoveredFn:       func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) { return true, nil },
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) { return 0, nil },
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return true, nil
			},
		}
		rules := &bookingRuleRepoMock{
			findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
				return model.BookingRule{
					ResourceTypeID:           resourceTypeID,
					MinDurationMinutes:       10,
					MaxDurationMinutes:       240,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					IsActive:                 true,
				}, nil
			},
		}
		svc := service.NewBookingService(bookings, baseResource, baseUsers, rules)

		_, err := svc.Create(context.Background(), 55, baseReq)
		if !errors.Is(err, service.ErrBookingConflict) {
			t.Fatalf("expected ErrBookingConflict, got %v", err)
		}
	})

	t.Run("outside availability returns error", func(t *testing.T) {
		bookings := &bookingRepoMock{
			isCoveredFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) { return false, nil },
		}
		rules := &bookingRuleRepoMock{}
		svc := service.NewBookingService(bookings, baseResource, baseUsers, rules)

		_, err := svc.Create(context.Background(), 55, baseReq)
		if !errors.Is(err, service.ErrBookingOutOfAvailability) {
			t.Fatalf("expected ErrBookingOutOfAvailability, got %v", err)
		}
	})

	t.Run("invalid interval returns validation", func(t *testing.T) {
		svc := service.NewBookingService(&bookingRepoMock{}, baseResource, baseUsers, &bookingRuleRepoMock{})
		req := baseReq
		req.StartAt = req.EndAt

		_, err := svc.Create(context.Background(), 55, req)
		if !errors.Is(err, service.ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})

	t.Run("booking rule violation returns validation", func(t *testing.T) {
		bookings := &bookingRepoMock{
			isCoveredFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) { return true, nil },
		}
		rules := &bookingRuleRepoMock{
			findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
				return model.BookingRule{
					ResourceTypeID:           resourceTypeID,
					MinDurationMinutes:       120,
					MaxDurationMinutes:       240,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					IsActive:                 true,
				}, nil
			},
		}
		svc := service.NewBookingService(bookings, baseResource, baseUsers, rules)

		_, err := svc.Create(context.Background(), 55, baseReq) // duration is 60m
		if !errors.Is(err, service.ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})
}

func TestBookingService_Actions_StatusRules(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)

	t.Run("approve works only for pending", func(t *testing.T) {
		svcOK := service.NewBookingService(
			&bookingRepoMock{
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					return model.Booking{
						ID:         id,
						ResourceID: 10,
						UserID:     77,
						Status:     model.BookingStatusConfirmed,
						StartAt:    now,
						EndAt:      now.Add(time.Hour),
					}, model.BookingStatusPending, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		resp, err := svcOK.Approve(context.Background(), 1, 1000)
		if err != nil {
			t.Fatalf("Approve should succeed, got %v", err)
		}
		if resp.Status != model.BookingStatusConfirmed {
			t.Fatalf("unexpected status after approve: %q", resp.Status)
		}

		svcFail := service.NewBookingService(
			&bookingRepoMock{
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					return model.Booking{}, "", sql.ErrNoRows
				},
				findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusConfirmed, StartAt: now, EndAt: now.Add(time.Hour)}, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		_, err = svcFail.Approve(context.Background(), 1, 1000)
		if !errors.Is(err, service.ErrBookingInvalidStatusAction) {
			t.Fatalf("expected ErrBookingInvalidStatusAction, got %v", err)
		}
	})

	t.Run("reject works only for pending", func(t *testing.T) {
		svcOK := service.NewBookingService(
			&bookingRepoMock{
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusRejected, StartAt: now, EndAt: now.Add(time.Hour)}, model.BookingStatusPending, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		resp, err := svcOK.Reject(context.Background(), 1, 1000)
		if err != nil {
			t.Fatalf("Reject should succeed, got %v", err)
		}
		if resp.Status != model.BookingStatusRejected {
			t.Fatalf("unexpected status after reject: %q", resp.Status)
		}

		svcFail := service.NewBookingService(
			&bookingRepoMock{
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					return model.Booking{}, "", sql.ErrNoRows
				},
				findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusCancelled, StartAt: now, EndAt: now.Add(time.Hour)}, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		_, err = svcFail.Reject(context.Background(), 1, 1000)
		if !errors.Is(err, service.ErrBookingInvalidStatusAction) {
			t.Fatalf("expected ErrBookingInvalidStatusAction, got %v", err)
		}
	})

	t.Run("cancel works only for pending or confirmed", func(t *testing.T) {
		for _, from := range []string{model.BookingStatusPending, model.BookingStatusConfirmed} {
			from := from
			t.Run("from_"+from, func(t *testing.T) {
				svc := service.NewBookingService(
					&bookingRepoMock{
						findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
							return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: from, StartAt: now, EndAt: now.Add(time.Hour)}, nil
						},
						transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
							return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusCancelled, StartAt: now, EndAt: now.Add(time.Hour)}, from, nil
						},
					},
					&resourceRepoMock{},
					&userRepoMock{},
					&bookingRuleRepoMock{},
				)
				resp, err := svc.Cancel(context.Background(), 1, 77, false)
				if err != nil {
					t.Fatalf("Cancel should succeed from %s, got %v", from, err)
				}
				if resp.Status != model.BookingStatusCancelled {
					t.Fatalf("unexpected status after cancel: %q", resp.Status)
				}
			})
		}

		svcFail := service.NewBookingService(
			&bookingRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusRejected, StartAt: now, EndAt: now.Add(time.Hour)}, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		_, err := svcFail.Cancel(context.Background(), 1, 77, false)
		if !errors.Is(err, service.ErrBookingInvalidStatusAction) {
			t.Fatalf("expected ErrBookingInvalidStatusAction, got %v", err)
		}
	})

	t.Run("complete works only for confirmed", func(t *testing.T) {
		svcOK := service.NewBookingService(
			&bookingRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusConfirmed, StartAt: now, EndAt: now.Add(time.Hour)}, nil
				},
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusCompleted, StartAt: now, EndAt: now.Add(time.Hour)}, model.BookingStatusConfirmed, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		resp, err := svcOK.Complete(context.Background(), 1, 77, false)
		if err != nil {
			t.Fatalf("Complete should succeed, got %v", err)
		}
		if resp.Status != model.BookingStatusCompleted {
			t.Fatalf("unexpected status after complete: %q", resp.Status)
		}

		svcFail := service.NewBookingService(
			&bookingRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusPending, StartAt: now, EndAt: now.Add(time.Hour)}, nil
				},
			},
			&resourceRepoMock{},
			&userRepoMock{},
			&bookingRuleRepoMock{},
		)
		_, err = svcFail.Complete(context.Background(), 1, 77, false)
		if !errors.Is(err, service.ErrBookingInvalidStatusAction) {
			t.Fatalf("expected ErrBookingInvalidStatusAction, got %v", err)
		}
	})
}
