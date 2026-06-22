package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

type MatchStore interface {
	GetDiscoverMatches(ctx context.Context, playerID *uuid.UUID, limit, offset int) ([]db.DiscoverMatch, error)
	GetMatchByHash(ctx context.Context, hash string) (*db.Match, error)
	GetMatchByHashWithGroup(ctx context.Context, hash string) (*db.MatchWithGroupName, error)
	GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error)
	GetMatchesByGroup(ctx context.Context, groupID uuid.UUID) ([]db.Match, error)
	GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error)
	GetAttendancesForMatch(ctx context.Context, matchID uuid.UUID) ([]db.AttendanceWithPlayer, error)
	GetMatchPlayerStats(ctx context.Context, matchID uuid.UUID) ([]db.MatchPlayerStat, error)
	GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	NextMatchNumber(ctx context.Context, groupID uuid.UUID) (int, error)
	CreateMatch(ctx context.Context, params db.CreateMatchParams) (*db.Match, error)
	UpdateMatch(ctx context.Context, matchID uuid.UUID, params db.UpdateMatchParams) (*db.Match, error)
	DeleteMatch(ctx context.Context, matchID uuid.UUID) error
	GetGroupMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	GetNonAdminMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error
	CountAttendances(ctx context.Context, matchID uuid.UUID, status string) (int, error)
	UpsertMatchPlayerStat(ctx context.Context, matchID, playerID, recordedBy uuid.UUID, goals, assists int) error
}

type pgMatchStore struct {
	pool *pgxpool.Pool
}

func (s *pgMatchStore) GetDiscoverMatches(ctx context.Context, playerID *uuid.UUID, limit, offset int) ([]db.DiscoverMatch, error) {
	return db.GetDiscoverMatches(ctx, s.pool, playerID, limit, offset)
}
func (s *pgMatchStore) GetMatchByHash(ctx context.Context, hash string) (*db.Match, error) {
	return db.GetMatchByHash(ctx, s.pool, hash)
}
func (s *pgMatchStore) GetMatchByHashWithGroup(ctx context.Context, hash string) (*db.MatchWithGroupName, error) {
	return db.GetMatchByHashWithGroup(ctx, s.pool, hash)
}
func (s *pgMatchStore) GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error) {
	return db.GetMatchByID(ctx, s.pool, matchID)
}
func (s *pgMatchStore) GetMatchesByGroup(ctx context.Context, groupID uuid.UUID) ([]db.Match, error) {
	return db.GetMatchesByGroup(ctx, s.pool, groupID)
}
func (s *pgMatchStore) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error) {
	return db.GetGroupByID(ctx, s.pool, groupID)
}
func (s *pgMatchStore) GetAttendancesForMatch(ctx context.Context, matchID uuid.UUID) ([]db.AttendanceWithPlayer, error) {
	return db.GetAttendancesForMatch(ctx, s.pool, matchID)
}
func (s *pgMatchStore) GetMatchPlayerStats(ctx context.Context, matchID uuid.UUID) ([]db.MatchPlayerStat, error) {
	return db.GetMatchPlayerStats(ctx, s.pool, matchID)
}
func (s *pgMatchStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return db.GetGroupMember(ctx, s.pool, groupID, playerID)
}
func (s *pgMatchStore) NextMatchNumber(ctx context.Context, groupID uuid.UUID) (int, error) {
	return db.NextMatchNumber(ctx, s.pool, groupID)
}
func (s *pgMatchStore) CreateMatch(ctx context.Context, params db.CreateMatchParams) (*db.Match, error) {
	return db.CreateMatch(ctx, s.pool, params)
}
func (s *pgMatchStore) UpdateMatch(ctx context.Context, matchID uuid.UUID, params db.UpdateMatchParams) (*db.Match, error) {
	return db.UpdateMatch(ctx, s.pool, matchID, params)
}
func (s *pgMatchStore) DeleteMatch(ctx context.Context, matchID uuid.UUID) error {
	return db.DeleteMatch(ctx, s.pool, matchID)
}
func (s *pgMatchStore) GetGroupMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return db.GetGroupMemberPlayerIDs(ctx, s.pool, groupID)
}
func (s *pgMatchStore) GetNonAdminMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return db.GetNonAdminMemberPlayerIDs(ctx, s.pool, groupID)
}
func (s *pgMatchStore) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	return db.SetAttendance(ctx, s.pool, matchID, playerID, status)
}
func (s *pgMatchStore) CountAttendances(ctx context.Context, matchID uuid.UUID, status string) (int, error) {
	return db.CountAttendances(ctx, s.pool, matchID, status)
}
func (s *pgMatchStore) UpsertMatchPlayerStat(ctx context.Context, matchID, playerID, recordedBy uuid.UUID, goals, assists int) error {
	return db.UpsertMatchPlayerStat(ctx, s.pool, matchID, playerID, recordedBy, goals, assists)
}

