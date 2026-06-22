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
			got, err := svc.CreateAt(context.Background(), 55, req, now)
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

		_, err := svc.CreateAt(context.Background(), 55, baseReq, now)
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

		_, err := svc.CreateAt(context.Background(), 55, baseReq, now)
		if !errors.Is(err, service.ErrBookingOutOfAvailability) {
			t.Fatalf("expected ErrBookingOutOfAvailability, got %v", err)
		}
	})

	t.Run("booking without availability is allowed", func(t *testing.T) {
		bookings := &bookingRepoMock{
			isCoveredFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) { return true, nil },
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
				return 0, nil
			},
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
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

		if _, err := svc.CreateAt(context.Background(), 55, baseReq, now); err != nil {
			t.Fatalf("expected booking without availability to succeed, got %v", err)
		}
	})

	t.Run("invalid interval returns validation", func(t *testing.T) {
		svc := service.NewBookingService(&bookingRepoMock{}, baseResource, baseUsers, &bookingRuleRepoMock{})
		req := baseReq
		req.StartAt = req.EndAt

		_, err := svc.CreateAt(context.Background(), 55, req, now)
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

		_, err := svc.CreateAt(context.Background(), 55, baseReq, now) // duration is 60m
		if !errors.Is(err, service.ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})
}

func TestBookingService_Create_CoreInvariants(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 34, 45, 0, time.UTC)
	baseReq := dto.CreateBookingRequest{
		ResourceID: 10,
		StartAt:    now.Add(2 * time.Hour),
		EndAt:      now.Add(3 * time.Hour),
	}

	baseUsers := &userRepoMock{
		findByIDFn: func(ctx context.Context, id int64) (model.User, error) {
			return model.User{ID: id, IsActive: true}, nil
		},
	}
	baseRule := &bookingRuleRepoMock{
		findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
			return model.BookingRule{
				ResourceTypeID:           resourceTypeID,
				MinDurationMinutes:       15,
				MaxDurationMinutes:       180,
				MaxActiveBookingsPerUser: 3,
				BookingHorizonDays:       30,
				IsActive:                 true,
			}, nil
		},
	}

	tests := []struct {
		name     string
		req      dto.CreateBookingRequest
		resource model.Resource
		covered  bool
		conflict bool
		wantErr  error
	}{
		{
			name:     "success",
			req:      baseReq,
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
			covered:  true,
		},
		{
			name:     "start in past",
			req:      dto.CreateBookingRequest{ResourceID: 10, StartAt: now.Truncate(time.Minute).Add(-time.Minute), EndAt: now.Add(time.Hour)},
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
			wantErr:  service.ErrBookingStartNotFuture,
		},
		{
			name:     "outside availability",
			req:      baseReq,
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
			covered:  false,
			wantErr:  service.ErrBookingOutOfAvailability,
		},
		{
			name:     "conflict with active booking",
			req:      baseReq,
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
			covered:  true,
			conflict: true,
			wantErr:  service.ErrBookingConflict,
		},
		{
			name:     "resource unavailable",
			req:      baseReq,
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: false, IsBookable: true},
			wantErr:  service.ErrBookingResourceUnavailable,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bookings := &bookingRepoMock{
				isCoveredFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
					return tc.covered, nil
				},
				countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) {
					return 0, nil
				},
				hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
					return tc.conflict, nil
				},
				createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
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
					return tc.resource, nil
				},
			}

			svc := service.NewBookingService(bookings, resources, baseUsers, baseRule)
			_, err := svc.CreateAt(context.Background(), 55, tc.req, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestBookingService_CreateAt_CurrentMinuteSemantics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 34, 45, 0, time.UTC)
	currentMinute := now.Truncate(time.Minute)

	newService := func(t *testing.T) *service.BookingService {
		t.Helper()

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
				return model.Resource{ID: id, TypeID: 20, IsActive: true, IsBookable: true}, nil
			},
		}
		users := &userRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.User, error) {
				return model.User{ID: id, IsActive: true}, nil
			},
		}
		rules := &bookingRuleRepoMock{
			findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
				return model.BookingRule{
					ResourceTypeID:           resourceTypeID,
					MinDurationMinutes:       15,
					MaxDurationMinutes:       180,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					IsActive:                 true,
				}, nil
			},
		}

		return service.NewBookingService(bookings, resources, users, rules)
	}

	t.Run("start exactly at current minute is allowed", func(t *testing.T) {
		t.Parallel()

		svc := newService(t)
		req := dto.CreateBookingRequest{
			ResourceID: 10,
			StartAt:    currentMinute,
			EndAt:      currentMinute.Add(30 * time.Minute),
		}

		resp, err := svc.CreateAt(context.Background(), 55, req, now)
		if err != nil {
			t.Fatalf("CreateAt returned error: %v", err)
		}
		if !resp.StartAt.Equal(currentMinute) {
			t.Fatalf("unexpected start_at: got %s want %s", resp.StartAt, currentMinute)
		}
	})

	t.Run("start one minute earlier is forbidden", func(t *testing.T) {
		t.Parallel()

		svc := newService(t)
		req := dto.CreateBookingRequest{
			ResourceID: 10,
			StartAt:    currentMinute.Add(-time.Minute),
			EndAt:      currentMinute.Add(29 * time.Minute),
		}

		_, err := svc.CreateAt(context.Background(), 55, req, now)
		if !errors.Is(err, service.ErrBookingStartNotFuture) {
			t.Fatalf("expected ErrBookingStartNotFuture, got %v", err)
		}
	})

	t.Run("future start is allowed", func(t *testing.T) {
		t.Parallel()

		svc := newService(t)
		req := dto.CreateBookingRequest{
			ResourceID: 10,
			StartAt:    currentMinute.Add(time.Minute),
			EndAt:      currentMinute.Add(31 * time.Minute),
		}

		if _, err := svc.CreateAt(context.Background(), 55, req, now); err != nil {
			t.Fatalf("CreateAt returned error: %v", err)
		}
	})

	t.Run("other validations still run", func(t *testing.T) {
		t.Parallel()

		bookings := &bookingRepoMock{
			isCoveredFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
				return false, nil
			},
		}
		resources := &resourceRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
				return model.Resource{ID: id, TypeID: 20, IsActive: true, IsBookable: true}, nil
			},
		}
		users := &userRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.User, error) {
				return model.User{ID: id, IsActive: true}, nil
			},
		}
		rules := &bookingRuleRepoMock{
			findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
				return model.BookingRule{
					ResourceTypeID:           resourceTypeID,
					MinDurationMinutes:       15,
					MaxDurationMinutes:       180,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					IsActive:                 true,
				}, nil
			},
		}

		svc := service.NewBookingService(bookings, resources, users, rules)
		req := dto.CreateBookingRequest{
			ResourceID: 10,
			StartAt:    currentMinute,
			EndAt:      currentMinute.Add(30 * time.Minute),
		}

		_, err := svc.CreateAt(context.Background(), 55, req, now)
		if !errors.Is(err, service.ErrBookingOutOfAvailability) {
			t.Fatalf("expected ErrBookingOutOfAvailability, got %v", err)
		}
	})
}

