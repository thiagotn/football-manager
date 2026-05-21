package unit_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// doRequest fires an arbitrary HTTP method with an optional JSON body.
func doRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

// fakePlayer builds a Player for use in tests.
func fakePlayer(opts ...func(*db.Player)) *db.Player {
	p := &db.Player{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:         "Test Player",
		WhatsApp:     "+5511999990000",
		PasswordHash: "$2b$12$fake-hash",
		Role:         db.PlayerRolePlayer,
		Active:       true,
		ApiV2Enabled: true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func asAdmin() func(*db.Player) {
	return func(p *db.Player) { p.Role = db.PlayerRoleAdmin }
}

func withApiV2Disabled() func(*db.Player) {
	return func(p *db.Player) { p.ApiV2Enabled = false }
}

// mockAuthService implements services.AuthService for testing.
type mockAuthService struct {
	loginFn                 func(ctx context.Context, req services.LoginRequest) (*services.TokenResponse, error)
	sendOTPFn               func(ctx context.Context, req services.SendOTPRequest) (*services.SendOTPResponse, error)
	verifyOTPFn             func(ctx context.Context, req services.VerifyOTPRequest) (*services.VerifyOTPResponse, error)
	registerFn              func(ctx context.Context, req services.RegisterRequest) (*services.TokenResponse, error)
	getMeFn                 func(ctx context.Context, playerID uuid.UUID) (*services.PlayerResponse, error)
	forgotPasswordSendOTPFn func(ctx context.Context, req services.SendOTPRequest) (*services.SendOTPResponse, error)
	forgotPasswordVerifyFn  func(ctx context.Context, req services.VerifyOTPRequest) (*services.VerifyOTPResponse, error)
	forgotPasswordResetFn   func(ctx context.Context, req services.ForgotPasswordResetRequest) error
	sendOTPMeFn             func(ctx context.Context, playerID uuid.UUID) (*services.SendOTPResponse, error)
	verifyOTPMeFn           func(ctx context.Context, playerID uuid.UUID, req services.VerifyOTPMeRequest) (*services.VerifyOTPResponse, error)
	changePasswordFn        func(ctx context.Context, playerID uuid.UUID, req services.ChangePasswordRequest) error
	refreshTokenFn          func(ctx context.Context, req services.RefreshRequest) (*services.RefreshResponse, error)
}

func (m *mockAuthService) Login(ctx context.Context, req services.LoginRequest) (*services.TokenResponse, error) {
	return m.loginFn(ctx, req)
}
func (m *mockAuthService) SendOTP(ctx context.Context, req services.SendOTPRequest) (*services.SendOTPResponse, error) {
	return m.sendOTPFn(ctx, req)
}
func (m *mockAuthService) VerifyOTP(ctx context.Context, req services.VerifyOTPRequest) (*services.VerifyOTPResponse, error) {
	return m.verifyOTPFn(ctx, req)
}
func (m *mockAuthService) Register(ctx context.Context, req services.RegisterRequest) (*services.TokenResponse, error) {
	return m.registerFn(ctx, req)
}
func (m *mockAuthService) GetMe(ctx context.Context, playerID uuid.UUID) (*services.PlayerResponse, error) {
	return m.getMeFn(ctx, playerID)
}
func (m *mockAuthService) ForgotPasswordSendOTP(ctx context.Context, req services.SendOTPRequest) (*services.SendOTPResponse, error) {
	return m.forgotPasswordSendOTPFn(ctx, req)
}
func (m *mockAuthService) ForgotPasswordVerifyOTP(ctx context.Context, req services.VerifyOTPRequest) (*services.VerifyOTPResponse, error) {
	return m.forgotPasswordVerifyFn(ctx, req)
}
func (m *mockAuthService) ForgotPasswordReset(ctx context.Context, req services.ForgotPasswordResetRequest) error {
	return m.forgotPasswordResetFn(ctx, req)
}
func (m *mockAuthService) SendOTPMe(ctx context.Context, playerID uuid.UUID) (*services.SendOTPResponse, error) {
	return m.sendOTPMeFn(ctx, playerID)
}
func (m *mockAuthService) VerifyOTPMe(ctx context.Context, playerID uuid.UUID, req services.VerifyOTPMeRequest) (*services.VerifyOTPResponse, error) {
	return m.verifyOTPMeFn(ctx, playerID, req)
}
func (m *mockAuthService) ChangePassword(ctx context.Context, playerID uuid.UUID, req services.ChangePasswordRequest) error {
	return m.changePasswordFn(ctx, playerID, req)
}
func (m *mockAuthService) RefreshToken(ctx context.Context, req services.RefreshRequest) (*services.RefreshResponse, error) {
	return m.refreshTokenFn(ctx, req)
}
