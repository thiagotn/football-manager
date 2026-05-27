package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_RegisterLoginMe(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Zico Integration")
	enableApiV2(t, player.ID)

	// GET /auth/me with token (requires api_v2_enabled=true)
	r := apiCall(t, srv, http.MethodGet, "/api/v2/auth/me", player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "Zico Integration", r.Body["name"])
	assert.Equal(t, "player", r.Body["role"])
}

func TestAuth_Login_InvalidCredentials(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Ronaldo Integration")

	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/login", "",
		map[string]string{"whatsapp": player.WhatsApp, "password": "wrong_password"})
	assert.Equal(t, http.StatusForbidden, r.Code)
}

func TestAuth_Refresh(t *testing.T) {
	srv := newTestServer(t)

	// Send OTP + verify to get otp_token, then register to get refresh_token
	player := registerAndLogin(t, srv, "Refresh Test Player")

	// Login again to get a refresh token (register doesn't return refresh)
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/login", "",
		map[string]string{"whatsapp": player.WhatsApp, "password": player.Password})
	require.Equal(t, http.StatusOK, r.Code)
	refreshToken, _ := r.Body["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)

	// Use refresh token to get a new access token
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/refresh", "",
		map[string]string{"refresh_token": refreshToken})
	require.Equal(t, http.StatusOK, r.Code)
	newAccess, _ := r.Body["access_token"].(string)
	assert.NotEmpty(t, newAccess)
}

func TestAuth_Me_Unauthenticated(t *testing.T) {
	srv := newTestServer(t)
	r := apiCall(t, srv, http.MethodGet, "/api/v2/auth/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, r.Code)
}

func TestAuth_Me_InvalidToken(t *testing.T) {
	srv := newTestServer(t)
	r := apiCall(t, srv, http.MethodGet, "/api/v2/auth/me", "not.a.valid.token", nil)
	assert.Equal(t, http.StatusUnauthorized, r.Code)
}

func TestAuth_Register_DuplicateWhatsApp(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Duplicate Test")

	// Try to register again with the same whatsapp
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/send-otp", "",
		map[string]string{"whatsapp": player.WhatsApp})
	require.Equal(t, http.StatusOK, r.Code)

	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/verify-otp", "",
		map[string]string{"whatsapp": player.WhatsApp, "otp_code": testOTPBypassCode})
	require.Equal(t, http.StatusOK, r.Code)
	otpToken, _ := r.Body["otp_token"].(string)

	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/register", "",
		map[string]any{"name": "Another", "whatsapp": player.WhatsApp, "password": "senha123", "otp_token": otpToken})
	assert.Equal(t, http.StatusConflict, r.Code)
}

func TestAuth_Health(t *testing.T) {
	srv := newTestServer(t)
	r := apiCall(t, srv, http.MethodGet, "/api/v2/health", "", nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "ok", r.Body["status"])
}

// ── Forgot Password ────────────────────────────────────────────────────────

func TestAuth_ForgotPasswordFlow(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Forgot Password Test")

	// Step 1: Send OTP to forgot-password endpoint
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/forgot-password/send-otp", "",
		map[string]string{"whatsapp": player.WhatsApp})
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "pending", r.Body["status"])

	// Step 2: Verify OTP to get forgot-password token
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/forgot-password/verify-otp", "",
		map[string]string{"whatsapp": player.WhatsApp, "otp_code": testOTPBypassCode})
	require.Equal(t, http.StatusOK, r.Code)
	forgotToken, ok := r.Body["otp_token"].(string)
	require.True(t, ok, "otp_token not found in response: %v", r.Body)
	require.NotEmpty(t, forgotToken)

	// Step 3: Reset password using forgot-password token
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/forgot-password/reset", "",
		map[string]any{"whatsapp": player.WhatsApp, "otp_token": forgotToken, "new_password": "new_password_123"})
	require.Equal(t, http.StatusNoContent, r.Code, "response: %v", r.Body)

	// Step 4: Verify can login with new password
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/login", "",
		map[string]string{"whatsapp": player.WhatsApp, "password": "new_password_123"})
	require.Equal(t, http.StatusOK, r.Code)
	assert.NotEmpty(t, r.Body["access_token"])
}

func TestAuth_ForgotPassword_InvalidWhatsApp(t *testing.T) {
	srv := newTestServer(t)

	// Try forgot password with non-existent whatsapp
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/forgot-password/send-otp", "",
		map[string]string{"whatsapp": "+5500000000000"})
	// Should still return 200 for security reasons (not leak if user exists)
	assert.Equal(t, http.StatusOK, r.Code)
}

func TestAuth_ForgotPassword_InvalidOTP(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "OTP Test")

	// Send OTP
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/forgot-password/send-otp", "",
		map[string]string{"whatsapp": player.WhatsApp})
	require.Equal(t, http.StatusOK, r.Code)

	// Verify with wrong OTP code
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/forgot-password/verify-otp", "",
		map[string]string{"whatsapp": player.WhatsApp, "otp_code": "000000"})
	assert.Equal(t, http.StatusForbidden, r.Code)
}

// ── OTP Me (Change Phone Number) ───────────────────────────────────────────

func TestAuth_SendOTPMe(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "SendOTPMe Test")
	enableApiV2(t, player.ID) // API v2 endpoints require this flag

	// Send OTP for current authenticated user
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/send-otp/me", player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code, "response: %v", r.Body)
	assert.Equal(t, "pending", r.Body["status"])
}

func TestAuth_VerifyOTPMe(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "VerifyOTPMe Test")
	enableApiV2(t, player.ID) // API v2 endpoints require this flag

	// Send OTP first
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/send-otp/me", player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)

	// Verify OTP
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/verify-otp/me", player.Token,
		map[string]string{"otp_code": testOTPBypassCode})
	require.Equal(t, http.StatusOK, r.Code)
	// OTP verification success (returns a token but doesn't change the account)
	assert.NotEmpty(t, r.Body["otp_token"])
}

// ── Change Password ────────────────────────────────────────────────────────

func TestAuth_ChangePassword(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Change Password Test")
	enableApiV2(t, player.ID) // API v2 endpoints require this flag

	// Change password with correct current password
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/change-password", player.Token,
		map[string]string{"current_password": player.Password, "new_password": "new_password_456"})
	require.Equal(t, http.StatusNoContent, r.Code)

	// Verify can't login with old password
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/login", "",
		map[string]string{"whatsapp": player.WhatsApp, "password": player.Password})
	assert.Equal(t, http.StatusForbidden, r.Code)

	// Verify can login with new password
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/login", "",
		map[string]string{"whatsapp": player.WhatsApp, "password": "new_password_456"})
	require.Equal(t, http.StatusOK, r.Code)
}

func TestAuth_ChangePassword_WrongCurrentPassword(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Wrong Password Test")
	enableApiV2(t, player.ID) // API v2 endpoints require this flag

	// Try to change password with incorrect current password
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/change-password", player.Token,
		map[string]string{"current_password": "wrong_password", "new_password": "new_password_789"})
	assert.Equal(t, http.StatusForbidden, r.Code)
}

func TestAuth_ChangePassword_Unauthenticated(t *testing.T) {
	srv := newTestServer(t)

	// Try to change password without token
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/change-password", "",
		map[string]string{"current_password": "any", "new_password": "new"})
	assert.Equal(t, http.StatusUnauthorized, r.Code)
}
