package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type matchHandler struct {
	pool *pgxpool.Pool
}

func NewMatchHandler(pool *pgxpool.Pool) *matchHandler {
	return &matchHandler{pool: pool}
}

// GroupMatchRoutes returns routes mounted under /groups/{groupID}.
func (h *matchHandler) GroupMatchRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.listGroupMatches)
	r.Post("/", h.createMatch)
	r.Route("/{matchID}", func(r chi.Router) {
		r.Get("/", h.getMatch)
		r.Patch("/", h.updateMatch)
		r.Delete("/", h.deleteMatch)
		r.Post("/attendance", h.setAttendance)
	})
	return r
}

// ── Request / Response types ─────────────────────────────────────────────────

type createMatchReq struct {
	MatchDate      string  `json:"match_date"`
	StartTime      string  `json:"start_time"`
	EndTime        *string `json:"end_time"`
	Location       string  `json:"location"`
	Address        *string `json:"address"`
	CourtType      *string `json:"court_type"`
	PlayersPerTeam *int    `json:"players_per_team"`
	MaxPlayers     *int    `json:"max_players"`
	Notes          *string `json:"notes"`
}

type updateMatchReq struct {
	MatchDate      *string `json:"match_date"`
	StartTime      *string `json:"start_time"`
	EndTime        *string `json:"end_time"`
	Location       *string `json:"location"`
	Address        *string `json:"address"`
	CourtType      *string `json:"court_type"`
	PlayersPerTeam *int    `json:"players_per_team"`
	MaxPlayers     *int    `json:"max_players"`
	Notes          *string `json:"notes"`
	Status         *string `json:"status"`
}

type setAttendanceReq struct {
	PlayerID uuid.UUID `json:"player_id"`
	Status   string    `json:"status"`
}

type playerStatInput struct {
	PlayerID uuid.UUID `json:"player_id"`
	Goals    int       `json:"goals"`
	Assists  int       `json:"assists"`
}

type upsertStatsReq struct {
	Stats []playerStatInput `json:"stats"`
}

type attendancePlayerView struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Nickname  *string   `json:"nickname"`
	Role      string    `json:"role"`
	AvatarURL *string   `json:"avatar_url"`
}

type attendanceResp struct {
	ID            uuid.UUID            `json:"id"`
	Player        attendancePlayerView `json:"player"`
	Status        string               `json:"status"`
	UpdatedAt     interface{}          `json:"updated_at"`
	Position      string               `json:"position"`
	GroupNickname *string              `json:"group_nickname"`
}

