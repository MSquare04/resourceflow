package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"resourceflow/backend/internal/auth"
	"resourceflow/backend/internal/dto"
	"resourceflow/backend/internal/handler"
	"resourceflow/backend/internal/middleware"
	"resourceflow/backend/internal/model"
	"resourceflow/backend/internal/repository"
	"resourceflow/backend/internal/service"
)

type authUserRepoMock struct {
	usersByEmail map[string]model.User
	usersByID    map[int64]model.User
	rolesByUser  map[int64][]string
}

func (m *authUserRepoMock) FindByEmail(ctx context.Context, email string) (model.User, error) {
	user, ok := m.usersByEmail[email]
	if !ok {
		return model.User{}, sql.ErrNoRows
	}
	return user, nil
}

func (m *authUserRepoMock) FindByID(ctx context.Context, id int64) (model.User, error) {
	user, ok := m.usersByID[id]
	if !ok {
		return model.User{}, sql.ErrNoRows
	}
	return user, nil
}

func (m *authUserRepoMock) Create(ctx context.Context, params repository.CreateUserParams) (model.User, error) {
	return model.User{}, sql.ErrNoRows
}

func (m *authUserRepoMock) List(ctx context.Context) ([]model.User, error) {
	return nil, nil
}

func (m *authUserRepoMock) Update(ctx context.Context, id int64, params repository.UpdateUserParams) (model.User, error) {
	return model.User{}, sql.ErrNoRows
}

func (m *authUserRepoMock) UpdatePassword(ctx context.Context, id int64, params repository.UpdateUserPasswordParams) (model.User, error) {
	user, ok := m.usersByID[id]
	if !ok {
		return model.User{}, sql.ErrNoRows
	}

	user.PasswordHash = params.PasswordHash
	user.AuthVersion++
	m.usersByID[id] = user
	m.usersByEmail[user.Email] = user
	return user, nil
}

func (m *authUserRepoMock) ListRolesByUserID(ctx context.Context, userID int64) ([]string, error) {
	return m.rolesByUser[userID], nil
}

func (m *authUserRepoMock) ValidateRoleCodes(ctx context.Context, roleCodes []string) error {
	return nil
}

func (m *authUserRepoMock) ReplaceRolesByUserID(ctx context.Context, userID int64, roleCodes []string) error {
	return nil
}

