package service

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"

	"github.com/lib/pq"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/repository"
)

var (
	ErrValidation       = errors.New("validation error")
	ErrUserEmailExists  = errors.New("user email already exists")
	ErrRoleCodeNotFound = errors.New("role code not found")
)

type UserService struct {
	users  repository.UserRepository
	hasher auth.PasswordHasher
}

func NewUserService(users repository.UserRepository, hasher auth.PasswordHasher) *UserService {
	return &UserService{
		users:  users,
		hasher: hasher,
	}
}

func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest) (dto.UserResponse, error) {
	fullName := strings.TrimSpace(req.FullName)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)
	if fullName == "" || email == "" || password == "" || !isValidEmail(email) {
		return dto.UserResponse{}, ErrValidation
	}

	if err := s.users.ValidateRoleCodes(ctx, req.Roles); err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			return dto.UserResponse{}, ErrRoleCodeNotFound
		}
		return dto.UserResponse{}, err
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return dto.UserResponse{}, err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user, err := s.users.Create(ctx, repository.CreateUserParams{
		FullName:     fullName,
		Email:        email,
		PasswordHash: passwordHash,
		DepartmentID: req.DepartmentID,
		IsActive:     isActive,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return dto.UserResponse{}, ErrUserEmailExists
		}
		if isForeignKeyViolation(err) {
			return dto.UserResponse{}, ErrValidation
		}
		return dto.UserResponse{}, err
	}

	if err := s.users.ReplaceRolesByUserID(ctx, user.ID, req.Roles); err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			return dto.UserResponse{}, ErrRoleCodeNotFound
		}
		return dto.UserResponse{}, err
	}

	return s.GetByID(ctx, user.ID)
}

func (s *UserService) List(ctx context.Context) ([]dto.UserResponse, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		roles, err := s.users.ListRolesByUserID(ctx, user.ID)
		if err != nil {
			return nil, err
		}

		result = append(result, dto.UserResponse{
			ID:           user.ID,
			FullName:     user.FullName,
			Email:        user.Email,
			DepartmentID: user.DepartmentID,
			IsActive:     user.IsActive,
			Roles:        roles,
		})
	}

	return result, nil
}

func (s *UserService) GetByID(ctx context.Context, id int64) (dto.UserResponse, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.UserResponse{}, ErrUserNotFound
		}
		return dto.UserResponse{}, err
	}

	roles, err := s.users.ListRolesByUserID(ctx, user.ID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	return dto.UserResponse{
		ID:           user.ID,
		FullName:     user.FullName,
		Email:        user.Email,
		DepartmentID: user.DepartmentID,
		IsActive:     user.IsActive,
		Roles:        roles,
	}, nil
}

func (s *UserService) Update(ctx context.Context, id int64, req dto.UpdateUserRequest) (dto.UserResponse, error) {
	current, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.UserResponse{}, ErrUserNotFound
		}
		return dto.UserResponse{}, err
	}

	fullName := strings.TrimSpace(req.FullName)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if fullName == "" || email == "" || !isValidEmail(email) {
		return dto.UserResponse{}, ErrValidation
	}

	var passwordHash *string
	if strings.TrimSpace(req.Password) != "" {
		hash, err := s.hasher.Hash(strings.TrimSpace(req.Password))
		if err != nil {
			return dto.UserResponse{}, err
		}
		passwordHash = &hash
	}

	isActive := current.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	departmentID := current.DepartmentID
	if req.DepartmentID != nil {
		departmentID = req.DepartmentID
	}

	_, err = s.users.Update(ctx, id, repository.UpdateUserParams{
		FullName:     fullName,
		Email:        email,
		PasswordHash: passwordHash,
		DepartmentID: departmentID,
		IsActive:     isActive,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.UserResponse{}, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return dto.UserResponse{}, ErrUserEmailExists
		}
		if isForeignKeyViolation(err) {
			return dto.UserResponse{}, ErrValidation
		}
		return dto.UserResponse{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *UserService) ReplaceRoles(ctx context.Context, id int64, roleCodes []string) (dto.UserResponse, error) {
	if err := s.users.ValidateRoleCodes(ctx, roleCodes); err != nil {
		if errors.Is(err, repository.ErrRoleNotFound) {
			return dto.UserResponse{}, ErrRoleCodeNotFound
		}
		return dto.UserResponse{}, err
	}

	if err := s.users.ReplaceRolesByUserID(ctx, id, roleCodes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.UserResponse{}, ErrUserNotFound
		}
		if errors.Is(err, repository.ErrRoleNotFound) {
			return dto.UserResponse{}, ErrRoleCodeNotFound
		}
		return dto.UserResponse{}, err
	}

	return s.GetByID(ctx, id)
}

func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23503"
	}
	return false
}

func isValidEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Address, email)
}