func TestBookingService_ListBusyIntervalsByResourceID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	bookings := &bookingRepoMock{
		listBusyFn: func(ctx context.Context, resourceID int64, statuses []string, from, until time.Time) ([]model.Booking, error) {
			if resourceID != 10 {
				t.Fatalf("unexpected resource id: %d", resourceID)
			}
			if len(statuses) != 2 || statuses[0] != model.BookingStatusPending || statuses[1] != model.BookingStatusConfirmed {
				t.Fatalf("unexpected statuses: %#v", statuses)
			}
			if !until.After(from) {
				t.Fatalf("expected until > from")
			}

			return []model.Booking{
				{StartAt: now.Add(3 * time.Hour), EndAt: now.Add(4 * time.Hour)},
				{StartAt: now.Add(6 * time.Hour), EndAt: now.Add(7 * time.Hour)},
			}, nil
		},
	}
	resources := &resourceRepoMock{
		findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
			return model.Resource{ID: id, Name: "Room 1"}, nil
		},
	}

	svc := service.NewBookingService(bookings, resources, &userRepoMock{}, &bookingRuleRepoMock{})
	got, err := svc.ListBusyIntervalsByResourceID(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListBusyIntervalsByResourceID returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected intervals count: got %d want 2", len(got))
	}
	if !got[0].StartAt.Equal(now.Add(3*time.Hour)) || !got[1].EndAt.Equal(now.Add(7*time.Hour)) {
		t.Fatalf("unexpected intervals payload: %#v", got)
	}
}

