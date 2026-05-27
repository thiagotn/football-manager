package handlers

import (
	cryptorand "crypto/rand"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

type PlayerStore interface {
	GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error)
	CreatePlayer(ctx context.Context, args db.CreatePlayerParams) (*db.Player, error)
	UpdatePlayerPassword(ctx context.Context, playerID uuid.UUID, hash string) error
	UpdatePlayerMustChangePassword(ctx context.Context, playerID uuid.UUID, val bool) error
	EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error
	GetPlayerMatches(ctx context.Context, playerID uuid.UUID) ([]db.PlayerMatch, error)
	GetPlayerStatsMinutes(ctx context.Context, playerID uuid.UUID) (int, error)
	GetPlatformMatchStats(ctx context.Context) (closedMatches, uniquePlayers, totalMinutes int, err error)
	GetPlayerGroupStats(ctx context.Context, playerID uuid.UUID) ([]db.GroupStat, error)
	GetPlayerGoalsAssists(ctx context.Context, playerID uuid.UUID) (goals, assists int, err error)
	GetPublicPlayerStats(ctx context.Context, playerID uuid.UUID) (totalConfirmed, totalGoals, totalAssists int, err error)
	UpdatePlayerProfile(ctx context.Context, playerID uuid.UUID, name, nickname, passwordHash string) error
	ListPlayersActive(ctx context.Context, limit, offset int, activeOnly bool) ([]*db.Player, error)
	GetSignupStats(ctx context.Context) (total, last7, last30 int, err error)
	GetRecentSignups(ctx context.Context, limit int) ([]db.RecentSignup, error)
	UpdatePlayerAvatarURL(ctx context.Context, playerID uuid.UUID, avatarURL string) error
	DeletePlayerAvatarURL(ctx context.Context, playerID uuid.UUID) error
}

type pgPlayerStore struct {
	pool *pgxpool.Pool
}

func (s *pgPlayerStore) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	return db.GetPlayerByID(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) CreatePlayer(ctx context.Context, args db.CreatePlayerParams) (*db.Player, error) {
	return db.CreatePlayer(ctx, s.pool, args)
}

func (s *pgPlayerStore) UpdatePlayerPassword(ctx context.Context, playerID uuid.UUID, hash string) error {
	return db.UpdatePlayerPassword(ctx, s.pool, playerID, hash)
}

func (s *pgPlayerStore) UpdatePlayerMustChangePassword(ctx context.Context, playerID uuid.UUID, val bool) error {
	return db.UpdatePlayerMustChangePassword(ctx, s.pool, playerID, val)
}

