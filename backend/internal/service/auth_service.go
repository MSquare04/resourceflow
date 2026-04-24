package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveUser       = errors.New("user is inactive")
	ErrUserNotFound       = errors.New("user not found")
)

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        dto.AuthUser
}

type AuthService struct {
	users  repository.UserRepository
	hasher auth.PasswordHasher
	tokens *auth.TokenManager
}

func NewAuthService(
	users repository.UserRepository,
	hasher auth.PasswordHasher,
	tokens *auth.TokenManager,
) *AuthService {
	return &AuthService{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))

	user, err := s.users.FindByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}

	if !user.IsActive {
		return LoginResult{}, ErrInactiveUser
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	roles, err := s.users.ListRolesByUserID(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}

	accessToken, expiresAt, err := s.tokens.GenerateAccessToken(user.ID, user.Email, roles)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		User: dto.AuthUser{
			ID:       user.ID,
			FullName: user.FullName,
			Email:    user.Email,
			IsActive: user.IsActive,
			Roles:    roles,
		},
	}, nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID int64) (dto.AuthUser, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dto.AuthUser{}, ErrUserNotFound
		}
		return dto.AuthUser{}, err
	}

	if !user.IsActive {
		return dto.AuthUser{}, ErrInactiveUser
	}

	roles, err := s.users.ListRolesByUserID(ctx, user.ID)
	if err != nil {
		return dto.AuthUser{}, err
	}

	return dto.AuthUser{
		ID:       user.ID,
		FullName: user.FullName,
		Email:    user.Email,
		IsActive: user.IsActive,
		Roles:    roles,
	}, nil
}