type matchDetailResp struct {
	db.Match
	Attendances    []attendanceResp `json:"attendances"`
	ConfirmedCount int              `json:"confirmed_count"`
	DeclinedCount  int              `json:"declined_count"`
	PendingCount   int              `json:"pending_count"`
	GroupName      string           `json:"group_name,omitempty"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func matchIDParam(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "matchID"))
}

func generateHash() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(b)
	if len(encoded) > 10 {
		return encoded[:10], nil
	}
	return encoded, nil
}

func buildAttendanceResp(a db.AttendanceWithPlayer) attendanceResp {
	return attendanceResp{
		ID: a.ID,
		Player: attendancePlayerView{
			ID:        a.PlayerID,
			Name:      a.PlayerName,
			Nickname:  a.PlayerNickname,
			Role:      a.PlayerRole,
			AvatarURL: a.PlayerAvatarURL,
		},
		Status:        a.Status,
		UpdatedAt:     a.UpdatedAt,
		Position:      a.Position,
		GroupNickname: a.GroupNickname,
	}
}

func buildMatchDetail(match *db.Match, atts []db.AttendanceWithPlayer, groupName string) matchDetailResp {
	resp := matchDetailResp{
		Match:       *match,
		GroupName:   groupName,
		Attendances: make([]attendanceResp, 0, len(atts)),
	}
	for _, a := range atts {
		resp.Attendances = append(resp.Attendances, buildAttendanceResp(a))
		switch a.Status {
		case "confirmed":
			resp.ConfirmedCount++
		case "declined":
			resp.DeclinedCount++
		default:
			resp.PendingCount++
		}
	}
	return resp
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (h *matchHandler) DiscoverMatches(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 50 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	matches, err := db.GetDiscoverMatches(r.Context(), h.pool, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, matches)
}

func (h *matchHandler) GetPublicMatch(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := db.GetMatchByHash(r.Context(), h.pool, hash)
	if err != nil {
		renderError(w, err)
		return
	}

	group, _ := db.GetGroupByID(r.Context(), h.pool, match.GroupID)
	groupName := ""
	if group != nil {
		groupName = group.Name
	}

	atts, err := db.GetAttendancesForMatch(r.Context(), h.pool, match.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, buildMatchDetail(match, atts, groupName))
}

func (h *matchHandler) GetPublicMatchStats(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := db.GetMatchByHash(r.Context(), h.pool, hash)
	if err != nil {
		renderError(w, err)
		return
	}

	stats, err := db.GetMatchPlayerStats(r.Context(), h.pool, match.ID)
	if err != nil {
		renderError(w, err)
		return
	}

	statResp := make([]map[string]any, 0, len(stats))
	for _, s := range stats {
		statResp = append(statResp, map[string]any{
			"player_id":   s.PlayerID,
			"player_name": s.PlayerName,
			"avatar_url":  s.AvatarURL,
			"goals":       s.Goals,
			"assists":     s.Assists,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"match_hash": hash,
		"registered": len(stats) > 0,
		"stats":      statResp,
	})
}

func (h *matchHandler) listGroupMatches(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		if _, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID); err != nil {
			renderError(w, apierror.Forbidden("not a member"))
			return
		}
	}

	matches, err := db.GetMatchesByGroup(r.Context(), h.pool, groupID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, matches)
}

func (h *matchHandler) createMatch(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can create matches"))
			return
		}
	}

	var req createMatchReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if req.MatchDate == "" || req.StartTime == "" || req.Location == "" {
		renderError(w, apierror.Unprocessable("match_date, start_time, and location are required"))
		return
	}

	hash, err := generateHash()
	if err != nil {
		renderError(w, err)
		return
	}

	number, err := db.NextMatchNumber(r.Context(), h.pool, groupID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get next match number"))
		return
	}

	match, err := db.CreateMatch(r.Context(), h.pool, db.CreateMatchParams{
		GroupID:        groupID,
		Hash:           hash,
		Number:         number,
		MatchDate:      req.MatchDate,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Location:       req.Location,
		Address:        req.Address,
		CourtType:      req.CourtType,
		PlayersPerTeam: req.PlayersPerTeam,
		MaxPlayers:     req.MaxPlayers,
		Notes:          req.Notes,
		CreatedByID:    player.ID,
	})
	if err != nil {
		renderError(w, apierror.Internal("failed to create match"))
		return
	}

	// Add PENDING attendance for all group members
	memberIDs, _ := db.GetGroupMemberPlayerIDs(r.Context(), h.pool, groupID)
	for _, pid := range memberIDs {
		_ = db.SetAttendance(r.Context(), h.pool, match.ID, pid, "pending")
	}

	renderJSON(w, http.StatusCreated, match)
}

func (h *matchHandler) getMatch(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		if _, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID); err != nil {
			renderError(w, apierror.Forbidden("not a member"))
			return
		}
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, err)
		return
	}
	if match.GroupID != groupID {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	atts, err := db.GetAttendancesForMatch(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, buildMatchDetail(match, atts, ""))
}

func (h *matchHandler) updateMatch(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can update matches"))
			return
		}
	}

	var req updateMatchReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	match, err := db.UpdateMatch(r.Context(), h.pool, matchID, db.UpdateMatchParams{
		MatchDate:      req.MatchDate,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Location:       req.Location,
		Address:        req.Address,
		CourtType:      req.CourtType,
		PlayersPerTeam: req.PlayersPerTeam,
		MaxPlayers:     req.MaxPlayers,
		Notes:          req.Notes,
		Status:         req.Status,
	})
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, match)
}

func (h *matchHandler) deleteMatch(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can delete matches"))
			return
		}
	}

	if err := db.DeleteMatch(r.Context(), h.pool, matchID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *matchHandler) setAttendance(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player.Role == db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admins cannot set attendance"))
		return
	}
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	var req setAttendanceReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	if req.Status != "pending" && req.Status != "confirmed" && req.Status != "declined" {
		renderError(w, apierror.Unprocessable("invalid status"))
		return
	}

	// Auth check
	callerMem, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
	if err != nil {
		renderError(w, apierror.Forbidden("not a member"))
		return
	}

	// Non-admins can only set their own attendance
	if callerMem.Role != db.GroupMemberRoleAdmin && req.PlayerID != player.ID {
		renderError(w, apierror.Forbidden("can only set own attendance"))
		return
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, err)
		return
	}
	if match.GroupID != groupID {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if match.Status == "closed" {
		renderError(w, apierror.Forbidden("match is closed"))
		return
	}

	// Max players check for confirming
	if req.Status == "confirmed" && match.MaxPlayers != nil {
		confirmed, _ := db.CountAttendances(r.Context(), h.pool, matchID, "confirmed")
		if confirmed >= *match.MaxPlayers {
			renderError(w, apierror.Forbidden("match is full"))
			return
		}
	}

	if err := db.SetAttendance(r.Context(), h.pool, matchID, req.PlayerID, req.Status); err != nil {
		renderError(w, err)
		return
	}

	// Build response
	atts, _ := db.GetAttendancesForMatch(r.Context(), h.pool, matchID)
	for _, a := range atts {
		if a.PlayerID == req.PlayerID {
			renderJSON(w, http.StatusOK, buildAttendanceResp(a))
			return
		}
	}
	renderJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (h *matchHandler) UpsertPlayerStats(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	hash := chi.URLParam(r, "hash")

	match, err := db.GetMatchByHash(r.Context(), h.pool, hash)
	if err != nil {
		renderError(w, err)
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, match.GroupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can record stats"))
			return
		}
	}

	var req upsertStatsReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	for _, s := range req.Stats {
		if err := db.UpsertMatchPlayerStat(r.Context(), h.pool, match.ID, s.PlayerID, player.ID, s.Goals, s.Assists); err != nil {
			renderError(w, err)
			return
		}
	}

	stats, err := db.GetMatchPlayerStats(r.Context(), h.pool, match.ID)
	if err != nil {
		renderError(w, err)
		return
	}

	statResp := make([]map[string]any, 0, len(stats))
	for _, s := range stats {
		statResp = append(statResp, map[string]any{
			"player_id": s.PlayerID, "player_name": s.PlayerName,
			"avatar_url": s.AvatarURL, "goals": s.Goals, "assists": s.Assists,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"match_hash": hash, "registered": true, "stats": statResp,
	})
}
