package handlers

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

type playerHandler struct {
	pool    *pgxpool.Pool
	storage *services.StorageService
}

func NewPlayerHandler(pool *pgxpool.Pool, storage *services.StorageService) *playerHandler {
	return &playerHandler{pool: pool, storage: storage}
}

func (h *playerHandler) Routes() chi.Router {
	r := chi.NewRouter()
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

func (h *playerHandler) myMatches(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	rows, err := h.pool.Query(r.Context(), `
		SELECT
			m.id, m.group_id, m.number, m.hash,
			m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
			m.location, m.address, m.court_type::TEXT,
			m.players_per_team, m.max_players, m.notes,
			m.status::TEXT, m.created_at, m.updated_at,
			g.name AS group_name,
			g.timezone AS group_timezone,
			a.status::TEXT AS my_attendance
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		JOIN attendances a ON a.match_id = m.id AND a.player_id = $1
		ORDER BY m.match_date DESC, m.start_time DESC`, player.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	type matchItem struct {
		db.Match
		GroupName     string `json:"group_name"`
		GroupTimezone string `json:"group_timezone"`
		MyAttendance  string `json:"my_attendance"`
	}

	result := make([]matchItem, 0)
	for rows.Next() {
		var item matchItem
		if err := rows.Scan(
			&item.ID, &item.GroupID, &item.Number, &item.Hash,
			&item.MatchDate, &item.StartTime, &item.EndTime,
			&item.Location, &item.Address, &item.CourtType,
			&item.PlayersPerTeam, &item.MaxPlayers, &item.Notes,
			&item.Status, &item.CreatedAt, &item.UpdatedAt,
			&item.GroupName, &item.GroupTimezone, &item.MyAttendance,
		); err != nil {
			renderError(w, err)
			return
		}
		result = append(result, item)
	}
	renderJSON(w, http.StatusOK, result)
}

func (h *playerHandler) myStats(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	var minutesPlayed int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(
			EXTRACT(EPOCH FROM (
				CASE
					WHEN m.end_time IS NOT NULL
					THEN m.end_time::INTERVAL - m.start_time::INTERVAL
					ELSE INTERVAL '90 minutes'
				END
			)) / 60
		), 0)::INT
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		WHERE a.player_id = $1
		  AND a.status = 'confirmed'
		  AND m.status = 'closed'`, player.ID).Scan(&minutesPlayed)

	resp := map[string]any{"minutes_played": minutesPlayed}
	if player.Role == db.PlayerRoleAdmin {
		var platMinutes, platTotal int
		_ = h.pool.QueryRow(r.Context(), `
			SELECT
				COALESCE(COUNT(*) FILTER (WHERE m.status='closed'), 0),
				COALESCE(COUNT(DISTINCT a.player_id) FILTER (WHERE m.status='closed'), 0)
			FROM attendances a
			JOIN matches m ON m.id = a.match_id
			WHERE a.status = 'confirmed'`).Scan(&platTotal, &platMinutes)
		resp["platform_minutes_played"] = platMinutes * 90
		resp["platform_total_matches"] = platTotal
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *playerHandler) myStatsFull(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	type groupStat struct {
		GroupID   uuid.UUID `json:"group_id"`
		GroupName string    `json:"group_name"`
		Matches   int       `json:"matches_confirmed"`
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT g.id, g.name, COUNT(a.id)
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		JOIN groups g ON g.id = m.group_id
		WHERE a.player_id = $1 AND a.status = 'confirmed'
		GROUP BY g.id, g.name
		ORDER BY g.name`, player.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	stats := make([]groupStat, 0)
	for rows.Next() {
		var s groupStat
		if err := rows.Scan(&s.GroupID, &s.GroupName, &s.Matches); err != nil {
			renderError(w, err)
			return
		}
		stats = append(stats, s)
	}
	renderJSON(w, http.StatusOK, map[string]any{"groups": stats})
}

func (h *playerHandler) publicStats(w http.ResponseWriter, r *http.Request) {
	playerID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	target, err := db.GetPlayerByID(r.Context(), h.pool, playerID)
	if err != nil {
		renderError(w, err)
		return
	}

	var totalConfirmed int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM attendances a
		JOIN matches m ON m.id = a.match_id
		WHERE a.player_id = $1 AND a.status = 'confirmed' AND m.status = 'closed'`, playerID).
		Scan(&totalConfirmed)

	var totalGoals, totalAssists int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(goals),0), COALESCE(SUM(assists),0)
		FROM match_player_stats WHERE player_id = $1`, playerID).
		Scan(&totalGoals, &totalAssists)

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

func (h *playerHandler) listPlayers(w http.ResponseWriter, r *http.Request) {
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

	query := `
		SELECT ` + db.PlayerSelectCols + `
		FROM players
		WHERE role = 'player'`
	if activeOnly {
		query += ` AND active = TRUE`
	}
	query += ` ORDER BY name LIMIT $1 OFFSET $2`

	rows, err := h.pool.Query(r.Context(), query, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	players := make([]*db.Player, 0)
	for rows.Next() {
		p, err := db.ScanPlayer(rows.Scan)
		if err != nil {
			renderError(w, err)
			return
		}
		players = append(players, p)
	}
	renderJSON(w, http.StatusOK, players)
}

func (h *playerHandler) createPlayer(w http.ResponseWriter, r *http.Request) {
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

	p, err := db.CreatePlayer(r.Context(), h.pool, db.CreatePlayerParams{
		Name:         strings.TrimSpace(req.Name),
		WhatsApp:     normalizePhone(req.WhatsApp),
		PasswordHash: hash,
	})
	if err != nil {
		renderError(w, apierror.Conflict("whatsapp already registered"))
		return
	}
	_ = db.EnsurePlayerSubscription(r.Context(), h.pool, p.ID)
	renderJSON(w, http.StatusCreated, p)
}

func (h *playerHandler) getPlayer(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	targetID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	if caller.Role != db.PlayerRoleAdmin && caller.ID != targetID {
		renderError(w, apierror.Forbidden("access denied"))
		return
	}

	p, err := db.GetPlayerByID(r.Context(), h.pool, targetID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, p)
}

func (h *playerHandler) updatePlayer(w http.ResponseWriter, r *http.Request) {
	caller := middleware.PlayerFromCtx(r.Context())
	targetID, err := targetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
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

	target, err := db.GetPlayerByID(r.Context(), h.pool, targetID)
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

	_, err = h.pool.Exec(r.Context(), `
		UPDATE players SET name=$1, nickname=$2, password_hash=$3 WHERE id=$4`,
		target.Name, target.Nickname, target.PasswordHash, targetID)
	if err != nil {
		renderError(w, err)
		return
	}

	updated, _ := db.GetPlayerByID(r.Context(), h.pool, targetID)
	renderJSON(w, http.StatusOK, updated)
}

func (h *playerHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
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

	if err := db.UpdatePlayerPassword(r.Context(), h.pool, targetID, hash); err != nil {
		renderError(w, err)
		return
	}
	_ = db.UpdatePlayerMustChangePassword(r.Context(), h.pool, targetID, true)

	renderJSON(w, http.StatusOK, map[string]string{"temp_password": temp})
}

func (h *playerHandler) signupStats(w http.ResponseWriter, r *http.Request) {
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

	var total, last7, last30 int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE role='player'),
			COUNT(*) FILTER (WHERE role='player' AND created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE role='player' AND created_at >= NOW() - INTERVAL '30 days')
		FROM players WHERE active=TRUE`).Scan(&total, &last7, &last30)

	rows, _ := h.pool.Query(r.Context(), `
		SELECT id, name, nickname, whatsapp, active, created_at
		FROM players WHERE role='player' AND active=TRUE
		ORDER BY created_at DESC LIMIT $1`, limit)
	defer rows.Close()

	type recentSignup struct {
		ID        uuid.UUID   `json:"id"`
		Name      string      `json:"name"`
		Nickname  *string     `json:"nickname"`
		WhatsApp  string      `json:"whatsapp"`
		Active    bool        `json:"active"`
		CreatedAt interface{} `json:"created_at"`
	}
	recent := make([]recentSignup, 0)
	for rows.Next() {
		var s recentSignup
		_ = rows.Scan(&s.ID, &s.Name, &s.Nickname, &s.WhatsApp, &s.Active, &s.CreatedAt)
		recent = append(recent, s)
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "last_7_days": last7, "last_30_days": last30, "recent": recent,
	})
}

func (h *playerHandler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.pool.Exec(r.Context(),
		`UPDATE players SET avatar_url=$2 WHERE id=$1`, player.ID, publicURL); err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"avatar_url": publicURL})
}

func (h *playerHandler) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	if player.AvatarURL != nil && h.storage != nil {
		_ = h.storage.DeleteAvatarByURL(r.Context(), *player.AvatarURL)
	}

	if _, err := h.pool.Exec(r.Context(),
		`UPDATE players SET avatar_url=NULL WHERE id=$1`, player.ID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}