type MatchHandler struct {
	Store MatchStore
	pool  *pgxpool.Pool // retained for fire-and-forget service calls (lazy status sync)
}

func NewMatchHandler(pool *pgxpool.Pool) *MatchHandler {
	return &MatchHandler{Store: &pgMatchStore{pool: pool}, pool: pool}
}

// GroupMatchRoutes returns routes mounted under /groups/{groupID}.
func (h *MatchHandler) GroupMatchRoutes() chi.Router {
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
	Attendances         []attendanceResp `json:"attendances"`
	ConfirmedCount      int              `json:"confirmed_count"`
	DeclinedCount       int              `json:"declined_count"`
	PendingCount        int              `json:"pending_count"`
	GroupName           string           `json:"group_name"`
	GroupTimezone       string           `json:"group_timezone"`
	GroupPerMatchAmount *float64         `json:"group_per_match_amount"`
	GroupMonthlyAmount  *float64         `json:"group_monthly_amount"`
	GroupIsPublic       bool             `json:"group_is_public"`
	GroupVotingEnabled  bool             `json:"group_voting_enabled"`
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

// matchGroupFields carries the per-group fields embedded in match detail
// responses. Pass nil/empty values via groupFieldsFromName when only the
// group name is known.
type matchGroupFields struct {
	Name           string
	Timezone       string
	PerMatchAmount *float64
	MonthlyAmount  *float64
	IsPublic       bool
	VotingEnabled  bool
}

func groupFieldsFromName(name string) matchGroupFields {
	return matchGroupFields{
		Name:          name,
		Timezone:      "America/Sao_Paulo",
		IsPublic:      true,
		VotingEnabled: true,
	}
}

func groupFieldsFromMatch(m *db.MatchWithGroupName) matchGroupFields {
	return matchGroupFields{
		Name:           m.GroupName,
		Timezone:       m.GroupTimezone,
		PerMatchAmount: m.GroupPerMatchAmount,
		MonthlyAmount:  m.GroupMonthlyAmount,
		IsPublic:       m.GroupIsPublic,
		VotingEnabled:  m.GroupVotingEnabled,
	}
}

func buildMatchDetail(match *db.Match, atts []db.AttendanceWithPlayer, group matchGroupFields) matchDetailResp {
	resp := matchDetailResp{
		Match:               *match,
		GroupName:           group.Name,
		GroupTimezone:       group.Timezone,
		GroupPerMatchAmount: group.PerMatchAmount,
		GroupMonthlyAmount:  group.MonthlyAmount,
		GroupIsPublic:       group.IsPublic,
		GroupVotingEnabled:  group.VotingEnabled,
		Attendances:         make([]attendanceResp, 0, len(atts)),
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

func (h *MatchHandler) DiscoverMatches(w http.ResponseWriter, r *http.Request) {
	// Optional auth — only extract player if available
	var playerID *uuid.UUID
	player := middleware.PlayerFromCtx(r.Context())
	if player != nil {
		// Admins don't see discover matches
		if player.Role == db.PlayerRoleAdmin {
			renderJSON(w, http.StatusOK, []db.DiscoverMatch{})
			return
		}
		playerID = &player.ID
	}

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

	matches, err := h.Store.GetDiscoverMatches(r.Context(), playerID, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}

	// Compute spots_left = max_players - confirmed_count (null when unlimited)
	// to match v1 DiscoverMatchResponse shape; the frontend reads this directly.
	type discoverItem struct {
		db.DiscoverMatch
		SpotsLeft *int `json:"spots_left"`
	}
	out := make([]discoverItem, len(matches))
	for i, m := range matches {
		item := discoverItem{DiscoverMatch: m}
		if m.MaxPlayers != nil {
			left := *m.MaxPlayers - m.ConfirmedCount
			if left < 0 {
				left = 0
			}
			item.SpotsLeft = &left
		}
		out[i] = item
	}
	renderJSON(w, http.StatusOK, out)
}

func (h *MatchHandler) GetPublicMatch(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	matchWithGroup, err := h.Store.GetMatchByHashWithGroup(r.Context(), hash)
	if err != nil {
		renderError(w, err)
		return
	}

	atts, err := h.Store.GetAttendancesForMatch(r.Context(), matchWithGroup.ID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, buildMatchDetail(&matchWithGroup.Match, atts, groupFieldsFromMatch(matchWithGroup)))
}

func (h *MatchHandler) GetPublicMatchStats(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := h.Store.GetMatchByHash(r.Context(), hash)
	if err != nil {
		renderError(w, err)
		return
	}

	stats, err := h.Store.GetMatchPlayerStats(r.Context(), match.ID)
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

func (h *MatchHandler) listGroupMatches(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		if _, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID); err != nil {
			renderError(w, apierror.Forbidden("not a member"))
			return
		}
	}

	// Opportunistic status sync: matches the v1 behaviour where every call
	// to GET /groups/{id}/matches closes past matches, transitions today's
	// matches to in_progress (with "bola rolando" push), and triggers the
	// recurrence job when anything closed — so the user always sees a fresh
	// state without depending on the hourly cron.
	if h.pool != nil {
		closed, err := services.RunStatusSyncJob(r.Context(), h.pool)
		if err != nil {
			slog.Warn("listGroupMatches: status sync failed", "error", err)
		}
		if closed > 0 {
			if _, err := services.RunRecurrence(r.Context(), h.pool); err != nil {
				slog.Warn("listGroupMatches: recurrence failed", "error", err)
			}
		}
	}

	matches, err := h.Store.GetMatchesByGroup(r.Context(), groupID)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, enrichGroupMatches(matches))
}

func (h *MatchHandler) createMatch(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var req createMatchReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	// Validate all required fields first
	if req.MatchDate == "" || req.StartTime == "" || req.Location == "" {
		renderError(w, apierror.Unprocessable("match_date, start_time, and location are required"))
		return
	}

	// Validate location length
	if len(req.Location) < 2 || len(req.Location) > 200 {
		renderError(w, apierror.Unprocessable("location must be 2-200 characters"))
		return
	}

	// Validate players_per_team range
	if req.PlayersPerTeam != nil && (*req.PlayersPerTeam < 2 || *req.PlayersPerTeam > 15) {
		renderError(w, apierror.Unprocessable("players_per_team must be 2-15"))
		return
	}

	// Validate max_players range
	if req.MaxPlayers != nil && *req.MaxPlayers < 2 {
		renderError(w, apierror.Unprocessable("max_players must be at least 2"))
		return
	}

	// Now check authorization and group exists
	group, err := h.Store.GetGroupByID(r.Context(), groupID)
	if err != nil || group == nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can create matches"))
			return
		}
	}

	hash, err := generateHash()
	if err != nil {
		renderError(w, err)
		return
	}

	number, err := h.Store.NextMatchNumber(r.Context(), groupID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get next match number"))
		return
	}

	match, err := h.Store.CreateMatch(r.Context(), db.CreateMatchParams{
		GroupID:              groupID,
		Hash:                 hash,
		Number:               number,
		MatchDate:            req.MatchDate,
		StartTime:            req.StartTime,
		EndTime:              req.EndTime,
		Location:             req.Location,
		Address:              req.Address,
		CourtType:            req.CourtType,
		PlayersPerTeam:       req.PlayersPerTeam,
		MaxPlayers:           req.MaxPlayers,
		Notes:                req.Notes,
		CreatedByID:          player.ID,
		VoteOpenDelayMinutes: group.VoteOpenDelayMinutes,
		VoteDurationHours:    group.VoteDurationHours,
	})
	if err != nil {
		renderError(w, apierror.Internal("failed to create match"))
		return
	}

	// Add PENDING attendance for non-admin group members
	memberIDs, _ := h.Store.GetNonAdminMemberPlayerIDs(r.Context(), groupID)
	for _, pid := range memberIDs {
		_ = h.Store.SetAttendance(r.Context(), match.ID, pid, "pending")
	}

	renderJSON(w, http.StatusCreated, enrichOneMatch(match))
}

