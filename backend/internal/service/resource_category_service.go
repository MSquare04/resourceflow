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
	ErrResourceCategoryNotFound   = errors.New("resource category not found")
	ErrResourceCategoryCodeExists = errors.New("resource category code already exists")
)

type ResourceCategoryService struct {
	categories repository.ResourceCategoryRepository
}

func NewResourceCategoryService(categories repository.ResourceCategoryRepository) *ResourceCategoryService {
	return &ResourceCategoryService{categories: categories}
}

func (s *ResourceCategoryService) Create(ctx context.Context, req dto.CreateResourceCategoryRequest) (dto.ResourceCategoryResponse, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return dto.ResourceCategoryResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	category, err := s.categories.Create(ctx, code, name, description, isActive)
	if err != nil {
		if isUniqueViolation(err) {
			return dto.ResourceCategoryResponse{}, ErrResourceCategoryCodeExists
		}
		return dto.ResourceCategoryResponse{}, err
	}

	return dto.ResourceCategoryResponse{
		ID:          category.ID,
		Code:        category.Code,
		Name:        category.Name,
		Description: category.Description,
		IsActive:    category.IsActive,
	}, nil
}

func (s *ResourceCategoryService) List(ctx context.Context) ([]dto.ResourceCategoryResponse, error) {
	categories, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ResourceCategoryResponse, 0, len(categories))
	for _, category := range categories {
		result = append(result, dto.ResourceCategoryResponse{
			ID:          category.ID,
			Code:        category.Code,
			Name:        category.Name,
			Description: category.Description,
			IsActive:    category.IsActive,
		})
	}

	return result, nil
}

func (s *ResourceCategoryService) GetByID(ctx context.Context, id int64) (dto.ResourceCategoryResponse, error) {
	category, err := s.categories.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceCategoryResponse{}, ErrResourceCategoryNotFound
		}
		return dto.ResourceCategoryResponse{}, err
	}

	return dto.ResourceCategoryResponse{
		ID:          category.ID,
		Code:        category.Code,
		Name:        category.Name,
		Description: category.Description,
		IsActive:    category.IsActive,
	}, nil
}

func (s *ResourceCategoryService) Update(ctx context.Context, id int64, req dto.UpdateResourceCategoryRequest) (dto.ResourceCategoryResponse, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return dto.ResourceCategoryResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	current, err := s.categories.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceCategoryResponse{}, ErrResourceCategoryNotFound
		}
		return dto.ResourceCategoryResponse{}, err
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	category, err := s.categories.Update(ctx, id, code, name, description, isActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.ResourceCategoryResponse{}, ErrResourceCategoryNotFound
		}
		if isUniqueViolation(err) {
			return dto.ResourceCategoryResponse{}, ErrResourceCategoryCodeExists
		}
		return dto.ResourceCategoryResponse{}, err
	}

	return dto.ResourceCategoryResponse{
		ID:          category.ID,
		Code:        category.Code,
		Name:        category.Name,
		Description: category.Description,
		IsActive:    category.IsActive,
	}, nil
}