func TestBookingService_StatusTransitions_TableDriven(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		name          string
		action        string
		fromStatus    string
		isPrivileged  bool
		actorUserID   int64
		bookingUserID int64
		wantStatus    string
		wantErr       error
	}{
		{name: "approve pending", action: "approve", fromStatus: model.BookingStatusPending, wantStatus: model.BookingStatusConfirmed},
		{name: "reject pending", action: "reject", fromStatus: model.BookingStatusPending, wantStatus: model.BookingStatusRejected},
		{name: "cancel pending owner", action: "cancel", fromStatus: model.BookingStatusPending, actorUserID: 77, bookingUserID: 77, wantStatus: model.BookingStatusCancelled},
		{name: "cancel confirmed owner", action: "cancel", fromStatus: model.BookingStatusConfirmed, actorUserID: 77, bookingUserID: 77, wantStatus: model.BookingStatusCancelled},
		{name: "complete confirmed owner", action: "complete", fromStatus: model.BookingStatusConfirmed, actorUserID: 77, bookingUserID: 77, wantStatus: model.BookingStatusCompleted},
		{name: "approve final forbidden", action: "approve", fromStatus: model.BookingStatusCancelled, wantErr: service.ErrBookingInvalidStatusAction},
		{name: "reject final forbidden", action: "reject", fromStatus: model.BookingStatusCompleted, wantErr: service.ErrBookingInvalidStatusAction},
		{name: "cancel final forbidden", action: "cancel", fromStatus: model.BookingStatusRejected, actorUserID: 77, bookingUserID: 77, wantErr: service.ErrBookingInvalidStatusAction},
		{name: "complete pending forbidden", action: "complete", fromStatus: model.BookingStatusPending, actorUserID: 77, bookingUserID: 77, wantErr: service.ErrBookingInvalidStatusAction},
		{name: "cancel чужую booking forbidden", action: "cancel", fromStatus: model.BookingStatusConfirmed, actorUserID: 77, bookingUserID: 99, wantErr: service.ErrBookingNotFound},
		{name: "complete чужую booking forbidden", action: "complete", fromStatus: model.BookingStatusConfirmed, actorUserID: 77, bookingUserID: 99, wantErr: service.ErrBookingNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bookingID := int64(1)
			bookingUserID := tc.bookingUserID
			if bookingUserID == 0 {
				bookingUserID = 77
			}
			actorUserID := tc.actorUserID
			if actorUserID == 0 {
				actorUserID = 1000
			}

			repo := &bookingRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
					endAt := now.Add(time.Hour)
					if tc.action == "complete" && tc.fromStatus == model.BookingStatusConfirmed {
						endAt = now.Add(-time.Minute)
					}
					return model.Booking{ID: id, ResourceID: 10, UserID: bookingUserID, Status: tc.fromStatus, StartAt: now, EndAt: endAt}, nil
				},
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					allowed := false
					for _, status := range expectedFrom {
						if status == tc.fromStatus {
							allowed = true
							break
						}
					}
					if !allowed {
						return model.Booking{}, "", sql.ErrNoRows
					}
					endAt := now.Add(time.Hour)
					if tc.action == "complete" && tc.fromStatus == model.BookingStatusConfirmed {
						endAt = now.Add(-time.Minute)
					}
					return model.Booking{ID: id, ResourceID: 10, UserID: bookingUserID, Status: params.Status, StartAt: now, EndAt: endAt}, tc.fromStatus, nil
				},
			}

			svc := service.NewBookingService(repo, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})

			var (
				resp dto.BookingResponse
				err  error
			)
			switch tc.action {
			case "approve":
				resp, err = svc.Approve(context.Background(), bookingID, actorUserID)
			case "reject":
				resp, err = svc.Reject(context.Background(), bookingID, actorUserID)
			case "cancel":
				resp, err = svc.Cancel(context.Background(), bookingID, actorUserID, tc.isPrivileged)
			case "complete":
				resp, err = svc.Complete(context.Background(), bookingID, actorUserID, tc.isPrivileged)
			}

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr == nil && resp.Status != tc.wantStatus {
				t.Fatalf("unexpected status: got %q want %q", resp.Status, tc.wantStatus)
			}
		})
	}
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
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusConfirmed, StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Minute)}, nil
				},
				transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
					return model.Booking{ID: id, ResourceID: 10, UserID: 77, Status: model.BookingStatusCompleted, StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Minute)}, model.BookingStatusConfirmed, nil
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