func (h *MatchHandler) getMatch(w http.ResponseWriter, r *http.Request) {
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
		if _, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID); err != nil {
			renderError(w, apierror.Forbidden("not a member"))
			return
		}
	}

	match, err := h.Store.GetMatchByID(r.Context(), matchID)
	if err != nil {
		renderError(w, err)
		return
	}
	if match.GroupID != groupID {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	atts, err := h.Store.GetAttendancesForMatch(r.Context(), matchID)
	if err != nil {
		renderError(w, err)
		return
	}

	// Fetch group to include its name/timezone/pricing/visibility — the
	// frontend MatchBannerCard reads match.group_timezone (formatMatchTimeRange)
	// and other group_* fields directly off the match payload.
	group := matchGroupFields{Timezone: "America/Sao_Paulo", IsPublic: true, VotingEnabled: true}
	if g, err := h.Store.GetGroupByID(r.Context(), match.GroupID); err == nil && g != nil {
		group.Name = g.Name
		group.Timezone = g.Timezone
		group.PerMatchAmount = g.PerMatchAmount
		group.MonthlyAmount = g.MonthlyAmount
		group.IsPublic = g.IsPublic
		group.VotingEnabled = g.VotingEnabled
	}

	renderJSON(w, http.StatusOK, buildMatchDetail(match, atts, group))
}

