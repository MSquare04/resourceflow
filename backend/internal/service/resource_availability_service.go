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
	ErrResourceAvailabilityNotFound = errors.New("resource availability not found")
)

type ResourceAvailabilityService struct {
	availability repository.ResourceAvailabilityRepository
	resources    repository.ResourceRepository
}

func NewResourceAvailabilityService(
	availability repository.ResourceAvailabilityRepository,
	resources repository.ResourceRepository,
) *ResourceAvailabilityService {
	return &ResourceAvailabilityService{
		availability: availability,
		resources:    resources,
	}
}

func (s *ResourceAvailabilityService) Create(ctx context.Context, resourceID int64, req dto.CreateResourceAvailabilityRequest) (dto.ResourceAvailabilityResponse, error) {
	startAt, endAt, err := validateAvailabilityRange(req.StartAt, req.EndAt)
	if err != nil {
		return dto.ResourceAvailabilityResponse{}, err
	}

	if err := s.ensureResourceCanManageAvailability(ctx, resourceID); err != nil {
		return dto.ResourceAvailabilityResponse{}, err
	}

	availability, err := s.availability.Create(ctx, repository.CreateResourceAvailabilityParams{
		ResourceID: resourceID,
		StartAt:    startAt,
		EndAt:      endAt,
	})
	if err != nil {
		if isForeignKeyViolation(err) || isCheckViolation(err) {
			return dto.ResourceAvailabilityResponse{}, ErrValidation
		}
		return dto.ResourceAvailabilityResponse{}, err
	}

	return mapResourceAvailabilityResponse(availability), nil
}

func (s *ResourceAvailabilityService) ListByResourceID(ctx context.Context, resourceID int64) ([]dto.ResourceAvailabilityResponse, error) {
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return nil, err
	}

	availabilityList, err := s.availability.ListByResourceID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ResourceAvailabilityResponse, 0, len(availabilityList))
	for _, availability := range availabilityList {
		result = append(result, mapResourceAvailabilityResponse(availability))
	}

	return result, nil
}

func (s *ResourceAvailabilityService) GetByID(ctx context.Context, resourceID int64, id int64) (dto.ResourceAvailabilityResponse, error) {
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return dto.ResourceAvailabilityResponse{}, err
	}

	availability, err := s.availability.FindByIDAndResourceID(ctx, resourceID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceAvailabilityResponse{}, ErrResourceAvailabilityNotFound
		}
		return dto.ResourceAvailabilityResponse{}, err
	}

	return mapResourceAvailabilityResponse(availability), nil
}

func (s *ResourceAvailabilityService) Update(ctx context.Context, resourceID int64, id int64, req dto.UpdateResourceAvailabilityRequest) (dto.ResourceAvailabilityResponse, error) {
	startAt, endAt, err := validateAvailabilityRange(req.StartAt, req.EndAt)
	if err != nil {
		return dto.ResourceAvailabilityResponse{}, err
	}

	if err := s.ensureResourceCanManageAvailability(ctx, resourceID); err != nil {
		return dto.ResourceAvailabilityResponse{}, err
	}

	availability, err := s.availability.Update(ctx, resourceID, id, repository.UpdateResourceAvailabilityParams{
		StartAt: startAt,
		EndAt:   endAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return dto.ResourceAvailabilityResponse{}, ErrResourceAvailabilityNotFound
		case isForeignKeyViolation(err), isCheckViolation(err):
			return dto.ResourceAvailabilityResponse{}, ErrValidation
		default:
			return dto.ResourceAvailabilityResponse{}, err
		}
	}

	return mapResourceAvailabilityResponse(availability), nil
}

func (s *ResourceAvailabilityService) Delete(ctx context.Context, resourceID int64, id int64) error {
	if err := s.ensureResourceExists(ctx, resourceID); err != nil {
		return err
	}

	deleted, err := s.availability.Delete(ctx, resourceID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrResourceAvailabilityNotFound
	}

	return nil
}

func (s *ResourceAvailabilityService) ensureResourceExists(ctx context.Context, resourceID int64) error {
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

func (s *ResourceAvailabilityService) ensureResourceCanManageAvailability(ctx context.Context, resourceID int64) error {
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
		return ErrValidation
	}

	return nil
}

func validateAvailabilityRange(startAt, endAt time.Time) (time.Time, time.Time, error) {
	if startAt.IsZero() || endAt.IsZero() || !startAt.Before(endAt) {
		return time.Time{}, time.Time{}, ErrValidation
	}
	return startAt.UTC(), endAt.UTC(), nil
}

func mapResourceAvailabilityResponse(availability model.ResourceAvailability) dto.ResourceAvailabilityResponse {
	return dto.ResourceAvailabilityResponse{
		ID:         availability.ID,
		ResourceID: availability.ResourceID,
		StartAt:    availability.StartAt.UTC(),
		EndAt:      availability.EndAt.UTC(),
		CreatedAt:  availability.CreatedAt.UTC(),
		UpdatedAt:  availability.UpdatedAt.UTC(),
	}
}