func TestBookingService_ProcessExpiredBookings(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		result        repository.ExpiredBookingProcessingResult
		wantCompleted int64
		wantCancelled int64
	}{
		{
			name:          "expired confirmed becomes completed",
			result:        repository.ExpiredBookingProcessingResult{CompletedCount: 1, CancelledCount: 0},
			wantCompleted: 1,
			wantCancelled: 0,
		},
		{
			name:          "expired pending becomes cancelled",
			result:        repository.ExpiredBookingProcessingResult{CompletedCount: 0, CancelledCount: 1},
			wantCompleted: 0,
			wantCancelled: 1,
		},
		{
			name:          "future and terminal statuses stay unchanged",
			result:        repository.ExpiredBookingProcessingResult{CompletedCount: 0, CancelledCount: 0},
			wantCompleted: 0,
			wantCancelled: 0,
		},
		{
			name:          "repeated run is idempotent",
			result:        repository.ExpiredBookingProcessingResult{CompletedCount: 0, CancelledCount: 0},
			wantCompleted: 0,
			wantCancelled: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bookings := &bookingRepoMock{
				processExpiredFn: func(ctx context.Context, gotNow time.Time) (repository.ExpiredBookingProcessingResult, error) {
					if !gotNow.Equal(now) {
						t.Fatalf("unexpected now: got %s want %s", gotNow, now)
					}
					return tc.result, nil
				},
			}

			svc := service.NewBookingService(bookings, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})
			result, err := svc.ProcessExpiredBookings(context.Background(), now)
			if err != nil {
				t.Fatalf("ProcessExpiredBookings returned error: %v", err)
			}
			if result.CompletedCount != tc.wantCompleted || result.CancelledCount != tc.wantCancelled {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestBookingService_ProcessExpiredBookings_StateMatrix(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)
	bookingsState := []model.Booking{
		{ID: 1, Status: model.BookingStatusConfirmed, EndAt: now.Add(-time.Minute)},
		{ID: 2, Status: model.BookingStatusPending, EndAt: now.Add(-time.Minute)},
		{ID: 3, Status: model.BookingStatusConfirmed, EndAt: now.Add(time.Hour)},
		{ID: 4, Status: model.BookingStatusPending, EndAt: now.Add(time.Hour)},
		{ID: 5, Status: model.BookingStatusCancelled, EndAt: now.Add(-time.Hour)},
		{ID: 6, Status: model.BookingStatusRejected, EndAt: now.Add(-time.Hour)},
		{ID: 7, Status: model.BookingStatusCompleted, EndAt: now.Add(-time.Hour)},
	}

	bookings := &bookingRepoMock{
		processExpiredFn: func(ctx context.Context, gotNow time.Time) (repository.ExpiredBookingProcessingResult, error) {
			if !gotNow.Equal(now) {
				t.Fatalf("unexpected now: got %s want %s", gotNow, now)
			}

			var result repository.ExpiredBookingProcessingResult
			for index := range bookingsState {
				booking := &bookingsState[index]
				if booking.EndAt.After(gotNow) {
					continue
				}

				switch booking.Status {
				case model.BookingStatusConfirmed:
					booking.Status = model.BookingStatusCompleted
					completedAt := gotNow
					booking.CompletedAt = &completedAt
					result.CompletedCount++
				case model.BookingStatusPending:
					booking.Status = model.BookingStatusCancelled
					cancelledAt := gotNow
					booking.CancelledAt = &cancelledAt
					result.CancelledCount++
				}
			}

			return result, nil
		},
	}

	svc := service.NewBookingService(bookings, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})

	firstRun, err := svc.ProcessExpiredBookings(context.Background(), now)
	if err != nil {
		t.Fatalf("first ProcessExpiredBookings returned error: %v", err)
	}
	if firstRun.CompletedCount != 1 || firstRun.CancelledCount != 1 {
		t.Fatalf("unexpected first run result: %+v", firstRun)
	}

	if bookingsState[0].Status != model.BookingStatusCompleted || bookingsState[0].CompletedAt == nil || !bookingsState[0].CompletedAt.Equal(now) {
		t.Fatalf("expired confirmed booking was not completed correctly: %+v", bookingsState[0])
	}
	if bookingsState[1].Status != model.BookingStatusCancelled || bookingsState[1].CancelledAt == nil || !bookingsState[1].CancelledAt.Equal(now) {
		t.Fatalf("expired pending booking was not cancelled correctly: %+v", bookingsState[1])
	}
	if bookingsState[2].Status != model.BookingStatusConfirmed || bookingsState[3].Status != model.BookingStatusPending {
		t.Fatalf("future bookings changed unexpectedly: %+v %+v", bookingsState[2], bookingsState[3])
	}
	if bookingsState[5].Status != model.BookingStatusRejected || bookingsState[6].Status != model.BookingStatusCompleted || bookingsState[4].Status != model.BookingStatusCancelled {
		t.Fatalf("terminal bookings changed unexpectedly: %+v %+v %+v", bookingsState[4], bookingsState[5], bookingsState[6])
	}

	secondRun, err := svc.ProcessExpiredBookings(context.Background(), now)
	if err != nil {
		t.Fatalf("second ProcessExpiredBookings returned error: %v", err)
	}
	if secondRun.CompletedCount != 0 || secondRun.CancelledCount != 0 {
		t.Fatalf("expected idempotent second run, got %+v", secondRun)
	}
}

