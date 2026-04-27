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
				IsActive:     true,
			},
			"employee@example.com": {
				ID:           2,
				FullName:     "Employee User",
				Email:        "employee@example.com",
				PasswordHash: employeeHash,
				IsActive:     true,
			},
		},
		usersByID: map[int64]model.User{
			1: {
				ID:           1,
				FullName:     "Admin User",
				Email:        "admin@example.com",
				PasswordHash: adminHash,
				IsActive:     true,
			},
			2: {
				ID:           2,
				FullName:     "Employee User",
				Email:        "employee@example.com",
				PasswordHash: employeeHash,
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
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	e := echo.New()
	e.POST("/auth/login", authHandler.Login)
	e.GET("/auth/me", authHandler.Me, authMiddleware.RequireAuth)
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
		token, _, err := tokenManager.GenerateAccessToken(1, "admin@example.com", []string{"admin"})
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
		token, _, err := tokenManager.GenerateAccessToken(2, "employee@example.com", []string{"employee"})
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		rec := performJSONRequest(t, e, http.MethodGet, "/admin-only", nil, token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
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