func TestAuthAccessSmoke(t *testing.T) {
	t.Parallel()

	hasher := auth.NewBcryptHasher()
	adminHash, err := hasher.Hash("secret123")
	if err != nil {
		t.Fatalf("failed to hash admin password: %v", err)
	}
	employeeHash, err := hasher.Hash("emp123")
	if err != nil {
		t.Fatalf("failed to hash employee password: %v", err)
	}

	repo := &authUserRepoMock{
		usersByEmail: map[string]model.User{
			"admin@example.com": {
				ID:           1,
				FullName:     "Admin User",
				Email:        "admin@example.com",
				PasswordHash: adminHash,
				AuthVersion:  1,
				IsActive:     true,
			},
			"employee@example.com": {
				ID:           2,
				FullName:     "Employee User",
				Email:        "employee@example.com",
				PasswordHash: employeeHash,
				AuthVersion:  1,
				IsActive:     true,
			},
		},
		usersByID: map[int64]model.User{
			1: {
				ID:           1,
				FullName:     "Admin User",
				Email:        "admin@example.com",
				PasswordHash: adminHash,
				AuthVersion:  1,
				IsActive:     true,
			},
			2: {
				ID:           2,
				FullName:     "Employee User",
				Email:        "employee@example.com",
				PasswordHash: employeeHash,
				AuthVersion:  1,
				IsActive:     true,
			},
		},
		rolesByUser: map[int64][]string{
			1: {"admin"},
			2: {"employee"},
		},
	}

	tokenManager := auth.NewTokenManager("test-secret", time.Hour)
	authService := service.NewAuthService(repo, hasher, tokenManager)
	authHandler := handler.NewAuthHandler(authService)
	authMiddleware := middleware.NewAuthMiddleware(tokenManager, repo)

	e := echo.New()
	e.POST("/auth/login", authHandler.Login)
	e.GET("/auth/me", authHandler.Me, authMiddleware.RequireAuth)
	e.POST("/auth/change-password", authHandler.ChangePassword, authMiddleware.RequireAuth)
	e.GET("/bookings", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, authMiddleware.RequireAuth, middleware.RequireRoles("admin", "manager"))
	e.GET("/admin-only", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, authMiddleware.RequireAuth, middleware.RequireRoles("admin"))

	t.Run("login success", func(t *testing.T) {
		rec := performJSONRequest(t, e, http.MethodPost, "/auth/login", map[string]any{
			"email":    "admin@example.com",
			"password": "secret123",
		}, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode login response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if data["access_token"] == "" {
			t.Fatalf("expected access_token in response")
		}
	})

	t.Run("login invalid credentials", func(t *testing.T) {
		rec := performJSONRequest(t, e, http.MethodPost, "/auth/login", map[string]any{
			"email":    "admin@example.com",
			"password": "wrong",
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("auth me with token", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(1, "admin@example.com", []string{"admin"}, repo.usersByID[1].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/auth/me", nil, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("auth me without token", func(t *testing.T) {
		rec := performJSONRequest(t, e, http.MethodGet, "/auth/me", nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin endpoint without auth", func(t *testing.T) {
		rec := performJSONRequest(t, e, http.MethodGet, "/admin-only", nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin endpoint without role", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(2, "employee@example.com", []string{"employee"}, repo.usersByID[2].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/admin-only", nil, token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("employee cannot access all bookings route", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(2, "employee@example.com", []string{"employee"}, repo.usersByID[2].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/bookings", nil, token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin can access all bookings route", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(1, "admin@example.com", []string{"admin"}, repo.usersByID[1].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/bookings", nil, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("inactive user token is rejected", func(t *testing.T) {
		repo.usersByID[2] = model.User{
			ID:           2,
			FullName:     "Employee User",
			Email:        "employee@example.com",
			PasswordHash: employeeHash,
			AuthVersion:  1,
			IsActive:     false,
		}
		repo.usersByEmail["employee@example.com"] = repo.usersByID[2]

		token, _, err := tokenManager.GenerateAccessToken(2, "employee@example.com", []string{"employee"}, 1)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/auth/me", nil, token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}

		repo.usersByID[2] = model.User{
			ID:           2,
			FullName:     "Employee User",
			Email:        "employee@example.com",
			PasswordHash: employeeHash,
			AuthVersion:  1,
			IsActive:     true,
		}
		repo.usersByEmail["employee@example.com"] = repo.usersByID[2]
	})

	t.Run("stale admin claim does not bypass current roles", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(2, "employee@example.com", []string{"admin"}, repo.usersByID[2].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/admin-only", nil, token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("change password success invalidates old token and old password", func(t *testing.T) {
		loginResponse := performJSONRequest(t, e, http.MethodPost, "/auth/login", map[string]any{
			"email":    "admin@example.com",
			"password": "secret123",
		}, "")
		if loginResponse.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", loginResponse.Code, loginResponse.Body.String())
		}

		var loginPayload struct {
			Data dto.LoginResponse `json:"data"`
		}
		if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginPayload); err != nil {
			t.Fatalf("failed to decode login payload: %v", err)
		}
		oldToken := loginPayload.Data.AccessToken

		changePasswordResponse := performJSONRequest(t, e, http.MethodPost, "/auth/change-password", map[string]any{
			"current_password": "secret123",
			"new_password":     "new-secret-123",
		}, oldToken)
		if changePasswordResponse.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", changePasswordResponse.Code, changePasswordResponse.Body.String())
		}

		oldPasswordLogin := performJSONRequest(t, e, http.MethodPost, "/auth/login", map[string]any{
			"email":    "admin@example.com",
			"password": "secret123",
		}, "")
		if oldPasswordLogin.Code != http.StatusUnauthorized {
			t.Fatalf("expected old password login to fail with 401, got %d body=%s", oldPasswordLogin.Code, oldPasswordLogin.Body.String())
		}

		staleTokenResponse := performJSONRequest(t, e, http.MethodGet, "/auth/me", nil, oldToken)
		if staleTokenResponse.Code != http.StatusUnauthorized {
			t.Fatalf("expected stale token to fail with 401, got %d body=%s", staleTokenResponse.Code, staleTokenResponse.Body.String())
		}

		newPasswordLogin := performJSONRequest(t, e, http.MethodPost, "/auth/login", map[string]any{
			"email":    "admin@example.com",
			"password": "new-secret-123",
		}, "")
		if newPasswordLogin.Code != http.StatusOK {
			t.Fatalf("expected new password login to succeed, got %d body=%s", newPasswordLogin.Code, newPasswordLogin.Body.String())
		}

		var newLoginPayload struct {
			Data dto.LoginResponse `json:"data"`
		}
		if err := json.Unmarshal(newPasswordLogin.Body.Bytes(), &newLoginPayload); err != nil {
			t.Fatalf("failed to decode new login payload: %v", err)
		}

		newTokenResponse := performJSONRequest(t, e, http.MethodGet, "/auth/me", nil, newLoginPayload.Data.AccessToken)
		if newTokenResponse.Code != http.StatusOK {
			t.Fatalf("expected new token to work, got %d body=%s", newTokenResponse.Code, newTokenResponse.Body.String())
		}
	})

	t.Run("change password invalid current password", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(1, "admin@example.com", []string{"admin"}, repo.usersByID[1].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		rec := performJSONRequest(t, e, http.MethodPost, "/auth/change-password", map[string]any{
			"current_password": "wrong-password",
			"new_password":     "another-secret",
		}, token)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec, dto.ErrorCodeCurrentPasswordInvalid)
	})

	t.Run("change password same as current", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(1, "admin@example.com", []string{"admin"}, repo.usersByID[1].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		rec := performJSONRequest(t, e, http.MethodPost, "/auth/change-password", map[string]any{
			"current_password": "new-secret-123",
			"new_password":     "new-secret-123",
		}, token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec, dto.ErrorCodeNewPasswordSameAsCurrent)
	})

	t.Run("change password policy violation", func(t *testing.T) {
		token, _, err := tokenManager.GenerateAccessToken(1, "admin@example.com", []string{"admin"}, repo.usersByID[1].AuthVersion)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		rec := performJSONRequest(t, e, http.MethodPost, "/auth/change-password", map[string]any{
			"current_password": "new-secret-123",
			"new_password":     "   ",
		}, token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec, dto.ErrorCodePasswordPolicyViolation)
	})

	t.Run("change password without auth", func(t *testing.T) {
		rec := performJSONRequest(t, e, http.MethodPost, "/auth/change-password", map[string]any{
			"current_password": "new-secret-123",
			"new_password":     "final-secret-123",
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()

	var payload dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if payload.Error.Code != expectedCode {
		t.Fatalf("unexpected error code: got %q want %q body=%s", payload.Error.Code, expectedCode, rec.Body.String())
	}
}

func performJSONRequest(t *testing.T, e *echo.Echo, method, path string, payload any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal request payload: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
