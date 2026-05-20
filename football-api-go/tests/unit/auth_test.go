package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

func loginRouter(svc services.AuthService) http.Handler {
	loginRL := middleware.NewLoginRateLimiter()
	h := handlers.NewAuthHandler(svc, loginRL)
	r := chi.NewRouter()
	r.Mount("/auth", h.PublicRoutes())
	return r
}

func postJSON(router http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── Login ────────────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, req services.LoginRequest) (*services.TokenResponse, error) {
			return &services.TokenResponse{
				AccessToken:  "access-token-abc",
				RefreshToken: "refresh-token-xyz",
				TokenType:    "bearer",
				PlayerID:     uuid.New().String(),
				Name:         "Zico",
				Role:         "player",
			}, nil
		},
	}

	w := postJSON(loginRouter(svc), "/auth/login",
		`{"whatsapp":"+5511999990000","password":"senha123"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var resp services.TokenResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "bearer", resp.TokenType)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "Zico", resp.Name)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ services.LoginRequest) (*services.TokenResponse, error) {
			return nil, apierror.Forbidden("invalid credentials")
		},
	}

	w := postJSON(loginRouter(svc), "/auth/login",
		`{"whatsapp":"+5511999990000","password":"wrong"}`)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invalid credentials")
}

func TestLogin_MalformedBody(t *testing.T) {
	svc := &mockAuthService{}
	w := postJSON(loginRouter(svc), "/auth/login", `{invalid json}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLogin_RateLimit(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ services.LoginRequest) (*services.TokenResponse, error) {
			return nil, apierror.Forbidden("invalid credentials")
		},
	}
	r := loginRouter(svc)

	// 5 allowed, 6th blocked
	for i := 0; i < 5; i++ {
		w := postJSON(r, "/auth/login", `{"whatsapp":"+5511999990000","password":"wrong"}`)
		assert.NotEqual(t, http.StatusTooManyRequests, w.Code, "attempt %d should not be rate-limited", i+1)
	}
	w := postJSON(r, "/auth/login", `{"whatsapp":"+5511999990000","password":"wrong"}`)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// ── Register ─────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, req services.RegisterRequest) (*services.TokenResponse, error) {
			return &services.TokenResponse{
				AccessToken: "tok",
				TokenType:   "bearer",
				PlayerID:    uuid.New().String(),
				Name:        req.Name,
				Role:        "player",
			}, nil
		},
	}

	w := postJSON(loginRouter(svc), "/auth/register",
		`{"name":"Romário","whatsapp":"+5511999990000","password":"senha123","otp_token":"otp-tok"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp services.TokenResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "Romário", resp.Name)
}

func TestRegister_Conflict(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _ services.RegisterRequest) (*services.TokenResponse, error) {
			return nil, apierror.Conflict("whatsapp already registered")
		},
	}

	w := postJSON(loginRouter(svc), "/auth/register",
		`{"name":"Zico","whatsapp":"+5511999990000","password":"senha123","otp_token":"tok"}`)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	svc := &mockAuthService{
		refreshTokenFn: func(_ context.Context, req services.RefreshRequest) (*services.RefreshResponse, error) {
			return &services.RefreshResponse{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
				TokenType:    "bearer",
			}, nil
		},
	}

	w := postJSON(loginRouter(svc), "/auth/refresh",
		`{"refresh_token":"old-token"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var resp services.RefreshResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "new-access", resp.AccessToken)
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := &mockAuthService{
		refreshTokenFn: func(_ context.Context, _ services.RefreshRequest) (*services.RefreshResponse, error) {
			return nil, apierror.Forbidden("invalid or expired refresh token")
		},
	}

	w := postJSON(loginRouter(svc), "/auth/refresh", `{"refresh_token":"bad"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── GetMe ─────────────────────────────────────────────────────────────────────

func TestGetMe_Success(t *testing.T) {
	player := fakePlayer()
	svc := &mockAuthService{
		getMeFn: func(_ context.Context, playerID uuid.UUID) (*services.PlayerResponse, error) {
			return &services.PlayerResponse{
				ID:   playerID.String(),
				Name: "Test Player",
				Role: "player",
			}, nil
		},
	}

	loginRL := middleware.NewLoginRateLimiter()
	h := handlers.NewAuthHandler(svc, loginRL)
	r := chi.NewRouter()

	// inject player into context (simulates middleware.Auth)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.InjectPlayerForTest(r.Context(), player)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Mount("/auth", h.ProtectedRoutes())

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer fake-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp services.PlayerResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "Test Player", resp.Name)
}

// ── SendOTP / VerifyOTP ───────────────────────────────────────────────────────

func TestSendOTP_Success(t *testing.T) {
	svc := &mockAuthService{
		sendOTPFn: func(_ context.Context, _ services.SendOTPRequest) (*services.SendOTPResponse, error) {
			return &services.SendOTPResponse{Status: "pending", ExpiresInSeconds: 600}, nil
		},
	}

	w := postJSON(loginRouter(svc), "/auth/send-otp", `{"whatsapp":"+5511999990000"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var resp services.SendOTPResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "pending", resp.Status)
}

func TestVerifyOTP_Success(t *testing.T) {
	svc := &mockAuthService{
		verifyOTPFn: func(_ context.Context, _ services.VerifyOTPRequest) (*services.VerifyOTPResponse, error) {
			return &services.VerifyOTPResponse{OTPToken: "otp-jwt-token"}, nil
		},
	}

	w := postJSON(loginRouter(svc), "/auth/verify-otp",
		`{"whatsapp":"+5511999990000","otp_code":"123456"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var resp services.VerifyOTPResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.OTPToken)
}

func TestVerifyOTP_InvalidCode(t *testing.T) {
	svc := &mockAuthService{
		verifyOTPFn: func(_ context.Context, _ services.VerifyOTPRequest) (*services.VerifyOTPResponse, error) {
			return nil, apierror.Forbidden("invalid OTP code")
		},
	}

	w := postJSON(loginRouter(svc), "/auth/verify-otp",
		`{"whatsapp":"+5511999990000","otp_code":"000000"}`)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
