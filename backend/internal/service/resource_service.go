package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
)

var ErrResourceNotFound = errors.New("resource not found")

type ResourceService struct {
	resources repository.ResourceRepository
}

func NewResourceService(resources repository.ResourceRepository) *ResourceService {
	return &ResourceService{resources: resources}
}

func (s *ResourceService) Create(ctx context.Context, req dto.CreateResourceRequest) (dto.ResourceResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || req.CategoryID <= 0 || req.TypeID <= 0 {
		return dto.ResourceResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	location := trimNullableString(req.Location)
	capacity := trimCapacity(req.Capacity)
	if capacity != nil && *capacity < 0 {
		return dto.ResourceResponse{}, ErrValidation
	}

	isBookable := true
	if req.IsBookable != nil {
		isBookable = *req.IsBookable
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	resource, err := s.resources.Create(ctx, repository.CreateResourceParams{
		Name:         name,
		Description:  description,
		CategoryID:   req.CategoryID,
		TypeID:       req.TypeID,
		DepartmentID: req.DepartmentID,
		Location:     location,
		Capacity:     capacity,
		IsBookable:   isBookable,
		IsActive:     isActive,
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return dto.ResourceResponse{}, ErrValidation
		}
		return dto.ResourceResponse{}, err
	}

	return mapResourceResponse(resource), nil
}

func (s *ResourceService) List(ctx context.Context) ([]dto.ResourceResponse, error) {
	resources, err := s.resources.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ResourceResponse, 0, len(resources))
	for _, resource := range resources {
		result = append(result, mapResourceResponse(resource))
	}

	return result, nil
}

func (s *ResourceService) GetByID(ctx context.Context, id int64) (dto.ResourceResponse, error) {
	resource, err := s.resources.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceResponse{}, ErrResourceNotFound
		}
		return dto.ResourceResponse{}, err
	}

	return mapResourceResponse(resource), nil
}

func (s *ResourceService) Update(ctx context.Context, id int64, req dto.UpdateResourceRequest) (dto.ResourceResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || req.CategoryID <= 0 || req.TypeID <= 0 {
		return dto.ResourceResponse{}, ErrValidation
	}

	current, err := s.resources.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceResponse{}, ErrResourceNotFound
		}
		return dto.ResourceResponse{}, err
	}

	description := strings.TrimSpace(req.Description)
	location := trimNullableString(req.Location)
	capacity := trimCapacity(req.Capacity)
	if capacity != nil && *capacity < 0 {
		return dto.ResourceResponse{}, ErrValidation
	}

	isBookable := current.IsBookable
	if req.IsBookable != nil {
		isBookable = *req.IsBookable
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	resource, err := s.resources.Update(ctx, id, repository.UpdateResourceParams{
		Name:         name,
		Description:  description,
		CategoryID:   req.CategoryID,
		TypeID:       req.TypeID,
		DepartmentID: req.DepartmentID,
		Location:     location,
		Capacity:     capacity,
		IsBookable:   isBookable,
		IsActive:     isActive,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceResponse{}, ErrResourceNotFound
		}
		if isForeignKeyViolation(err) {
			return dto.ResourceResponse{}, ErrValidation
		}
		return dto.ResourceResponse{}, err
	}

	return mapResourceResponse(resource), nil
}

func mapResourceResponse(resource model.Resource) dto.ResourceResponse {
	return dto.ResourceResponse{
		ID:           resource.ID,
		Name:         resource.Name,
		Description:  resource.Description,
		CategoryID:   resource.CategoryID,
		TypeID:       resource.TypeID,
		DepartmentID: resource.DepartmentID,
		Location:     resource.Location,
		Capacity:     resource.Capacity,
		IsBookable:   resource.IsBookable,
		IsActive:     resource.IsActive,
	}
}

func trimNullableString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func trimCapacity(value *int64) *int64 {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
