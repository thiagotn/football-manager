package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/config"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/server"
)

const testOTPBypassCode = "123456"

var (
	poolOnce sync.Once
	pool     *pgxpool.Pool
	poolErr  error
)

// getPool returns the shared test pool. Skips if DATABASE_URL is not set.
func getPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	poolOnce.Do(func() {
		pool, poolErr = db.NewPool(dbURL)
	})
	if poolErr != nil {
		t.Skipf("DB unavailable: %v", poolErr)
	}
	return pool
}

// testConfig builds a minimal config suitable for integration tests.
func testConfig() *config.Config {
	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		secretKey = "test-secret-key-ci-only-not-production"
	}
	return &config.Config{
		AppEnv:      "test",
		Port:        8080,
		SecretKey:   secretKey,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		CORSOrigins: "http://localhost:3000",
		FrontendURL: "http://localhost:3000",
		// OTP bypass: skip Twilio calls in tests
		OTPBypassCode: testOTPBypassCode,
		// Set high expire times so tokens don't expire mid-test
		AccessTokenExpireMinutes:  60,
		InviteTokenExpireMinutes:  30,
		// Fake Stripe config so webhook signature validation is active in tests.
		// Tests use "whsec_test_secret_12345" when computing valid HMACs.
		StripeSecretKey:     "sk_test_fake_ci_only",
		StripeWebhookSecret: "whsec_test_secret_12345",
	}
}

// newTestServer builds a test HTTP server backed by the real router + test pool.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	p := getPool(t)
	srv := httptest.NewServer(server.NewRouter(testConfig(), p))
	t.Cleanup(srv.Close)
	return srv
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// response wraps an HTTP status code and a decoded JSON body.
// Body is non-nil for object responses (map), List is non-nil for array responses.
type response struct {
	Code int
	Body map[string]any // object response (most endpoints)
	List []any          // array response (list endpoints like GET /groups)
}

func apiCall(t *testing.T, srv *httptest.Server, method, path, token string, payload any) response {
	t.Helper()
	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewBuffer(bodyBytes)) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var raw interface{}
	_ = json.NewDecoder(resp.Body).Decode(&raw)

	result := response{Code: resp.StatusCode}
	switch v := raw.(type) {
	case map[string]any:
		result.Body = v
	case []interface{}:
		result.List = v
	}
	return result
}

// ── Auth helpers ──────────────────────────────────────────────────────────────

type testPlayer struct {
	ID       string
	WhatsApp string
	Password string
	Token    string
}

// registerAndLogin creates a new player via the API and returns the player with token.
// Uses a unique whatsapp number derived from a random suffix.
func registerAndLogin(t *testing.T, srv *httptest.Server, name string) testPlayer {
	t.Helper()
	whatsapp := "+551199" + fmt.Sprintf("%07d", uuid.New().ID()%9999999)
	password := "senha123"

	// 1. Send OTP
	r := apiCall(t, srv, http.MethodPost, "/api/v2/auth/send-otp", "",
		map[string]string{"whatsapp": whatsapp})
	require.Equal(t, http.StatusOK, r.Code, "send-otp failed: %v", r.Body)

	// 2. Verify OTP using bypass code → get otp_token
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/verify-otp", "",
		map[string]string{"whatsapp": whatsapp, "otp_code": testOTPBypassCode})
	require.Equal(t, http.StatusOK, r.Code, "verify-otp failed: %v", r.Body)
	otpToken, _ := r.Body["otp_token"].(string)
	require.NotEmpty(t, otpToken, "otp_token must be present")

	// 3. Register
	r = apiCall(t, srv, http.MethodPost, "/api/v2/auth/register", "",
		map[string]string{"name": name, "whatsapp": whatsapp, "password": password, "otp_token": otpToken})
	require.Equal(t, http.StatusCreated, r.Code, "register failed: %v", r.Body)
	playerID, _ := r.Body["player_id"].(string)
	accessToken, _ := r.Body["access_token"].(string)
	require.NotEmpty(t, accessToken)

	p := testPlayer{ID: playerID, WhatsApp: whatsapp, Password: password, Token: accessToken}

	// Cleanup: delete player after test
	pool := getPool(t)
	t.Cleanup(func() {
		if p.ID != "" {
			id, err := uuid.Parse(p.ID)
			if err == nil {
				_, _ = pool.Exec(context.Background(),
					`DELETE FROM players WHERE id=$1`, id)
			}
		}
	})
	return p
}

// registerGroupCleanup schedules deletion of the group and its records at end of test.
func registerGroupCleanup(t *testing.T, groupIDStr string) {
	t.Helper()
	pool := getPool(t)
	t.Cleanup(func() {
		id, err := uuid.Parse(groupIDStr)
		if err == nil {
			_, _ = pool.Exec(context.Background(), `DELETE FROM groups WHERE id=$1`, id)
		}
	})
}

// makeAdmin promotes a player to admin role in the database.
func makeAdmin(t *testing.T, playerIDStr string) {
	t.Helper()
	pool := getPool(t)
	playerID, err := uuid.Parse(playerIDStr)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`UPDATE players SET role='admin' WHERE id=$1`, playerID)
	require.NoError(t, err)
}
