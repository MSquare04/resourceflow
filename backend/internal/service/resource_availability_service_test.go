package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
	"resourceflow/backend/internal/service"
)

func TestResourceAvailabilityService_ProtectsActiveBookings(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	resourceID := int64(10)
	activeBooking := model.Booking{
		ID:         100,
		ResourceID: resourceID,
		UserID:     7,
		Status:     model.BookingStatusConfirmed,
		StartAt:    now.Add(2 * time.Hour),
		EndAt:      now.Add(3 * time.Hour),
	}

	tests := []struct {
		name         string
		update       *dto.UpdateResourceAvailabilityRequest
		deleteID     int64
		availability []model.ResourceAvailability
		wantErr      error
	}{
		{
			name: "update that still covers booking",
			update: &dto.UpdateResourceAvailabilityRequest{
				StartAt: now.Add(time.Hour),
				EndAt:   now.Add(4 * time.Hour),
			},
			availability: []model.ResourceAvailability{
				{ID: 1, ResourceID: resourceID, StartAt: now.Add(time.Hour), EndAt: now.Add(4 * time.Hour)},
			},
		},
		{
			name: "update that breaks existing booking",
			update: &dto.UpdateResourceAvailabilityRequest{
				StartAt: now.Add(time.Hour),
				EndAt:   now.Add(150 * time.Minute),
			},
			availability: []model.ResourceAvailability{
				{ID: 1, ResourceID: resourceID, StartAt: now.Add(time.Hour), EndAt: now.Add(4 * time.Hour)},
			},
			wantErr: service.ErrAvailabilityConflict,
		},
		{
			name:     "delete last interval is allowed",
			deleteID: 1,
			availability: []model.ResourceAvailability{
				{ID: 1, ResourceID: resourceID, StartAt: now.Add(time.Hour), EndAt: now.Add(4 * time.Hour)},
			},
		},
		{
			name:     "delete interval when another interval still covers booking",
			deleteID: 1,
			availability: []model.ResourceAvailability{
				{ID: 1, ResourceID: resourceID, StartAt: now.Add(time.Hour), EndAt: now.Add(90 * time.Minute)},
				{ID: 2, ResourceID: resourceID, StartAt: now.Add(90 * time.Minute), EndAt: now.Add(4 * time.Hour)},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			availabilityRepo := &availabilityRepoMock{
				listByResourceIDFn: func(ctx context.Context, resourceID int64) ([]model.ResourceAvailability, error) {
					return tc.availability, nil
				},
				updateFn: func(ctx context.Context, resourceID int64, id int64, params repository.UpdateResourceAvailabilityParams) (model.ResourceAvailability, error) {
					return model.ResourceAvailability{ID: id, ResourceID: resourceID, StartAt: params.StartAt, EndAt: params.EndAt}, nil
				},
				deleteFn: func(ctx context.Context, resourceID int64, id int64) (bool, error) {
					return true, nil
				},
			}
			resourceRepo := &resourceRepoMock{
				findByIDFn: func(ctx context.Context, id int64) (model.Resource, error) {
					return model.Resource{ID: id, IsActive: true, IsBookable: true}, nil
				},
			}
			bookingRepo := &bookingRepoMock{
				listFn: func(ctx context.Context) ([]model.Booking, error) {
					return []model.Booking{activeBooking}, nil
				},
			}

			svc := service.NewResourceAvailabilityService(availabilityRepo, resourceRepo, bookingRepo)

			var err error
			if tc.update != nil {
				_, err = svc.Update(context.Background(), resourceID, 1, *tc.update)
			} else {
				err = svc.Delete(context.Background(), resourceID, tc.deleteID)
			}

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}
