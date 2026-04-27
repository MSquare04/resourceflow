package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/service"
)

func TestResourceService_Create_InvalidTypeCategoryRelation(t *testing.T) {
	t.Parallel()

	resources := &resourceRepoMock{}
	resourceTypes := &resourceTypeRepoMock{
		existsByIDAndCategory: func(ctx context.Context, id int64, categoryID int64) (bool, error) {
			return false, nil
		},
	}

	svc := service.NewResourceService(resources, resourceTypes)
	_, err := svc.Create(context.Background(), dto.CreateResourceRequest{
		Name:       "Meeting Room A",
		CategoryID: 1,
		TypeID:     100,
	})
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestResourceAvailabilityService_Create_InvalidInterval(t *testing.T) {
	t.Parallel()

	svc := service.NewResourceAvailabilityService(&availabilityRepoMock{}, &resourceRepoMock{})

	now := time.Now().UTC().Truncate(time.Second)
	_, err := svc.Create(context.Background(), 1, dto.CreateResourceAvailabilityRequest{
		StartAt: now,
		EndAt:   now,
	})
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestBookingRuleService_Create_InvalidPayload(t *testing.T) {
	t.Parallel()

	svc := service.NewBookingRuleService(&bookingRuleRepoMock{}, &resourceTypeRepoMock{})

	_, err := svc.Create(context.Background(), dto.CreateBookingRuleRequest{
		ResourceTypeID:           1,
		MinDurationMinutes:       60,
		MaxDurationMinutes:       30,
		MaxActiveBookingsPerUser: 1,
		BookingHorizonDays:       7,
	})
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