func TestBookingService_ProcessExpiredBookings_RemovesExpiredFromBusyIntervals(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)
	bookingsState := []model.Booking{
		{ID: 1, ResourceID: 10, UserID: 77, Status: model.BookingStatusConfirmed, StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Minute)},
		{ID: 2, ResourceID: 10, UserID: 77, Status: model.BookingStatusPending, StartAt: now.Add(-90 * time.Minute), EndAt: now.Add(-time.Minute)},
		{ID: 3, ResourceID: 10, UserID: 77, Status: model.BookingStatusConfirmed, StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)},
		{ID: 4, ResourceID: 10, UserID: 77, Status: model.BookingStatusPending, StartAt: now.Add(3 * time.Hour), EndAt: now.Add(4 * time.Hour)},
	}

	bookings := &bookingRepoMock{
		processExpiredFn: func(ctx context.Context, gotNow time.Time) (repository.ExpiredBookingProcessingResult, error) {
			var result repository.ExpiredBookingProcessingResult
			for index := range bookingsState {
				booking := &bookingsState[index]
				if booking.EndAt.After(gotNow) {
					continue
				}

				switch booking.Status {
				case model.BookingStatusConfirmed:
					booking.Status = model.BookingStatusCompleted
					completedAt := gotNow
					booking.CompletedAt = &completedAt
					result.CompletedCount++
				case model.BookingStatusPending:
					booking.Status = model.BookingStatusCancelled
					cancelledAt := gotNow
					booking.CancelledAt = &cancelledAt
					result.CancelledCount++
				}
			}

			return result, nil
		},
		listBusyFn: func(ctx context.Context, resourceID int64, statuses []string, from, until time.Time) ([]model.Booking, error) {
			var result []model.Booking
			for _, booking := range bookingsState {
				if booking.ResourceID != resourceID {
					continue
				}

				active := false
				for _, status := range statuses {
					if booking.Status == status {
						active = true
						break
					}
				}
				if !active {
					continue
				}
				if !booking.EndAt.After(from) || !booking.StartAt.Before(until) {
					continue
				}

				result = append(result, booking)
			}

			return result, nil
		},
	}
	resources := &resourceRepoMock{
		findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
			return model.Resource{ID: id, Name: "Room 1"}, nil
		},
	}

	svc := service.NewBookingService(bookings, resources, &userRepoMock{}, &bookingRuleRepoMock{})
	if _, err := svc.ProcessExpiredBookings(context.Background(), now); err != nil {
		t.Fatalf("ProcessExpiredBookings returned error: %v", err)
	}

	busyIntervals, err := svc.ListBusyIntervalsByResourceID(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListBusyIntervalsByResourceID returned error: %v", err)
	}
	if len(busyIntervals) != 2 {
		t.Fatalf("unexpected busy intervals count: got %d want 2", len(busyIntervals))
	}
	for _, interval := range busyIntervals {
		if !interval.StartAt.After(now) {
			t.Fatalf("expired booking remained in busy intervals: %+v", interval)
		}
	}
}

