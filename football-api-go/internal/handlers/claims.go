package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// ClaimStore is the data access surface for the registration-claim flow.
type ClaimStore interface {
	GetClaimInviteByToken(ctx context.Context, token string) (*db.ClaimInvite, error)
	GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error)
	GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error)
	ClaimPlayerRegistration(ctx context.Context, id uuid.UUID, whatsapp, hash string) error
	UseInvite(ctx context.Context, token string, playerID uuid.UUID) error
	RevokeAllRefreshTokensForPlayer(ctx context.Context, playerID uuid.UUID) error
}

type pgClaimStore struct {
	pool *pgxpool.Pool
}

func (s *pgClaimStore) GetClaimInviteByToken(ctx context.Context, token string) (*db.ClaimInvite, error) {
	return db.GetClaimInviteByToken(ctx, s.pool, token)
}

func (s *pgClaimStore) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	return db.GetPlayerByWhatsApp(ctx, s.pool, whatsapp)
}

func (s *pgClaimStore) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	return db.GetPlayerByID(ctx, s.pool, playerID)
}

func (s *pgClaimStore) ClaimPlayerRegistration(ctx context.Context, id uuid.UUID, whatsapp, hash string) error {
	return db.ClaimPlayerRegistration(ctx, s.pool, id, whatsapp, hash)
}

func (s *pgClaimStore) UseInvite(ctx context.Context, token string, playerID uuid.UUID) error {
	return db.UseInvite(ctx, s.pool, token, playerID)
}

func (s *pgClaimStore) RevokeAllRefreshTokensForPlayer(ctx context.Context, playerID uuid.UUID) error {
	return db.RevokeAllRefreshTokensForPlayer(ctx, s.pool, playerID)
}

type ClaimHandler struct {
	Store   ClaimStore
	authSvc services.AuthService
}

func NewClaimHandler(pool *pgxpool.Pool, authSvc services.AuthService) *ClaimHandler {
	return &ClaimHandler{Store: &pgClaimStore{pool: pool}, authSvc: authSvc}
}

// NewClaimHandlerWithDeps lets tests inject the Store and AuthService directly.
func NewClaimHandlerWithDeps(store ClaimStore, authSvc services.AuthService) *ClaimHandler {
	return &ClaimHandler{Store: store, authSvc: authSvc}
}

// Routes: all claim endpoints are public (the token is the credential).
func (h *ClaimHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{token}", h.getClaimInfo)
	r.Post("/{token}/send-otp", h.sendOTP)
	r.Post("/{token}/verify-otp", h.verifyOTP)
	r.Post("/{token}/complete", h.complete)
	return r
}

// ── Request types ─────────────────────────────────────────────────────────────

type claimSendOTPReq struct {
	WhatsApp string `json:"whatsapp"`
}

type claimVerifyOTPReq struct {
	WhatsApp string `json:"whatsapp"`
	OTPCode  string `json:"otp_code"`
}

