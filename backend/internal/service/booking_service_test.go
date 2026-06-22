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

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)
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

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)

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

	t.Run("outside workday returns error", func(t *testing.T) {
		bookings := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) { return 0, nil },
		}
		rules := &bookingRuleRepoMock{
			findActiveByResourceTypeIDFn: func(ctx context.Context, resourceTypeID int64) (model.BookingRule, error) {
				workdayStart, _ := time.Parse("15:04", "09:00")
				workdayEnd, _ := time.Parse("15:04", "18:00")
				return model.BookingRule{
					ResourceTypeID:           resourceTypeID,
					MinDurationMinutes:       10,
					MaxDurationMinutes:       240,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					WorkdayStart:             workdayStart,
					WorkdayEnd:               workdayEnd,
					IsActive:                 true,
				}, nil
			},
		}
		svc := service.NewBookingService(bookings, baseResource, baseUsers, rules)
		req := baseReq
		req.StartAt = time.Date(now.Year(), now.Month(), now.Day()+1, 8, 0, 0, 0, time.UTC)
		req.EndAt = req.StartAt.Add(time.Hour)

		_, err := svc.CreateAt(context.Background(), 55, req, now)
		if !errors.Is(err, service.ErrBookingOutsideWorkday) {
			t.Fatalf("expected ErrBookingOutsideWorkday, got %v", err)
		}
	})

	t.Run("booking without unavailability is allowed", func(t *testing.T) {
		bookings := &bookingRepoMock{
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
		name              string
		req               dto.CreateBookingRequest
		resource          model.Resource
		conflict          bool
		hasUnavailability bool
		wantErr           error
	}{
		{
			name:     "success",
			req:      baseReq,
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
		},
		{
			name:     "start in past",
			req:      dto.CreateBookingRequest{ResourceID: 10, StartAt: now.Truncate(time.Minute).Add(-time.Minute), EndAt: now.Add(time.Hour)},
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
			wantErr:  service.ErrBookingStartNotFuture,
		},
		{
			name:              "resource unavailability conflict",
			req:               baseReq,
			resource:          model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
			hasUnavailability: true,
			wantErr:           service.ErrBookingInUnavailability,
		},
		{
			name:     "conflict with active booking",
			req:      baseReq,
			resource: model.Resource{ID: 10, TypeID: 20, IsActive: true, IsBookable: true},
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

			svc := service.NewBookingService(bookings, resources, baseUsers, baseRule).WithUnavailabilityChecker(&unavailabilityCheckerMock{
				hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
					return tc.hasUnavailability, nil
				},
			})
			_, err := svc.CreateAt(context.Background(), 55, tc.req, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestBookingService_CreateAt_WorkdayAndUnavailabilityRules(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 8, 0, 0, 0, time.UTC)
	workdayStart, _ := time.Parse("15:04", "09:00")
	workdayEnd, _ := time.Parse("15:04", "18:00")

	newService := func(
		t *testing.T,
		checker func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error),
	) *service.BookingService {
		t.Helper()

		bookings := &bookingRepoMock{
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
					MinDurationMinutes:       60,
					MaxDurationMinutes:       180,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					WorkdayStart:             workdayStart,
					WorkdayEnd:               workdayEnd,
					IsActive:                 true,
				}, nil
			},
		}

		return service.NewBookingService(bookings, resources, users, rules).WithUnavailabilityChecker(&unavailabilityCheckerMock{
			hasConflictFn: checker,
		})
	}

	tests := []struct {
		name    string
		req     dto.CreateBookingRequest
		checker func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error)
		wantErr error
	}{
		{
			name: "inside workday is allowed",
			req: dto.CreateBookingRequest{
				ResourceID: 10,
				StartAt:    time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC),
				EndAt:      time.Date(2026, time.June, 22, 11, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "before workday start is rejected",
			req: dto.CreateBookingRequest{
				ResourceID: 10,
				StartAt:    time.Date(2026, time.June, 22, 8, 30, 0, 0, time.UTC),
				EndAt:      time.Date(2026, time.June, 22, 9, 30, 0, 0, time.UTC),
			},
			wantErr: service.ErrBookingOutsideWorkday,
		},
		{
			name: "after workday end is rejected",
			req: dto.CreateBookingRequest{
				ResourceID: 10,
				StartAt:    time.Date(2026, time.June, 22, 17, 30, 0, 0, time.UTC),
				EndAt:      time.Date(2026, time.June, 22, 18, 30, 0, 0, time.UTC),
			},
			wantErr: service.ErrBookingOutsideWorkday,
		},
		{
			name: "technical interval overlap is rejected",
			req: dto.CreateBookingRequest{
				ResourceID: 10,
				StartAt:    time.Date(2026, time.June, 22, 13, 30, 0, 0, time.UTC),
				EndAt:      time.Date(2026, time.June, 22, 14, 30, 0, 0, time.UTC),
			},
			checker: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
				return true, nil
			},
			wantErr: service.ErrBookingInUnavailability,
		},
		{
			name: "adjacent ranges are allowed",
			req: dto.CreateBookingRequest{
				ResourceID: 10,
				StartAt:    time.Date(2026, time.June, 22, 15, 0, 0, 0, time.UTC),
				EndAt:      time.Date(2026, time.June, 22, 16, 0, 0, 0, time.UTC),
			},
			checker: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
				return false, nil
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := newService(t, tc.checker)
			_, err := svc.CreateAt(context.Background(), 77, tc.req, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}

	t.Run("unrestricted time ignores workday bounds", func(t *testing.T) {
		t.Parallel()

		bookings := &bookingRepoMock{
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) { return 0, nil },
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time, statuses []string) (bool, error) {
				return false, nil
			},
			createFn: func(ctx context.Context, params repository.CreateBookingParams) (model.Booking, error) {
				return model.Booking{
					ID:         2,
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
					MinDurationMinutes:       60,
					MaxDurationMinutes:       180,
					MaxActiveBookingsPerUser: 3,
					BookingHorizonDays:       30,
					WorkdayStart:             workdayStart,
					WorkdayEnd:               workdayEnd,
					UnrestrictedTime:         true,
					IsActive:                 true,
				}, nil
			},
		}

		svc := service.NewBookingService(bookings, resources, users, rules)
		_, err := svc.CreateAt(context.Background(), 77, dto.CreateBookingRequest{
			ResourceID: 10,
			StartAt:    time.Date(2026, time.June, 22, 8, 0, 0, 0, time.UTC),
			EndAt:      time.Date(2026, time.June, 22, 9, 0, 0, 0, time.UTC),
		}, now)
		if err != nil {
			t.Fatalf("expected unrestricted booking to succeed, got %v", err)
		}
	})
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
			countByStatusesFn: func(ctx context.Context, userID int64, statuses []string) (int64, error) { return 0, nil },
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

		svc := service.NewBookingService(bookings, resources, users, rules).WithUnavailabilityChecker(&unavailabilityCheckerMock{
			hasConflictFn: func(ctx context.Context, resourceID int64, startAt, endAt time.Time) (bool, error) {
				return true, nil
			},
		})
		req := dto.CreateBookingRequest{
			ResourceID: 10,
			StartAt:    currentMinute,
			EndAt:      currentMinute.Add(30 * time.Minute),
		}

		_, err := svc.CreateAt(context.Background(), 55, req, now)
		if !errors.Is(err, service.ErrBookingInUnavailability) {
			t.Fatalf("expected ErrBookingInUnavailability, got %v", err)
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

func TestBookingService_ListBusyIntervalsByResourceIDInRange(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, time.June, 22, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)

	t.Run("passes explicit range and preserves overlapping intervals", func(t *testing.T) {
		bookings := &bookingRepoMock{
			listBusyFn: func(ctx context.Context, resourceID int64, statuses []string, gotFrom, gotTo time.Time) ([]model.Booking, error) {
				if resourceID != 10 {
					t.Fatalf("unexpected resource id: %d", resourceID)
				}
				if !gotFrom.Equal(from) || !gotTo.Equal(to) {
					t.Fatalf("unexpected range: %s..%s", gotFrom, gotTo)
				}

				return []model.Booking{
					{StartAt: from.Add(-time.Hour), EndAt: from.Add(time.Hour)},
					{StartAt: to.Add(-time.Hour), EndAt: to.Add(time.Hour)},
				}, nil
			},
		}
		resources := &resourceRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
				return model.Resource{ID: id, Name: "Room 1"}, nil
			},
		}

		svc := service.NewBookingService(bookings, resources, &userRepoMock{}, &bookingRuleRepoMock{})
		got, err := svc.ListBusyIntervalsByResourceIDInRange(context.Background(), 10, &from, &to)
		if err != nil {
			t.Fatalf("ListBusyIntervalsByResourceIDInRange returned error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("unexpected intervals count: got %d want 2", len(got))
		}
		if !got[0].StartAt.Equal(from.Add(-time.Hour)) || !got[1].EndAt.Equal(to.Add(time.Hour)) {
			t.Fatalf("unexpected intervals payload: %#v", got)
		}
	})

	t.Run("rejects incomplete range", func(t *testing.T) {
		resources := &resourceRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
				return model.Resource{ID: id, Name: "Room 1"}, nil
			},
		}
		svc := service.NewBookingService(&bookingRepoMock{}, resources, &userRepoMock{}, &bookingRuleRepoMock{})
		_, err := svc.ListBusyIntervalsByResourceIDInRange(context.Background(), 10, &from, nil)
		if !errors.Is(err, service.ErrBusyIntervalRangeInvalid) {
			t.Fatalf("expected ErrBusyIntervalRangeInvalid, got %v", err)
		}
	})

	t.Run("rejects non-positive range", func(t *testing.T) {
		resources := &resourceRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
				return model.Resource{ID: id, Name: "Room 1"}, nil
			},
		}
		svc := service.NewBookingService(&bookingRepoMock{}, resources, &userRepoMock{}, &bookingRuleRepoMock{})
		_, err := svc.ListBusyIntervalsByResourceIDInRange(context.Background(), 10, &to, &from)
		if !errors.Is(err, service.ErrBusyIntervalRangeInvalid) {
			t.Fatalf("expected ErrBusyIntervalRangeInvalid, got %v", err)
		}
	})

	t.Run("rejects range longer than 31 days", func(t *testing.T) {
		longTo := from.Add(31*24*time.Hour + time.Minute)
		resources := &resourceRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
				return model.Resource{ID: id, Name: "Room 1"}, nil
			},
		}
		svc := service.NewBookingService(&bookingRepoMock{}, resources, &userRepoMock{}, &bookingRuleRepoMock{})
		_, err := svc.ListBusyIntervalsByResourceIDInRange(context.Background(), 10, &from, &longTo)
		if !errors.Is(err, service.ErrBusyIntervalRangeInvalid) {
			t.Fatalf("expected ErrBusyIntervalRangeInvalid, got %v", err)
		}
	})
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
		{name: "cancel after end_at forbidden", action: "cancel", fromStatus: model.BookingStatusConfirmed, actorUserID: 77, bookingUserID: 77, wantErr: service.ErrBookingAlreadyEnded},
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
					if tc.action == "cancel" && tc.wantErr == service.ErrBookingAlreadyEnded {
						endAt = now
					}
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
					if tc.action == "cancel" && tc.wantErr == service.ErrBookingAlreadyEnded {
						endAt = now
					}
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
				resp, err = svc.CancelAt(context.Background(), bookingID, actorUserID, tc.isPrivileged, now)
			case "complete":
				resp, err = svc.CompleteAt(context.Background(), bookingID, actorUserID, tc.isPrivileged, now)
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
				resp, err := svc.CancelAt(context.Background(), 1, 77, false, now)
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
	if len(busyIntervals) != 1 {
		t.Fatalf("unexpected busy intervals count: got %d want 1", len(busyIntervals))
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

func TestBookingService_CancelAt_PrivilegedAndOwnershipRules(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)

	t.Run("employee cannot cancel foreign booking", func(t *testing.T) {
		repo := &bookingRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
				return model.Booking{
					ID:         id,
					ResourceID: 10,
					UserID:     99,
					Status:     model.BookingStatusConfirmed,
					StartAt:    now.Add(time.Hour),
					EndAt:      now.Add(2 * time.Hour),
				}, nil
			},
		}

		svc := service.NewBookingService(repo, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})
		_, err := svc.CancelAt(context.Background(), 1, 77, false, now)
		if !errors.Is(err, service.ErrBookingForbidden) {
			t.Fatalf("expected ErrBookingForbidden, got %v", err)
		}
	})

	t.Run("manager can cancel foreign confirmed booking", func(t *testing.T) {
		repo := &bookingRepoMock{
			findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
				return model.Booking{
					ID:         id,
					ResourceID: 10,
					UserID:     99,
					Status:     model.BookingStatusConfirmed,
					StartAt:    now.Add(time.Hour),
					EndAt:      now.Add(2 * time.Hour),
				}, nil
			},
			transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
				if params.CancelledAt == nil || !params.CancelledAt.Equal(now) {
					t.Fatalf("unexpected cancelled_at: %#v", params.CancelledAt)
				}
				return model.Booking{
					ID:          id,
					ResourceID:  10,
					UserID:      99,
					Status:      model.BookingStatusCancelled,
					StartAt:     now.Add(time.Hour),
					EndAt:       now.Add(2 * time.Hour),
					CancelledAt: params.CancelledAt,
					UpdatedAt:   now,
				}, model.BookingStatusConfirmed, nil
			},
		}

		svc := service.NewBookingService(repo, &resourceRepoMock{}, &userRepoMock{}, &bookingRuleRepoMock{})
		resp, err := svc.CancelAt(context.Background(), 1, 77, true, now)
		if err != nil {
			t.Fatalf("CancelAt returned error: %v", err)
		}
		if resp.Status != model.BookingStatusCancelled {
			t.Fatalf("unexpected status: %q", resp.Status)
		}
		if resp.CancelledAt == nil || !resp.CancelledAt.Equal(now) {
			t.Fatalf("unexpected cancelled_at in response: %#v", resp.CancelledAt)
		}
	})
}

