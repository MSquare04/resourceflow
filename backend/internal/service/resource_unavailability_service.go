package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
)

var (
	ErrResourceUnavailabilityNotFound = errors.New("resource unavailability not found")
	ErrUnavailabilityConflict         = errors.New("resource unavailability conflicts with existing active bookings")
)

type ResourceUnavailabilityService struct {
	unavailability repository.ResourceUnavailabilityRepository
	resources      repository.ResourceRepository
	bookings       repository.BookingRepository
}

func NewResourceUnavailabilityService(
	unavailability repository.ResourceUnavailabilityRepository,
	resources repository.ResourceRepository,
	bookings repository.BookingRepository,
) *ResourceUnavailabilityService {
	return &ResourceUnavailabilityService{
		unavailability: unavailability,
		resources:      resources,
		bookings:       bookings,
	}
}

func (s *ResourceUnavailabilityService) Create(
	ctx context.Context,
	resourceID int64,
	req dto.CreateResourceUnavailabilityRequest,
) (dto.ResourceUnavailabilityResponse, error) {
	startAt, endAt, err := validateAvailabilityRange(req.StartAt, req.EndAt)
	if err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}

	if err := s.ensureResourceCanManageAvailability(ctx, resourceID); err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}

	reason := trimNullableString(req.Reason)
	if err := s.ensureNoActiveBookingOverlap(ctx, resourceID, startAt, endAt); err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}

	unavailability, err := s.unavailability.Create(ctx, repository.CreateResourceUnavailabilityParams{
		ResourceID: resourceID,
		StartAt:    startAt,
		EndAt:      endAt,
		Reason:     reason,
	})
	if err != nil {
		if isForeignKeyViolation(err) || isCheckViolation(err) {
			return dto.ResourceUnavailabilityResponse{}, ErrUnavailabilityConflict
		}
		return dto.ResourceUnavailabilityResponse{}, err
	}

	return mapResourceUnavailabilityResponse(unavailability), nil
}

func (s *ResourceUnavailabilityService) ListByResourceID(
	ctx context.Context,
	resourceID int64,
) ([]dto.ResourceUnavailabilityResponse, error) {
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return nil, err
	}

	items, err := s.unavailability.ListByResourceID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ResourceUnavailabilityResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapResourceUnavailabilityResponse(item))
	}

	return result, nil
}

func (s *ResourceUnavailabilityService) GetByID(
	ctx context.Context,
	resourceID int64,
	id int64,
) (dto.ResourceUnavailabilityResponse, error) {
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}

	item, err := s.unavailability.FindByIDAndResourceID(ctx, resourceID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceUnavailabilityResponse{}, ErrResourceUnavailabilityNotFound
		}
		return dto.ResourceUnavailabilityResponse{}, err
	}

	return mapResourceUnavailabilityResponse(item), nil
}

func (s *ResourceUnavailabilityService) Update(
	ctx context.Context,
	resourceID int64,
	id int64,
	req dto.UpdateResourceUnavailabilityRequest,
) (dto.ResourceUnavailabilityResponse, error) {
	startAt, endAt, err := validateAvailabilityRange(req.StartAt, req.EndAt)
	if err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}

	if err := s.ensureResourceCanManageAvailability(ctx, resourceID); err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}
	if _, err := s.unavailability.FindByIDAndResourceID(ctx, resourceID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceUnavailabilityResponse{}, ErrResourceUnavailabilityNotFound
		}
		return dto.ResourceUnavailabilityResponse{}, err
	}
	if err := s.ensureNoActiveBookingOverlap(ctx, resourceID, startAt, endAt); err != nil {
		return dto.ResourceUnavailabilityResponse{}, err
	}

	item, err := s.unavailability.Update(ctx, resourceID, id, repository.UpdateResourceUnavailabilityParams{
		StartAt: startAt,
		EndAt:   endAt,
		Reason:  trimNullableString(req.Reason),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return dto.ResourceUnavailabilityResponse{}, ErrResourceUnavailabilityNotFound
		case isForeignKeyViolation(err), isCheckViolation(err):
			return dto.ResourceUnavailabilityResponse{}, ErrUnavailabilityConflict
		default:
			return dto.ResourceUnavailabilityResponse{}, err
		}
	}

	return mapResourceUnavailabilityResponse(item), nil
}

func (s *ResourceUnavailabilityService) Delete(ctx context.Context, resourceID int64, id int64) error {
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return err
	}

	deleted, err := s.unavailability.Delete(ctx, resourceID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrResourceUnavailabilityNotFound
	}

	return nil
}

func (s *ResourceUnavailabilityService) ensureNoActiveBookingOverlap(
	ctx context.Context,
	resourceID int64,
	startAt, endAt time.Time,
) error {
	hasConflict, err := s.bookings.HasConflict(ctx, resourceID, startAt, endAt, activeBookingStatuses)
	if err != nil {
		return err
	}
	if hasConflict {
		return ErrUnavailabilityConflict
	}
	return nil
}

func (s *ResourceUnavailabilityService) ensureResourceExists(ctx context.Context, resourceID int64) error {
	if resourceID <= 0 {
		return ErrValidation
	}

	_, err := s.resources.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrResourceNotFound
		}
		return err
	}

	return nil
}

func (s *ResourceUnavailabilityService) ensureResourceCanManageAvailability(ctx context.Context, resourceID int64) error {
	if resourceID <= 0 {
		return ErrValidation
	}

	resource, err := s.resources.FindByID(ctx, resourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrResourceNotFound
		}
		return err
	}

	if !resource.IsActive || !resource.IsBookable {
		return ErrUnavailabilityConflict
	}

	return nil
}

func mapResourceUnavailabilityResponse(item model.ResourceUnavailability) dto.ResourceUnavailabilityResponse {
	return dto.ResourceUnavailabilityResponse{
		ID:         item.ID,
		ResourceID: item.ResourceID,
		StartAt:    item.StartAt.UTC(),
		EndAt:      item.EndAt.UTC(),
		Reason:     trimNullableString(item.Reason),
		CreatedAt:  item.CreatedAt.UTC(),
		UpdatedAt:  item.UpdatedAt.UTC(),
	}
}

func validateAvailabilityRange(startAt, endAt time.Time) (time.Time, time.Time, error) {
	startUTC := startAt.UTC()
	endUTC := endAt.UTC()

	if startUTC.IsZero() || endUTC.IsZero() {
		return time.Time{}, time.Time{}, ErrValidation
	}

	if !startUTC.Before(endUTC) {
		return time.Time{}, time.Time{}, ErrValidation
	}

	return startUTC, endUTC, nil
}
