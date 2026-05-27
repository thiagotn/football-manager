package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/config"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

type InviteStore interface {
	GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	CreateInvite(ctx context.Context, groupID, callerID uuid.UUID, token string, expiresAt time.Time) (*db.Invite, error)
	GetInviteByToken(ctx context.Context, token string) (*db.InviteWithGroup, error)
	GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error)
	CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error)
	CreatePlayer(ctx context.Context, params db.CreatePlayerParams) (*db.Player, error)
	EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error
	AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error)
	GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error
	UseInvite(ctx context.Context, token string, playerID uuid.UUID) error
	EnsureMemberInCurrentPeriod(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error
}

type pgInviteStore struct {
	pool *pgxpool.Pool
}

func (s *pgInviteStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return db.GetGroupMember(ctx, s.pool, groupID, playerID)
}

func (s *pgInviteStore) CreateInvite(ctx context.Context, groupID, callerID uuid.UUID, token string, expiresAt time.Time) (*db.Invite, error) {
	return db.CreateInvite(ctx, s.pool, groupID, callerID, token, expiresAt)
}

func (s *pgInviteStore) GetInviteByToken(ctx context.Context, token string) (*db.InviteWithGroup, error) {
	return db.GetInviteByToken(ctx, s.pool, token)
}

func (s *pgInviteStore) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	return db.GetPlayerByWhatsApp(ctx, s.pool, whatsapp)
}

func (s *pgInviteStore) CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	return db.CountGroupMembers(ctx, s.pool, groupID)
}

func (s *pgInviteStore) CreatePlayer(ctx context.Context, params db.CreatePlayerParams) (*db.Player, error) {
	return db.CreatePlayer(ctx, s.pool, params)
}

func (s *pgInviteStore) EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error {
	return db.EnsurePlayerSubscription(ctx, s.pool, playerID)
}

func (s *pgInviteStore) AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error) {
	return db.AddGroupMember(ctx, s.pool, groupID, playerID, role)
}

func (s *pgInviteStore) GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return db.GetOpenMatchesForGroup(ctx, s.pool, groupID)
}

func (s *pgInviteStore) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	return db.SetAttendance(ctx, s.pool, matchID, playerID, status)
}

func (s *pgInviteStore) UseInvite(ctx context.Context, token string, playerID uuid.UUID) error {
	return db.UseInvite(ctx, s.pool, token, playerID)
}

func (s *pgInviteStore) EnsureMemberInCurrentPeriod(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error {
	return db.EnsureMemberInCurrentPeriod(ctx, s.pool, groupID, playerID, playerName)
}

type InviteHandler struct {
	Store   InviteStore
	authSvc services.AuthService
	cfg     *config.Config
}

func NewInviteHandler(pool *pgxpool.Pool, authSvc services.AuthService, cfg *config.Config) *InviteHandler {
	return &InviteHandler{Store: &pgInviteStore{pool: pool}, authSvc: authSvc, cfg: cfg}
}

// PublicRoutes: token lookup, check, and accept (no auth required).
func (h *InviteHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{token}", h.getInvite)
	r.Get("/{token}/check", h.checkInvite)
	r.Post("/{token}/accept", h.acceptInvite)
	return r
}

// AuthRoutes: create invite (requires auth — group admin or global admin).
func (h *InviteHandler) AuthRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.createInvite)
	return r
}

// Routes: combined public and auth routes (auth middleware applied at router level).
func (h *InviteHandler) Routes(authMw func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth)
	r.Get("/{token}", h.getInvite)
	r.Get("/{token}/check", h.checkInvite)
	r.Post("/{token}/accept", h.acceptInvite)

	// Auth-required routes
	r.Group(func(r chi.Router) {
		r.Use(authMw)
		r.Post("/", h.createInvite)
	})

	return r
}

// ── Request types ─────────────────────────────────────────────────────────────

type createInviteReq struct {
	GroupID string `json:"group_id"`
}

