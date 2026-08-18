package unit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// ── Fake ClaimStore ───────────────────────────────────────────────────────────

type fakeClaimStore struct {
	claimInvite   *db.ClaimInvite
	playerByWA    *db.Player
	playerByID    *db.Player
	claimedWA     string
	claimErr      error
	usedToken     string
	revokedPlayer *uuid.UUID
}

func (s *fakeClaimStore) GetClaimInviteByToken(ctx context.Context, token string) (*db.ClaimInvite, error) {
	if s.claimInvite == nil {
		return nil, db.ErrNotFound
	}
	return s.claimInvite, nil
}

func (s *fakeClaimStore) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	if s.playerByWA == nil {
		return nil, db.ErrNotFound
	}
	return s.playerByWA, nil
}

func (s *fakeClaimStore) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	if s.playerByID == nil {
		return nil, db.ErrNotFound
	}
	return s.playerByID, nil
}

func (s *fakeClaimStore) ClaimPlayerRegistration(ctx context.Context, id uuid.UUID, whatsapp, hash string) error {
	if s.claimErr != nil {
		return s.claimErr
	}
	s.claimedWA = whatsapp
	return nil
}

func (s *fakeClaimStore) UseInvite(ctx context.Context, token string, playerID uuid.UUID) error {
	s.usedToken = token
	return nil
}

func (s *fakeClaimStore) RevokeAllRefreshTokensForPlayer(ctx context.Context, playerID uuid.UUID) error {
	s.revokedPlayer = &playerID
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func makeClaimInvite(targetID uuid.UUID, pending bool) *db.ClaimInvite {
	inv := &db.ClaimInvite{}
	inv.ID = uuid.New()
	inv.GroupID = uuid.New()
	inv.Token = "claimtoken123"
	inv.Purpose = db.InvitePurposeRegistrationClaim
	inv.TargetPlayerID = &targetID
	inv.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	inv.GroupName = "Rachão da Firma"
	inv.TargetPlayerName = "Fulano de Tal"
	inv.TargetPlayerPending = pending
	return inv
}

func claimsRouter(store handlers.ClaimStore, authSvc services.AuthService) chi.Router {
	h := handlers.NewClaimHandlerWithDeps(store, authSvc)
	r := chi.NewRouter()
	r.Mount("/claims", h.Routes())
	return r
}

func doJSON(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── GET /claims/{token} ───────────────────────────────────────────────────────

func TestGetClaimInfo_OK(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true)}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodGet, "/claims/claimtoken123", "")

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["valid"])
	assert.Equal(t, "Fulano", resp["player_first_name"])
	assert.Equal(t, "Rachão da Firma", resp["group_name"])
}

func TestGetClaimInfo_NotFound(t *testing.T) {
	store := &fakeClaimStore{}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodGet, "/claims/naoexiste", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetClaimInfo_Used(t *testing.T) {
	targetID := uuid.New()
	inv := makeClaimInvite(targetID, true)
	inv.Used = true
	store := &fakeClaimStore{claimInvite: inv}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodGet, "/claims/claimtoken123", "")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "INVITE_USED")
}

func TestGetClaimInfo_Expired(t *testing.T) {
	targetID := uuid.New()
	inv := makeClaimInvite(targetID, true)
	inv.ExpiresAt = time.Now().Add(-time.Hour)
	store := &fakeClaimStore{claimInvite: inv}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodGet, "/claims/claimtoken123", "")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "INVITE_EXPIRED")
}

func TestGetClaimInfo_AlreadyClaimed(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, false)}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodGet, "/claims/claimtoken123", "")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "ALREADY_CLAIMED")
}

// ── POST /claims/{token}/send-otp ─────────────────────────────────────────────

func TestClaimSendOTP_WhatsAppTakenByOtherPlayer(t *testing.T) {
	targetID := uuid.New()
	other := &db.Player{ID: uuid.New(), WhatsApp: "+5511988887777"}
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true), playerByWA: other}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/send-otp",
		`{"whatsapp": "+5511988887777"}`)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "WHATSAPP_TAKEN")
}