func TestBookingService_CancelAt_RemovesBookingFromBusyIntervals(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 22, 12, 0, 0, 0, time.UTC)
	bookingsState := []model.Booking{
		{ID: 1, ResourceID: 10, UserID: 77, Status: model.BookingStatusConfirmed, StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)},
		{ID: 2, ResourceID: 10, UserID: 77, Status: model.BookingStatusPending, StartAt: now.Add(3 * time.Hour), EndAt: now.Add(4 * time.Hour)},
	}

	repo := &bookingRepoMock{
		findByIDFn: func(ctx context.Context, id int64) (model.Booking, error) {
			for _, booking := range bookingsState {
				if booking.ID == id {
					return booking, nil
				}
			}
			return model.Booking{}, sql.ErrNoRows
		},
		transitionStatusFn: func(ctx context.Context, id int64, expectedFrom []string, params repository.UpdateBookingStatusParams) (model.Booking, string, error) {
			for index := range bookingsState {
				if bookingsState[index].ID != id {
					continue
				}
				bookingsState[index].Status = params.Status
				bookingsState[index].CancelledAt = params.CancelledAt
				bookingsState[index].UpdatedAt = now
				return bookingsState[index], model.BookingStatusConfirmed, nil
			}
			return model.Booking{}, "", sql.ErrNoRows
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
				if !active || !booking.EndAt.After(from) || !booking.StartAt.Before(until) {
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

	svc := service.NewBookingService(repo, resources, &userRepoMock{}, &bookingRuleRepoMock{})
	if _, err := svc.CancelAt(context.Background(), 1, 77, false, now); err != nil {
		t.Fatalf("CancelAt returned error: %v", err)
	}

	busyIntervals, err := svc.ListBusyIntervalsByResourceID(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListBusyIntervalsByResourceID returned error: %v", err)
	}
	if len(busyIntervals) != 1 {
		t.Fatalf("unexpected busy intervals count: got %d want 1", len(busyIntervals))
	}
	if !busyIntervals[0].StartAt.Equal(now.Add(3 * time.Hour)) {
		t.Fatalf("cancelled booking remained in busy intervals: %+v", busyIntervals)
	}
}
