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
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInactiveUser             = errors.New("user is inactive")
	ErrUserNotFound             = errors.New("user not found")
	ErrCurrentPasswordInvalid   = errors.New("current password is invalid")
	ErrNewPasswordSameAsCurrent = errors.New("new password is the same as current password")
	ErrPasswordPolicyViolation  = errors.New("password policy violation")
)

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        dto.AuthUser
}

type ChangePasswordResult struct{}

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

	accessToken, expiresAt, err := s.tokens.GenerateAccessToken(user.ID, user.Email, roles, user.AuthVersion)
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

func (s *AuthService) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) (ChangePasswordResult, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ChangePasswordResult{}, ErrUserNotFound
		}
		return ChangePasswordResult{}, err
	}

	if !user.IsActive {
		return ChangePasswordResult{}, ErrInactiveUser
	}

	if err := s.hasher.Compare(user.PasswordHash, currentPassword); err != nil {
		return ChangePasswordResult{}, ErrCurrentPasswordInvalid
	}

	if err := s.hasher.Compare(user.PasswordHash, newPassword); err == nil {
		return ChangePasswordResult{}, ErrNewPasswordSameAsCurrent
	}

	if !isValidPassword(newPassword) {
		return ChangePasswordResult{}, ErrPasswordPolicyViolation
	}

	passwordHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return ChangePasswordResult{}, err
	}

	if _, err := s.users.UpdatePassword(ctx, userID, repository.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ChangePasswordResult{}, ErrUserNotFound
		}
		return ChangePasswordResult{}, err
	}

	return ChangePasswordResult{}, nil
}

func isValidPassword(password string) bool {
	return strings.TrimSpace(password) != ""
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
