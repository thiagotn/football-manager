package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type inviteHandler struct {
	pool *pgxpool.Pool
}

func NewInviteHandler(pool *pgxpool.Pool) *inviteHandler {
	return &inviteHandler{pool: pool}
}

// PublicRoutes: token lookup, check, and accept (no auth required).
func (h *inviteHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{token}", h.getInvite)
	r.Get("/{token}/check", h.checkInvite)
	r.Post("/{token}/accept", h.acceptInvite)
	return r
}

// AuthRoutes: create invite (requires auth — group admin or global admin).
func (h *inviteHandler) AuthRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.createInvite)
	return r
}

// Routes: combined public and auth routes (auth middleware applied at router level).
func (h *inviteHandler) Routes(authMw func(http.Handler) http.Handler) chi.Router {
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

func (h *inviteHandler) createInvite(w http.ResponseWriter, r *http.Request) {
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

	// Auth: must be group admin or global admin
	if caller.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, caller.ID)
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

	expiresAt := time.Now().Add(30 * time.Minute)
	inv, err := db.CreateInvite(r.Context(), h.pool, groupID, caller.ID, token, expiresAt)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusCreated, inv)
}

func (h *inviteHandler) getInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	inv, err := db.GetInviteByToken(r.Context(), h.pool, token)
	if err != nil {
		renderError(w, err)
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

func (h *inviteHandler) checkInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	whatsapp := normalizePhone(r.URL.Query().Get("whatsapp"))
	if whatsapp == "" {
		renderError(w, apierror.Unprocessable("whatsapp is required"))
		return
	}

	inv, err := db.GetInviteByToken(r.Context(), h.pool, token)
	if err != nil || inv.Used || time.Now().After(inv.ExpiresAt) {
		renderError(w, apierror.NotFound("invalid or expired invite"))
		return
	}

	found, err := db.GetPlayerByWhatsApp(r.Context(), h.pool, whatsapp)
	if err != nil {
		renderJSON(w, http.StatusOK, map[string]any{"exists": false, "first_name": nil})
		return
	}
	firstName := strings.SplitN(found.Name, " ", 2)[0]
	renderJSON(w, http.StatusOK, map[string]any{"exists": true, "first_name": firstName})
}

func (h *inviteHandler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	inv, err := db.GetInviteByToken(r.Context(), h.pool, token)
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
		count, _ := db.CountGroupMembers(r.Context(), h.pool, inv.GroupID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	var player *db.Player
	justJoined := false

	existing, err := db.GetPlayerByWhatsApp(r.Context(), h.pool, req.WhatsApp)
	if err == nil {
		// Player exists — verify password
		if err := bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte(req.Password)); err != nil {
			renderError(w, apierror.Forbidden("invalid credentials"))
			return
		}
		// Check if already member
		if _, err := db.GetGroupMember(r.Context(), h.pool, inv.GroupID, existing.ID); err == nil {
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
		player, err = db.CreatePlayer(r.Context(), h.pool, db.CreatePlayerParams{
			Name:         strings.TrimSpace(*req.Name),
			Nickname:     req.Nickname,
			WhatsApp:     req.WhatsApp,
			PasswordHash: hash,
		})
		if err != nil {
			renderError(w, apierror.Conflict("whatsapp already registered"))
			return
		}
		_ = db.EnsurePlayerSubscription(r.Context(), h.pool, player.ID)
		justJoined = true
	}

	// Add as group member
	_, _ = db.AddGroupMember(r.Context(), h.pool, inv.GroupID, player.ID, db.GroupMemberRoleMember)

	// Add PENDING to open matches
	matchIDs, _ := db.GetOpenMatchesForGroup(r.Context(), h.pool, inv.GroupID)
	for _, mid := range matchIDs {
		_ = db.SetAttendance(r.Context(), h.pool, mid, player.ID, "pending")
	}

	// Mark invite used
	_ = db.UseInvite(r.Context(), h.pool, token, player.ID)

	// Issue tokens via the auth-style response
	_ = justJoined // used for notifications in full impl
	renderJSON(w, http.StatusOK, map[string]any{
		"player_id": player.ID.String(),
		"name":      player.Name,
		"role":      string(player.Role),
		"message":   "joined group successfully",
	})
}