func (s *pgPlayerStore) EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error {
	return db.EnsurePlayerSubscription(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) GetPlayerMatches(ctx context.Context, playerID uuid.UUID) ([]db.PlayerMatch, error) {
	return db.GetPlayerMatches(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) GetPlayerStatsMinutes(ctx context.Context, playerID uuid.UUID) (int, error) {
	return db.GetPlayerStatsMinutes(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) GetPlatformMatchStats(ctx context.Context) (int, int, int, error) {
	return db.GetPlatformMatchStats(ctx, s.pool)
}

func (s *pgPlayerStore) GetPlayerGroupStats(ctx context.Context, playerID uuid.UUID) ([]db.GroupStat, error) {
	return db.GetPlayerGroupStats(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) GetPlayerGoalsAssists(ctx context.Context, playerID uuid.UUID) (int, int, error) {
	return db.GetPlayerGoalsAssists(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) GetPublicPlayerStats(ctx context.Context, playerID uuid.UUID) (int, int, int, error) {
	return db.GetPublicPlayerStats(ctx, s.pool, playerID)
}

func (s *pgPlayerStore) UpdatePlayerProfile(ctx context.Context, playerID uuid.UUID, name, nickname, passwordHash string) error {
	return db.UpdatePlayerProfile(ctx, s.pool, playerID, name, nickname, passwordHash)
}

func (s *pgPlayerStore) ListPlayersActive(ctx context.Context, limit, offset int, activeOnly bool) ([]*db.Player, error) {
	return db.ListPlayersActive(ctx, s.pool, limit, offset, activeOnly)
}

func (s *pgPlayerStore) GetSignupStats(ctx context.Context) (int, int, int, error) {
	return db.GetSignupStats(ctx, s.pool)
}

func (s *pgPlayerStore) GetRecentSignups(ctx context.Context, limit int) ([]db.RecentSignup, error) {
	return db.GetRecentSignups(ctx, s.pool, limit)
}

func (s *pgPlayerStore) UpdatePlayerAvatarURL(ctx context.Context, playerID uuid.UUID, avatarURL string) error {
	return db.UpdatePlayerAvatarURL(ctx, s.pool, playerID, avatarURL)
}

func (s *pgPlayerStore) DeletePlayerAvatarURL(ctx context.Context, playerID uuid.UUID) error {
	return db.DeletePlayerAvatarURL(ctx, s.pool, playerID)
}

type PlayerHandler struct {
	Store   PlayerStore
	storage *services.StorageService
}

func NewPlayerHandler(pool *pgxpool.Pool, storage *services.StorageService) *PlayerHandler {
	return &PlayerHandler{
		Store:   &pgPlayerStore{pool: pool},
		storage: storage,
	}
}

func (h *PlayerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/me", h.getMe)
	r.Get("/me/matches", h.myMatches)
	r.Get("/me/stats/full", h.myStatsFull)
	r.Get("/me/stats", h.myStats)
	r.Put("/me/avatar", h.uploadAvatar)
	r.Delete("/me/avatar", h.deleteAvatar)
	r.Get("/signups/stats", h.signupStats)
	r.Get("/", h.listPlayers)
	r.Post("/", h.createPlayer)
	r.Get("/{playerID}", h.getPlayer)
	r.Patch("/{playerID}", h.updatePlayer)
	r.Post("/{playerID}/reset-password", h.resetPassword)
	r.Get("/{playerID}/public-stats", h.publicStats)
	return r
}

// ── Request types ─────────────────────────────────────────────────────────────

type createPlayerReq struct {
	Name     string  `json:"name"`
	Nickname *string `json:"nickname"`
	WhatsApp string  `json:"whatsapp"`
	Password string  `json:"password"`
	Role     *string `json:"role"`
}

type updatePlayerReq struct {
	Name     *string `json:"name"`
	Nickname *string `json:"nickname"`
	WhatsApp *string `json:"whatsapp"`
	Password *string `json:"password"`
	Role     *string `json:"role"`
	Active   *bool   `json:"active"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

func targetPlayerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "playerID"))
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *PlayerHandler) myMatches(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	matches, err := h.Store.GetPlayerMatches(r.Context(), player.ID)
	if err != nil {
		renderError(w, err)
		return
	}

	type matchItem struct {
		ID            uuid.UUID `json:"id"`
		GroupID       uuid.UUID `json:"group_id"`
		Number        int       `json:"number"`
		Hash          string    `json:"hash"`
		MatchDate     string    `json:"match_date"`
		StartTime     string    `json:"start_time"`
		EndTime       *string   `json:"end_time"`
		Location      string    `json:"location"`
		Address       *string   `json:"address"`
		CourtType     *string   `json:"court_type"`
		PlayersPerTeam *int      `json:"players_per_team"`
		MaxPlayers    *int      `json:"max_players"`
		Notes         *string   `json:"notes"`
		Status        string    `json:"status"`
		CreatedAt     string    `json:"created_at"`
		UpdatedAt     string    `json:"updated_at"`
		GroupName     string    `json:"group_name"`
		GroupTimezone string    `json:"group_timezone"`
		MyAttendance  string    `json:"my_attendance"`
	}

	result := make([]matchItem, len(matches))
	for i, m := range matches {
		result[i] = matchItem{
			ID:             m.ID,
			GroupID:        m.GroupID,
			Number:         m.Number,
			Hash:           m.Hash,
			MatchDate:      m.MatchDate,
			StartTime:      m.StartTime,
			EndTime:        m.EndTime,
			Location:       m.Location,
			Address:        m.Address,
			CourtType:      m.CourtType,
			PlayersPerTeam: m.PlayersPerTeam,
			MaxPlayers:     m.MaxPlayers,
			Notes:          m.Notes,
			Status:         m.Status,
			CreatedAt:      m.CreatedAt,
			UpdatedAt:      m.UpdatedAt,
			GroupName:      m.GroupName,
			GroupTimezone:  m.GroupTimezone,
			MyAttendance:   m.MyAttendance,
		}
	}
	renderJSON(w, http.StatusOK, result)
}

func (h *PlayerHandler) myStats(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	minutesPlayed, _ := h.Store.GetPlayerStatsMinutes(r.Context(), player.ID)

	resp := map[string]any{"minutes_played": minutesPlayed}
	if player.Role == db.PlayerRoleAdmin {
		platTotal, _, platMinutes, _ := h.Store.GetPlatformMatchStats(r.Context())
		resp["platform_minutes_played"] = platMinutes
		resp["platform_total_matches"] = platTotal
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *PlayerHandler) myStatsFull(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	groupStats, err := h.Store.GetPlayerGroupStats(r.Context(), player.ID)
	if err != nil {
		renderError(w, err)
		return
	}

	type groupStat struct {
		GroupID   uuid.UUID `json:"group_id"`
		GroupName string    `json:"group_name"`
		Matches   int       `json:"matches_confirmed"`
	}

	stats := make([]groupStat, len(groupStats))
	for i, s := range groupStats {
		stats[i] = groupStat{
			GroupID:   s.GroupID,
			GroupName: s.GroupName,
			Matches:   s.Matches,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{"groups": stats})
}

func (h *PlayerHandler) publicStats(w http.ResponseWriter, r *http.Request) {
	playerID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	target, err := h.Store.GetPlayerByID(r.Context(), playerID)
	if err != nil {
		renderError(w, err)
		return
	}

	totalConfirmed, totalGoals, totalAssists, _ := h.Store.GetPublicPlayerStats(r.Context(), playerID)

	renderJSON(w, http.StatusOK, map[string]any{
		"player_id":               target.ID,
		"name":                    target.Name,
		"nickname":                target.Nickname,
		"avatar_url":              target.AvatarURL,
		"total_matches_confirmed": totalConfirmed,
		"total_goals":             totalGoals,
		"total_assists":           totalAssists,
	})
}

func (h *PlayerHandler) listPlayers(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin access required"))
		return
	}

	limit := 100
	offset := 0
	activeOnly := true
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 500 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := r.URL.Query().Get("active_only"); v == "false" {
		activeOnly = false
	}

	players, err := h.Store.ListPlayersActive(r.Context(), limit, offset, activeOnly)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, players)
}

func (h *PlayerHandler) createPlayer(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	if caller.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin access required"))
		return
	}

	var req createPlayerReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if len(strings.TrimSpace(req.Name)) < 2 {
		renderError(w, apierror.Unprocessable("name too short"))
		return
	}
	if len(req.Password) < 6 {
		renderError(w, apierror.Unprocessable("password must be at least 6 characters"))
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		renderError(w, err)
		return
	}

	p, err := h.Store.CreatePlayer(r.Context(), db.CreatePlayerParams{
		Name:         strings.TrimSpace(req.Name),
		WhatsApp:     normalizePhone(req.WhatsApp),
		PasswordHash: hash,
	})
	if err != nil {
		renderError(w, apierror.Conflict("whatsapp already registered"))
		return
	}
	_ = h.Store.EnsurePlayerSubscription(r.Context(), p.ID)
	renderJSON(w, http.StatusCreated, p)
}

func (h *PlayerHandler) getMe(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}
	p, err := h.Store.GetPlayerByID(r.Context(), player.ID)
	if err == db.ErrNotFound {
		renderError(w, apierror.NotFound("player not found"))
		return
	}
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch player"))
		return
	}
	renderJSON(w, http.StatusOK, p)
}

func (h *PlayerHandler) getPlayer(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	targetID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.Unprocessable("invalid player id"))
		return
	}

	if caller.Role != db.PlayerRoleAdmin && caller.ID != targetID {
		renderError(w, apierror.Forbidden("access denied"))
		return
	}

	p, err := h.Store.GetPlayerByID(r.Context(), targetID)
	if err == db.ErrNotFound {
		renderError(w, apierror.NotFound("player not found"))
		return
	}
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch player"))
		return
	}
	renderJSON(w, http.StatusOK, p)
}

func (h *PlayerHandler) updatePlayer(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	targetID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.Unprocessable("invalid player id"))
		return
	}

	if caller.Role != db.PlayerRoleAdmin && caller.ID != targetID {
		renderError(w, apierror.Forbidden("access denied"))
		return
	}

	var req updatePlayerReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	target, err := h.Store.GetPlayerByID(r.Context(), targetID)
	if err != nil {
		renderError(w, err)
		return
	}

	if req.Name != nil {
		target.Name = *req.Name
	}
	if req.Nickname != nil {
		target.Nickname = req.Nickname
	}
	if req.Password != nil {
		hash, err := hashPassword(*req.Password)
		if err != nil {
			renderError(w, err)
			return
		}
		target.PasswordHash = hash
	}

	nickname := ""
	if target.Nickname != nil {
		nickname = *target.Nickname
	}
	err = h.Store.UpdatePlayerProfile(r.Context(), targetID, target.Name, nickname, target.PasswordHash)
	if err != nil {
		renderError(w, err)
		return
	}

	updated, _ := h.Store.GetPlayerByID(r.Context(), targetID)
	renderJSON(w, http.StatusOK, updated)
}

func (h *PlayerHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	if caller.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin access required"))
		return
	}
	targetID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	// Generate temporary password
	b := make([]byte, 6)
	_, _ = cryptorand.Read(b)
	temp := strings.ToLower(strconv.FormatInt(int64(b[0])<<40|int64(b[1])<<32|int64(b[2])<<24|int64(b[3])<<16|int64(b[4])<<8|int64(b[5]), 36))
	if len(temp) > 8 {
		temp = temp[:8]
	}

	hash, err := hashPassword(temp)
	if err != nil {
		renderError(w, err)
		return
	}

	if err := h.Store.UpdatePlayerPassword(r.Context(), targetID, hash); err != nil {
		renderError(w, err)
		return
	}
	_ = h.Store.UpdatePlayerMustChangePassword(r.Context(), targetID, true)

	renderJSON(w, http.StatusOK, map[string]string{"temp_password": temp})
}

func (h *PlayerHandler) signupStats(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	if caller.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin access required"))
		return
	}

	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}

	total, last7, last30, _ := h.Store.GetSignupStats(r.Context())
	signups, _ := h.Store.GetRecentSignups(r.Context(), limit)

	type recentSignup struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		Nickname  *string   `json:"nickname"`
		WhatsApp  string    `json:"whatsapp"`
		Active    bool      `json:"active"`
		CreatedAt time.Time `json:"created_at"`
	}
	recent := make([]recentSignup, len(signups))
	for i, s := range signups {
		recent[i] = recentSignup{
			ID:        s.ID,
			Name:      s.Name,
			Nickname:  s.Nickname,
			WhatsApp:  s.WhatsApp,
			Active:    s.Active,
			CreatedAt: s.CreatedAt,
		}
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "last_7_days": last7, "last_30_days": last30, "recent": recent,
	})
}

func (h *PlayerHandler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil || !h.storage.IsConfigured() {
		renderError(w, apierror.Internal("storage not configured"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())

	const maxSize = 5 << 20 // 5 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		renderError(w, apierror.Unprocessable("failed to read upload body"))
		return
	}
	if len(data) == 0 {
		renderError(w, apierror.Unprocessable("empty file"))
		return
	}

	// Generate a random token to prevent enumeration by player_id
	tokenBytes := make([]byte, 8)
	if _, err := cryptorand.Read(tokenBytes); err != nil {
		renderError(w, err)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Delete previous avatar if exists
	if player.AvatarURL != nil {
		_ = h.storage.DeleteAvatarByURL(r.Context(), *player.AvatarURL)
	}

	publicURL, err := h.storage.UploadAvatar(r.Context(), player.ID.String(), token, data)
	if err != nil {
		renderError(w, apierror.Internal("failed to upload avatar"))
		return
	}

	if err := h.Store.UpdatePlayerAvatarURL(r.Context(), player.ID, publicURL); err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"avatar_url": publicURL})
}

func (h *PlayerHandler) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	if player.AvatarURL != nil && h.storage != nil {
		_ = h.storage.DeleteAvatarByURL(r.Context(), *player.AvatarURL)
	}

	if err := h.Store.DeletePlayerAvatarURL(r.Context(), player.ID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}