type claimCompleteReq struct {
	WhatsApp string `json:"whatsapp"`
	OTPToken string `json:"otp_token"`
	Password string `json:"password"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// getValidClaim loads and validates a claim invite: existence, purpose, usage,
// expiration, and that the target is still pending. Returns nil after writing
// the error response when invalid.
func (h *ClaimHandler) getValidClaim(w http.ResponseWriter, r *http.Request) *db.ClaimInvite {
	token := chi.URLParam(r, "token")
	inv, err := h.Store.GetClaimInviteByToken(r.Context(), token)
	if err != nil {
		renderError(w, apierror.NotFound("INVITE_NOT_FOUND"))
		return nil
	}
	if inv.Used {
		renderError(w, apierror.Forbidden("INVITE_USED"))
		return nil
	}
	if time.Now().After(inv.ExpiresAt) {
		renderError(w, apierror.Forbidden("INVITE_EXPIRED"))
		return nil
	}
	if !inv.TargetPlayerPending {
		renderError(w, apierror.Forbidden("ALREADY_CLAIMED"))
		return nil
	}
	return inv
}

// checkWhatsAppAvailable returns an error when the number already belongs to a
// player other than the claim target.
func (h *ClaimHandler) checkWhatsAppAvailable(ctx context.Context, whatsapp string, targetID uuid.UUID) error {
	existing, err := h.Store.GetPlayerByWhatsApp(ctx, whatsapp)
	if err == nil && existing != nil && existing.ID != targetID {
		return apierror.Conflict("WHATSAPP_TAKEN")
	}
	return nil
}

func firstName(name string) string {
	for i, ch := range name {
		if ch == ' ' {
			return name[:i]
		}
	}
	return name
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// @Summary     Get claim invite info
// @Description Public info for a registration-claim invite (target first name + group).
// @Tags        claims
// @Success     200 {object} map[string]any
// @Failure     403 {object} apierror.APIError
// @Failure     404 {object} apierror.APIError
// @Router      /claims/{token} [get]
func (h *ClaimHandler) getClaimInfo(w http.ResponseWriter, r *http.Request) {
	inv := h.getValidClaim(w, r)
	if inv == nil {
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"valid":             true,
		"player_first_name": firstName(inv.TargetPlayerName),
		"group_name":        inv.GroupName,
		"expires_at":        inv.ExpiresAt,
	})
}

// @Summary     Send OTP for claim
// @Description Sends an OTP to the real number the claiming player provided.
// @Tags        claims
// @Param       body body claimSendOTPReq true "WhatsApp number"
// @Success     200 {object} map[string]string
// @Failure     409 {object} apierror.APIError
// @Failure     422 {object} apierror.APIError
// @Router      /claims/{token}/send-otp [post]
func (h *ClaimHandler) sendOTP(w http.ResponseWriter, r *http.Request) {
	inv := h.getValidClaim(w, r)
	if inv == nil {
		return
	}
	var req claimSendOTPReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	wa, err := services.NormalizeWhatsApp(req.WhatsApp)
	if err != nil {
		renderError(w, err)
		return
	}
	if err := h.checkWhatsAppAvailable(r.Context(), wa, *inv.TargetPlayerID); err != nil {
		renderError(w, err)
		return
	}
	if err := h.authSvc.SendOTPTo(wa); err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, map[string]string{"status": "pending"})
}

// @Summary     Verify OTP for claim
// @Description Verifies the OTP and returns a signed otp_token proving number ownership.
// @Tags        claims
// @Param       body body claimVerifyOTPReq true "WhatsApp + OTP code"
// @Success     200 {object} services.VerifyOTPResponse
// @Failure     409 {object} apierror.APIError
// @Failure     422 {object} apierror.APIError
// @Router      /claims/{token}/verify-otp [post]
func (h *ClaimHandler) verifyOTP(w http.ResponseWriter, r *http.Request) {
	inv := h.getValidClaim(w, r)
	if inv == nil {
		return
	}
	var req claimVerifyOTPReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	wa, err := services.NormalizeWhatsApp(req.WhatsApp)
	if err != nil {
		renderError(w, err)
		return
	}
	if err := h.checkWhatsAppAvailable(r.Context(), wa, *inv.TargetPlayerID); err != nil {
		renderError(w, err)
		return
	}
	otpToken, err := h.authSvc.VerifyOTPFor(wa, req.OTPCode)
	if err != nil {
		renderError(w, apierror.Unprocessable("OTP_INVALID"))
		return
	}
	renderJSON(w, http.StatusOK, services.VerifyOTPResponse{OTPToken: otpToken})
}

// @Summary     Complete registration claim
// @Description Sets the real whatsapp + password on the pending player, clears the provisional flags, marks the invite used and logs the player in.
// @Tags        claims
// @Param       body body claimCompleteReq true "WhatsApp + otp_token + new password"
// @Success     200 {object} services.TokenResponse
// @Failure     401 {object} apierror.APIError
// @Failure     409 {object} apierror.APIError
// @Failure     422 {object} apierror.APIError
// @Router      /claims/{token}/complete [post]
func (h *ClaimHandler) complete(w http.ResponseWriter, r *http.Request) {
	inv := h.getValidClaim(w, r)
	if inv == nil {
		return
	}
	var req claimCompleteReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if len(req.Password) < 6 {
		renderError(w, apierror.Unprocessable("password must be at least 6 characters"))
		return
	}
	wa, err := services.NormalizeWhatsApp(req.WhatsApp)
	if err != nil {
		renderError(w, err)
		return
	}
	otpWA, err := h.authSvc.DecodeOTP(req.OTPToken)
	if err != nil || otpWA != wa {
		renderError(w, &apierror.APIError{Code: http.StatusUnauthorized, Detail: "OTP_TOKEN_INVALID"})
		return
	}
	if err := h.checkWhatsAppAvailable(r.Context(), wa, *inv.TargetPlayerID); err != nil {
		renderError(w, err)
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		renderError(w, err)
		return
	}
	if err := h.Store.ClaimPlayerRegistration(r.Context(), *inv.TargetPlayerID, wa, hash); err != nil {
		// Unique violation race: another account took the number between verify and complete
		renderError(w, apierror.Conflict("WHATSAPP_TAKEN"))
		return
	}
	_ = h.Store.RevokeAllRefreshTokensForPlayer(r.Context(), *inv.TargetPlayerID)
	if err := h.Store.UseInvite(r.Context(), inv.Token, *inv.TargetPlayerID); err != nil {
		renderError(w, apierror.Internal("failed to mark invite as used"))
		return
	}

	player, err := h.Store.GetPlayerByID(r.Context(), *inv.TargetPlayerID)
	if err != nil {
		renderError(w, apierror.Internal("failed to reload player"))
		return
	}
	tokenResp, err := h.authSvc.IssueTokenPairForPlayer(r.Context(), player)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, tokenResp)
}