func (h *MatchHandler) updateMatch(w http.ResponseWriter, r *http.Request) {
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
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
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

	match, err := h.Store.UpdateMatch(r.Context(), matchID, db.UpdateMatchParams{
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
	renderJSON(w, http.StatusOK, enrichOneMatch(match))
}

func (h *MatchHandler) deleteMatch(w http.ResponseWriter, r *http.Request) {
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
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can delete matches"))
			return
		}
	}

	if err := h.Store.DeleteMatch(r.Context(), matchID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *MatchHandler) setAttendance(w http.ResponseWriter, r *http.Request) {
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
	callerMem, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
	if err != nil {
		renderError(w, apierror.Forbidden("not a member"))
		return
	}

	// Non-admins can only set their own attendance
	if callerMem.Role != db.GroupMemberRoleAdmin && req.PlayerID != player.ID {
		renderError(w, apierror.Forbidden("can only set own attendance"))
		return
	}

	match, err := h.Store.GetMatchByID(r.Context(), matchID)
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
		confirmed, _ := h.Store.CountAttendances(r.Context(), matchID, "confirmed")
		if confirmed >= *match.MaxPlayers {
			renderError(w, apierror.Forbidden("match is full"))
			return
		}
	}

	if err := h.Store.SetAttendance(r.Context(), matchID, req.PlayerID, req.Status); err != nil {
		renderError(w, err)
		return
	}

	// Build response
	atts, _ := h.Store.GetAttendancesForMatch(r.Context(), matchID)
	for _, a := range atts {
		if a.PlayerID == req.PlayerID {
			renderJSON(w, http.StatusOK, buildAttendanceResp(a))
			return
		}
	}
	renderJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (h *MatchHandler) UpsertPlayerStats(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	hash := chi.URLParam(r, "hash")

	match, err := h.Store.GetMatchByHash(r.Context(), hash)
	if err != nil {
		renderError(w, err)
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), match.GroupID, player.ID)
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
		if err := h.Store.UpsertMatchPlayerStat(r.Context(), match.ID, s.PlayerID, player.ID, s.Goals, s.Assists); err != nil {
			renderError(w, err)
			return
		}
	}

	stats, err := h.Store.GetMatchPlayerStats(r.Context(), match.ID)
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