type inviteAcceptReq struct {
	Name     *string `json:"name"`
	Nickname *string `json:"nickname"`
	WhatsApp string  `json:"whatsapp"`
	Password string  `json:"password"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:32], nil
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *InviteHandler) createInvite(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())

	var req createInviteReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	groupID, err := uuid.Parse(req.GroupID)
	if err != nil {
		renderError(w, apierror.Unprocessable("invalid group_id"))
		return
	}

	// Check permissions: only global admins or group admins can create invites
	if caller.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, caller.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can create invites"))
			return
		}
	}

	token, err := generateToken()
	if err != nil {
		renderError(w, err)
		return
	}

	expiresAt := time.Now().Add(time.Duration(h.cfg.InviteTokenExpireMinutes) * time.Minute)
	inv, err := h.Store.CreateInvite(r.Context(), groupID, caller.ID, token, expiresAt)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusCreated, inv)
}

func (h *InviteHandler) getInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	inv, err := h.Store.GetInviteByToken(r.Context(), token)
	if err == db.ErrNotFound {
		renderError(w, apierror.NotFound("invite not found"))
		return
	}
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch invite"))
		return
	}
	if inv.Used {
		renderError(w, apierror.Forbidden("invite already used"))
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		renderError(w, apierror.Forbidden("invite expired"))
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"valid":      true,
		"group_id":   inv.GroupID.String(),
		"group_name": inv.GroupName,
		"expires_at": inv.ExpiresAt,
	})
}

func (h *InviteHandler) checkInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	whatsapp := normalizePhone(r.URL.Query().Get("whatsapp"))
	if whatsapp == "" {
		renderError(w, apierror.Unprocessable("whatsapp is required"))
		return
	}

	inv, err := h.Store.GetInviteByToken(r.Context(), token)
	if err != nil || inv.Used || time.Now().After(inv.ExpiresAt) {
		renderError(w, apierror.NotFound("invalid or expired invite"))
		return
	}

	found, err := h.Store.GetPlayerByWhatsApp(r.Context(), whatsapp)
	if err != nil {
		renderJSON(w, http.StatusOK, map[string]any{"exists": false, "first_name": nil})
		return
	}
	firstName := strings.SplitN(found.Name, " ", 2)[0]
	renderJSON(w, http.StatusOK, map[string]any{"exists": true, "first_name": firstName})
}

func (h *InviteHandler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	inv, err := h.Store.GetInviteByToken(r.Context(), token)
	if err != nil || inv.Used || time.Now().After(inv.ExpiresAt) {
		renderError(w, apierror.NotFound("invalid or expired invite"))
		return
	}

	var req inviteAcceptReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	req.WhatsApp = normalizePhone(req.WhatsApp)
	if len(req.Password) < 6 {
		renderError(w, apierror.Unprocessable("password must be at least 6 characters"))
		return
	}

	// Plan members limit (use first admin's plan of the group as proxy)
	plan := "free"
	limit := db.PlanMembersLimit(plan)
	if limit > 0 {
		count, _ := h.Store.CountGroupMembers(r.Context(), inv.GroupID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	var player *db.Player
	justJoined := false

	existing, err := h.Store.GetPlayerByWhatsApp(r.Context(), req.WhatsApp)
	if err == nil {
		// Player exists — verify password
		if err := bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte(req.Password)); err != nil {
			renderError(w, apierror.Forbidden("invalid credentials"))
			return
		}
		// Check if already member
		if _, err := h.Store.GetGroupMember(r.Context(), inv.GroupID, existing.ID); err == nil {
			renderError(w, apierror.Conflict("already a member of this group"))
			return
		}
		player = existing
	} else {
		// New player
		if req.Name == nil || len(strings.TrimSpace(*req.Name)) < 2 {
			renderError(w, apierror.Unprocessable("name is required for new players"))
			return
		}
		hash, err := hashPassword(req.Password)
		if err != nil {
			renderError(w, err)
			return
		}
		player, err = h.Store.CreatePlayer(r.Context(), db.CreatePlayerParams{
			Name:         strings.TrimSpace(*req.Name),
			Nickname:     req.Nickname,
			WhatsApp:     req.WhatsApp,
			PasswordHash: hash,
		})
		if err != nil {
			renderError(w, apierror.Conflict("whatsapp already registered"))
			return
		}
		_ = h.Store.EnsurePlayerSubscription(r.Context(), player.ID)
		justJoined = true
	}

	// Add as group member
	_, _ = h.Store.AddGroupMember(r.Context(), inv.GroupID, player.ID, db.GroupMemberRoleMember)

	// Add PENDING to open matches and ensure member in current finance period
	if justJoined {
		matchIDs, _ := h.Store.GetOpenMatchesForGroup(r.Context(), inv.GroupID)
		for _, mid := range matchIDs {
			_ = h.Store.SetAttendance(r.Context(), mid, player.ID, "pending")
		}
		playerDisplayName := player.Nickname
		if playerDisplayName == nil || *playerDisplayName == "" {
			playerDisplayName = &player.Name
		}
		_ = h.Store.EnsureMemberInCurrentPeriod(r.Context(), inv.GroupID, player.ID, *playerDisplayName)
	}

	// Mark invite used
	if err := h.Store.UseInvite(r.Context(), token, player.ID); err != nil {
		renderError(w, fmt.Errorf("failed to mark invite as used"))
		return
	}

	// Issue tokens via the auth-style response
	tokenResp, err := h.authSvc.IssueTokenPairForPlayer(r.Context(), player)
	if err != nil {
		renderError(w, err)
		return
	}
	_ = justJoined // used for notifications in full impl
	renderJSON(w, http.StatusOK, tokenResp)
}
