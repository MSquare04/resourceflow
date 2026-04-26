package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"

	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/repository"
)

var (
	ErrDepartmentNotFound   = errors.New("department not found")
	ErrDepartmentNameExists = errors.New("department name already exists")
)

type DepartmentService struct {
	departments repository.DepartmentRepository
}

func NewDepartmentService(departments repository.DepartmentRepository) *DepartmentService {
	return &DepartmentService{departments: departments}
}

func (s *DepartmentService) Create(ctx context.Context, req dto.CreateDepartmentRequest) (dto.DepartmentResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return dto.DepartmentResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	department, err := s.departments.Create(ctx, name, description, isActive)
	if err != nil {
		if isUniqueViolation(err) {
			return dto.DepartmentResponse{}, ErrDepartmentNameExists
		}
		return dto.DepartmentResponse{}, err
	}

	return dto.DepartmentResponse{
		ID:          department.ID,
		Name:        department.Name,
		Description: department.Description,
		IsActive:    department.IsActive,
	}, nil
}

func (s *DepartmentService) List(ctx context.Context) ([]dto.DepartmentResponse, error) {
	departments, err := s.departments.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.DepartmentResponse, 0, len(departments))
	for _, department := range departments {
		result = append(result, dto.DepartmentResponse{
			ID:          department.ID,
			Name:        department.Name,
			Description: department.Description,
			IsActive:    department.IsActive,
		})
	}

	return result, nil
}

func (s *DepartmentService) GetByID(ctx context.Context, id int64) (dto.DepartmentResponse, error) {
	department, err := s.departments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.DepartmentResponse{}, ErrDepartmentNotFound
		}
		return dto.DepartmentResponse{}, err
	}

	return dto.DepartmentResponse{
		ID:          department.ID,
		Name:        department.Name,
		Description: department.Description,
		IsActive:    department.IsActive,
	}, nil
}

func (s *DepartmentService) Update(ctx context.Context, id int64, req dto.UpdateDepartmentRequest) (dto.DepartmentResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return dto.DepartmentResponse{}, ErrValidation
	}

	description := strings.TrimSpace(req.Description)
	current, err := s.departments.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.DepartmentResponse{}, ErrDepartmentNotFound
		}
		return dto.DepartmentResponse{}, err
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	department, err := s.departments.Update(ctx, id, name, description, isActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.DepartmentResponse{}, ErrDepartmentNotFound
		}
		if isUniqueViolation(err) {
			return dto.DepartmentResponse{}, ErrDepartmentNameExists
		}
		return dto.DepartmentResponse{}, err
	}

	return dto.DepartmentResponse{
		ID:          department.ID,
		Name:        department.Name,
		Description: department.Description,
		IsActive:    department.IsActive,
	}, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