func TestBookingService_CompleteAt_TimeGuard(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)
	bookingID := int64(1)
	ownerID := int64(77)

	t.Run("manual complete before end_at is forbidden", func(t *testing.T) {
		t.Parallel()

		repo := &bookingRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
				return model.Booking{
					ID:         id,
					ResourceID: 10,
					UserID:     ownerID,
					Status:     model.BookingStatusConfirmed,
					StartAt:    now.Add(-time.Hour),
					EndAt:      now.Add(30 * time.Minute),
				}, nil
			},
		}

		svc := service.NewBookingService(repo, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})
		_, err := svc.CompleteAt(context.Background(), bookingID, ownerID, false, now)
		if !errors.Is(err, service.ErrBookingCompleteTooEarly) {
			t.Fatalf("expected ErrBookingCompleteTooEarly, got %v", err)
		}
	})

	t.Run("manual complete after end_at is allowed", func(t *testing.T) {
		t.Parallel()

		repo := &bookingRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
				return model.Booking{
					ID:         id,
					ResourceID: 10,
					UserID:     ownerID,
					Status:     model.BookingStatusConfirmed,
					StartAt:    now.Add(-2 * time.Hour),
					EndAt:      now.Add(-time.Minute),
				}, nil
			},
			transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
				if params.CompletedAt == nil || !params.CompletedAt.Equal(now) {
					t.Fatalf("unexpected completed_at: %#v", params.CompletedAt)
				}
				return model.Booking{
					ID:          id,
					ResourceID:  10,
					UserID:      ownerID,
					Status:      model.BookingStatusCompleted,
					StartAt:     now.Add(-2 * time.Hour),
					EndAt:       now.Add(-time.Minute),
					CompletedAt: params.CompletedAt,
					UpdatedAt:   now,
				}, model.BookingStatusConfirmed, nil
			},
		}

		svc := service.NewBookingService(repo, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})
		resp, err := svc.CompleteAt(context.Background(), bookingID, ownerID, false, now)
		if err != nil {
			t.Fatalf("CompleteAt returned error: %v", err)
		}
		if resp.Status != model.BookingStatusCompleted {
			t.Fatalf("unexpected status: %q", resp.Status)
		}
		if resp.CompletedAt == nil || !resp.CompletedAt.Equal(now) {
			t.Fatalf("unexpected completed_at in response: %#v", resp.CompletedAt)
		}
	})
}