func TestClaimSendOTP_AllowsOwnCurrentNumber(t *testing.T) {
	targetID := uuid.New()
	self := &db.Player{ID: targetID, WhatsApp: "+5511999991234"}
	sent := ""
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true), playerByWA: self}
	authSvc := &mockAuthService{sendOTPToFn: func(wa string) error { sent = wa; return nil }}
	r := claimsRouter(store, authSvc)

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/send-otp",
		`{"whatsapp": "+5511999991234"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "+5511999991234", sent)
}

func TestClaimSendOTP_InvalidWhatsAppFormat(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true)}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/send-otp",
		`{"whatsapp": "+0000"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── POST /claims/{token}/verify-otp ───────────────────────────────────────────

func TestClaimVerifyOTP_InvalidCode(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true)}
	authSvc := &mockAuthService{verifyOTPForFn: func(wa, code string) (string, error) {
		return "", assert.AnError
	}}
	r := claimsRouter(store, authSvc)

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/verify-otp",
		`{"whatsapp": "+5511999991234", "otp_code": "000000"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "OTP_INVALID")
}

func TestClaimVerifyOTP_Success(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true)}
	authSvc := &mockAuthService{verifyOTPForFn: func(wa, code string) (string, error) {
		return "otp-jwt-assinado", nil
	}}
	r := claimsRouter(store, authSvc)

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/verify-otp",
		`{"whatsapp": "+5511999991234", "otp_code": "123456"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "otp-jwt-assinado")
}

// ── POST /claims/{token}/complete ─────────────────────────────────────────────

func TestClaimComplete_Success(t *testing.T) {
	targetID := uuid.New()
	claimed := &db.Player{ID: targetID, Name: "Fulano de Tal", WhatsApp: "+5511999991234", Role: db.PlayerRolePlayer}
	store := &fakeClaimStore{
		claimInvite: makeClaimInvite(targetID, true),
		playerByID:  claimed,
	}
	authSvc := &mockAuthService{
		decodeOTPFn: func(token string) (string, error) { return "+5511999991234", nil },
		issueTokenPairForPlayerFn: func(ctx context.Context, p *db.Player) (*services.TokenResponse, error) {
			return &services.TokenResponse{AccessToken: "jwt-acesso", PlayerID: p.ID.String(), Name: p.Name}, nil
		},
	}
	r := claimsRouter(store, authSvc)

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/complete",
		`{"whatsapp": "+5511999991234", "otp_token": "otp-jwt", "password": "senhaNova123"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "jwt-acesso")
	assert.Equal(t, "+5511999991234", store.claimedWA)
	assert.Equal(t, "claimtoken123", store.usedToken)
	require.NotNil(t, store.revokedPlayer)
	assert.Equal(t, targetID, *store.revokedPlayer)
}

func TestClaimComplete_InvalidOTPToken(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true)}
	authSvc := &mockAuthService{
		decodeOTPFn: func(token string) (string, error) { return "", assert.AnError },
	}
	r := claimsRouter(store, authSvc)

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/complete",
		`{"whatsapp": "+5511999991234", "otp_token": "invalido", "password": "senhaNova123"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "OTP_TOKEN_INVALID")
}

func TestClaimComplete_WhatsAppTaken(t *testing.T) {
	targetID := uuid.New()
	other := &db.Player{ID: uuid.New(), WhatsApp: "+5511999991234"}
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true), playerByWA: other}
	authSvc := &mockAuthService{
		decodeOTPFn: func(token string) (string, error) { return "+5511999991234", nil },
	}
	r := claimsRouter(store, authSvc)

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/complete",
		`{"whatsapp": "+5511999991234", "otp_token": "otp-jwt", "password": "senhaNova123"}`)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "WHATSAPP_TAKEN")
}

func TestClaimComplete_ShortPassword(t *testing.T) {
	targetID := uuid.New()
	store := &fakeClaimStore{claimInvite: makeClaimInvite(targetID, true)}
	r := claimsRouter(store, &mockAuthService{})

	w := doJSON(t, r, http.MethodPost, "/claims/claimtoken123/complete",
		`{"whatsapp": "+5511999991234", "otp_token": "otp-jwt", "password": "123"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
