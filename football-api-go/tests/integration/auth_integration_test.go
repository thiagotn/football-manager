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

	// GET /auth/me with token
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
