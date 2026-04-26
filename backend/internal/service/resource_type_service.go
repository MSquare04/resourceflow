package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/repository"
)

var (
	ErrResourceTypeNotFound   = errors.New("resource type not found")
	ErrResourceTypeCodeExists = errors.New("resource type code already exists")
)

type ResourceTypeService struct {
	resourceTypes repository.ResourceTypeRepository
}

func NewResourceTypeService(resourceTypes repository.ResourceTypeRepository) *ResourceTypeService {
	return &ResourceTypeService{resourceTypes: resourceTypes}
}

func (s *ResourceTypeService) Create(ctx context.Context, req dto.CreateResourceTypeRequest) (dto.ResourceTypeResponse, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if req.CategoryID <= 0 || code == "" || name == "" {
		return dto.ResourceTypeResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	resourceType, err := s.resourceTypes.Create(ctx, req.CategoryID, code, name, description, isActive)
	if err != nil {
		if isUniqueViolation(err) {
			return dto.ResourceTypeResponse{}, ErrResourceTypeCodeExists
		}
		if isForeignKeyViolation(err) {
			return dto.ResourceTypeResponse{}, ErrValidation
		}
		return dto.ResourceTypeResponse{}, err
	}

	return dto.ResourceTypeResponse{
		ID:          resourceType.ID,
		CategoryID:  resourceType.CategoryID,
		Code:        resourceType.Code,
		Name:        resourceType.Name,
		Description: resourceType.Description,
		IsActive:    resourceType.IsActive,
	}, nil
}

func (s *ResourceTypeService) List(ctx context.Context) ([]dto.ResourceTypeResponse, error) {
	resourceTypes, err := s.resourceTypes.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ResourceTypeResponse, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		result = append(result, dto.ResourceTypeResponse{
			ID:          resourceType.ID,
			CategoryID:  resourceType.CategoryID,
			Code:        resourceType.Code,
			Name:        resourceType.Name,
			Description: resourceType.Description,
			IsActive:    resourceType.IsActive,
		})
	}

	return result, nil
}

func (s *ResourceTypeService) GetByID(ctx context.Context, id int64) (dto.ResourceTypeResponse, error) {
	resourceType, err := s.resourceTypes.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceTypeResponse{}, ErrResourceTypeNotFound
		}
		return dto.ResourceTypeResponse{}, err
	}

	return dto.ResourceTypeResponse{
		ID:          resourceType.ID,
		CategoryID:  resourceType.CategoryID,
		Code:        resourceType.Code,
		Name:        resourceType.Name,
		Description: resourceType.Description,
		IsActive:    resourceType.IsActive,
	}, nil
}

func (s *ResourceTypeService) Update(ctx context.Context, id int64, req dto.UpdateResourceTypeRequest) (dto.ResourceTypeResponse, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if req.CategoryID <= 0 || code == "" || name == "" {
		return dto.ResourceTypeResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	current, err := s.resourceTypes.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceTypeResponse{}, ErrResourceTypeNotFound
		}
		return dto.ResourceTypeResponse{}, err
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	resourceType, err := s.resourceTypes.Update(ctx, id, req.CategoryID, code, name, description, isActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceTypeResponse{}, ErrResourceTypeNotFound
		}
		if isUniqueViolation(err) {
			return dto.ResourceTypeResponse{}, ErrResourceTypeCodeExists
		}
		if isForeignKeyViolation(err) {
			return dto.ResourceTypeResponse{}, ErrValidation
		}
		return dto.ResourceTypeResponse{}, err
	}

	return dto.ResourceTypeResponse{
		ID:          resourceType.ID,
		CategoryID:  resourceType.CategoryID,
		Code:        resourceType.Code,
		Name:        resourceType.Name,
		Description: resourceType.Description,
		IsActive:    resourceType.IsActive,
	}, nil
}
