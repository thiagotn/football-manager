package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)


type GroupStore interface {
	GetGroupsByPlayer(ctx context.Context, playerID uuid.UUID, isAdmin bool) ([]db.Group, error)
	GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error)
	CreateGroup(ctx context.Context, p db.CreateGroupParams) (*db.Group, error)
	UpdateGroupFull(ctx context.Context, groupID uuid.UUID, g *db.Group) (*db.Group, error)
	DeleteGroup(ctx context.Context, groupID uuid.UUID) error
	SlugExists(ctx context.Context, slug string) (bool, error)
	CountGroupAdminCount(ctx context.Context, playerID uuid.UUID) (int, error)
	GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]db.GroupMemberWithPlayer, error)
	AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error)
	UpdateGroupMember(ctx context.Context, groupID, playerID uuid.UUID, p db.UpdateGroupMemberParams) (*db.GroupMember, error)
	RemoveGroupMember(ctx context.Context, groupID, playerID uuid.UUID) error
	CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error)
	GetGroupMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	GetNonAdminMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	GetPlayerPlan(ctx context.Context, playerID uuid.UUID) (string, error)
	EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error
	GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error)
	CreatePlayer(ctx context.Context, args db.CreatePlayerArgs) (*db.Player, error)
	UpdatePlayerMustChangePassword(ctx context.Context, id uuid.UUID, val bool) error
	GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error
	EnsureMemberInCurrentPeriod(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error
	GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error)
}

type pgGroupStore struct {
	pool *pgxpool.Pool
}

func (s *pgGroupStore) GetGroupsByPlayer(ctx context.Context, playerID uuid.UUID, isAdmin bool) ([]db.Group, error) {
	return db.GetGroupsByPlayer(ctx, s.pool, playerID, isAdmin)
}
func (s *pgGroupStore) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error) {
	return db.GetGroupByID(ctx, s.pool, groupID)
}
func (s *pgGroupStore) CreateGroup(ctx context.Context, p db.CreateGroupParams) (*db.Group, error) {
	return db.CreateGroup(ctx, s.pool, p)
}
func (s *pgGroupStore) UpdateGroupFull(ctx context.Context, groupID uuid.UUID, g *db.Group) (*db.Group, error) {
	return db.UpdateGroupFull(ctx, s.pool, groupID, g)
}
func (s *pgGroupStore) DeleteGroup(ctx context.Context, groupID uuid.UUID) error {
	return db.DeleteGroup(ctx, s.pool, groupID)
}
func (s *pgGroupStore) SlugExists(ctx context.Context, slug string) (bool, error) {
	return db.SlugExists(ctx, s.pool, slug)
}
func (s *pgGroupStore) CountGroupAdminCount(ctx context.Context, playerID uuid.UUID) (int, error) {
	return db.CountGroupAdminCount(ctx, s.pool, playerID)
}
func (s *pgGroupStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return db.GetGroupMember(ctx, s.pool, groupID, playerID)
}
func (s *pgGroupStore) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]db.GroupMemberWithPlayer, error) {
	return db.GetGroupMembers(ctx, s.pool, groupID)
}
func (s *pgGroupStore) AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error) {
	return db.AddGroupMember(ctx, s.pool, groupID, playerID, role)
}
func (s *pgGroupStore) UpdateGroupMember(ctx context.Context, groupID, playerID uuid.UUID, p db.UpdateGroupMemberParams) (*db.GroupMember, error) {
	return db.UpdateGroupMember(ctx, s.pool, groupID, playerID, p)
}
func (s *pgGroupStore) RemoveGroupMember(ctx context.Context, groupID, playerID uuid.UUID) error {
	return db.RemoveGroupMember(ctx, s.pool, groupID, playerID)
}
func (s *pgGroupStore) CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	return db.CountGroupMembers(ctx, s.pool, groupID)
}
func (s *pgGroupStore) GetGroupMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return db.GetGroupMemberPlayerIDs(ctx, s.pool, groupID)
}
func (s *pgGroupStore) GetNonAdminMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return db.GetNonAdminMemberPlayerIDs(ctx, s.pool, groupID)
}
func (s *pgGroupStore) GetPlayerPlan(ctx context.Context, playerID uuid.UUID) (string, error) {
	return db.GetPlayerPlan(ctx, s.pool, playerID)
}
func (s *pgGroupStore) EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error {
	return db.EnsurePlayerSubscription(ctx, s.pool, playerID)
}
func (s *pgGroupStore) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	return db.GetPlayerByWhatsApp(ctx, s.pool, whatsapp)
}
func (s *pgGroupStore) CreatePlayer(ctx context.Context, args db.CreatePlayerArgs) (*db.Player, error) {
	return db.CreatePlayer(ctx, s.pool, args)
}
func (s *pgGroupStore) UpdatePlayerMustChangePassword(ctx context.Context, id uuid.UUID, val bool) error {
	return db.UpdatePlayerMustChangePassword(ctx, s.pool, id, val)
}
func (s *pgGroupStore) GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return db.GetOpenMatchesForGroup(ctx, s.pool, groupID)
}
func (s *pgGroupStore) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	return db.SetAttendance(ctx, s.pool, matchID, playerID, status)
}
func (s *pgGroupStore) EnsureMemberInCurrentPeriod(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error {
	return db.EnsureMemberInCurrentPeriod(ctx, s.pool, groupID, playerID, playerName)
}
func (s *pgGroupStore) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	return db.GetPlayerByID(ctx, s.pool, playerID)
}

// ── Types ────────────────────────────────────────────────────────────────────

type GroupHandler struct {
	Store GroupStore
}

func NewGroupHandler(pool *pgxpool.Pool) *GroupHandler {
	return &GroupHandler{Store: &pgGroupStore{pool: pool}}
}

func (h *GroupHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.listGroups)
	r.Post("/", h.createGroup)
	r.Route("/{groupID}", func(r chi.Router) {
		r.Get("/", h.getGroup)
		r.Patch("/", h.updateGroup)
		r.Delete("/", h.deleteGroup)
		r.Get("/members", h.listMembers)
		r.Post("/members", h.addMember)
		r.Patch("/members/me", h.updateMyPosition)
		r.Get("/members/lookup", h.lookupMember)
		r.Post("/members/by-phone", h.addMemberByPhone)
		r.Patch("/members/{playerID}", h.updateMember)
		r.Delete("/members/{playerID}", h.removeMember)
		r.Get("/stats", h.groupStats)
		r.Get("/waitlist", h.listWaitlist)
		r.Post("/waitlist", h.joinWaitlist)
	})
	return r
}

// ── Request / Response types ─────────────────────────────────────────────────

type createGroupReq struct {
	Name                 string   `json:"name"`
	Description          *string  `json:"description"`
	Slug                 *string  `json:"slug"`
	PerMatchAmount       *float64 `json:"per_match_amount"`
	MonthlyAmount        *float64 `json:"monthly_amount"`
	IsPublic             *bool    `json:"is_public"`
	VoteOpenDelayMinutes *int     `json:"vote_open_delay_minutes"`
	VoteDurationHours    *int     `json:"vote_duration_hours"`
	Timezone             *string  `json:"timezone"`
}

type updateGroupReq struct {
	Name                 *string       `json:"name"`
	Description          *string       `json:"description"`
	PerMatchAmount       *float64      `json:"per_match_amount"`
	MonthlyAmount        *float64      `json:"monthly_amount"`
	RecurrenceEnabled    *bool         `json:"recurrence_enabled"`
	IsPublic             *bool         `json:"is_public"`
	VoteOpenDelayMinutes *int          `json:"vote_open_delay_minutes"`
	VoteDurationHours    *int          `json:"vote_duration_hours"`
	Timezone             *string       `json:"timezone"`
	TeamSlots            []db.TeamSlot `json:"team_slots"`
}

type addMemberReq struct {
	PlayerID uuid.UUID          `json:"player_id"`
	Role     db.GroupMemberRole `json:"role"`
}

type updateMemberReq struct {
	Role       *db.GroupMemberRole `json:"role"`
	SkillStars *int                `json:"skill_stars"`
	Position   *string             `json:"position"`
	Nickname   *string             `json:"nickname"`
}

type selfPositionReq struct {
	Position string `json:"position"`
}

type addMemberByPhoneReq struct {
	WhatsApp   string  `json:"whatsapp"`
	Name       *string `json:"name"`
	Nickname   *string `json:"nickname"`
	SkillStars *int    `json:"skill_stars"`
	Position   *string `json:"position"`
}

type waitlistReq struct {
	Agreed bool    `json:"agreed"`
	Intro  *string `json:"intro"`
}

type memberPlayerView struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Nickname  *string   `json:"nickname"`
	Role      string    `json:"role"`
	WhatsApp  *string   `json:"whatsapp,omitempty"`
	AvatarURL *string   `json:"avatar_url"`
}

type memberResponse struct {
	ID         uuid.UUID        `json:"id"`
	Player     memberPlayerView `json:"player"`
	Role       string           `json:"role"`
	SkillStars *int             `json:"skill_stars,omitempty"`
	Position   *string          `json:"position,omitempty"`
	Nickname   *string          `json:"nickname"`
	CreatedAt  interface{}      `json:"created_at"`
}

type groupDetailResponse struct {
	db.Group
	Members      []memberResponse `json:"members"`
	TotalMembers int              `json:"total_members"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func slugify(s string) string {
	var b strings.Builder
	prevHyphen := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	return slug
}

var positionRe = regexp.MustCompile(`^(gk|zag|lat|mei|ata)$`)

func buildMemberResponse(m db.GroupMemberWithPlayer, isGroupAdmin bool) memberResponse {
	player := memberPlayerView{
		ID:        m.PlayerID,
		Name:      m.PlayerName,
		Nickname:  m.PlayerNickname,
		Role:      string(m.PlayerRole),
		AvatarURL: m.PlayerAvatarURL,
	}
	if isGroupAdmin {
		player.WhatsApp = &m.PlayerWhatsApp
	}
	skill := m.SkillStars
	pos := m.Position
	var skillPtr *int
	var posPtr *string
	if isGroupAdmin {
		skillPtr = &skill
		posPtr = &pos
	}
	return memberResponse{
		ID:         m.ID,
		Player:     player,
		Role:       string(m.Role),
		SkillStars: skillPtr,
		Position:   posPtr,
		Nickname:   m.Nickname,
		CreatedAt:  m.CreatedAt,
	}
}

func groupIDParam(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "groupID"))
}

func playerIDParam(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "playerID"))
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (h *GroupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groups, err := h.Store.GetGroupsByPlayer(r.Context(), player.ID, player.Role == db.PlayerRoleAdmin)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	var req createGroupReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) < 2 || len(name) > 100 {
		renderError(w, apierror.Unprocessable("name must be 2-100 characters"))
		return
	}

	// Plan limit check (admins are exempt)
	if player.Role != db.PlayerRoleAdmin {
		plan, _ := h.Store.GetPlayerPlan(r.Context(), player.ID)
		limit := db.PlanGroupLimit(plan)
		count, _ := h.Store.CountGroupAdminCount(r.Context(), player.ID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	slug := req.Name
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		slug = *req.Slug
	}
	slug = slugify(slug)

	// Ensure unique slug
	exists, err := h.Store.SlugExists(r.Context(), slug)
	if err != nil {
		renderError(w, err)
		return
	}
	if exists {
		// Try append suffix
		for i := 2; i <= 9; i++ {
			candidate := slug + "-" + string(rune('0'+i))
			exists2, _ := h.Store.SlugExists(r.Context(), candidate)
			if !exists2 {
				slug = candidate
				exists = false
				break
			}
		}
		if exists {
			renderError(w, apierror.Conflict("slug already taken"))
			return
		}
	}

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	voteDelay := 20
	if req.VoteOpenDelayMinutes != nil {
		if *req.VoteOpenDelayMinutes < 0 || *req.VoteOpenDelayMinutes > 120 {
			renderError(w, apierror.Unprocessable("vote_open_delay_minutes must be 0-120"))
			return
		}
		voteDelay = *req.VoteOpenDelayMinutes
	}
	voteDur := 24
	if req.VoteDurationHours != nil {
		if *req.VoteDurationHours < 2 || *req.VoteDurationHours > 72 {
			renderError(w, apierror.Unprocessable("vote_duration_hours must be 2-72"))
			return
		}
		voteDur = *req.VoteDurationHours
	}
	tz := "America/Sao_Paulo"
	if req.Timezone != nil && *req.Timezone != "" {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			renderError(w, apierror.Unprocessable("invalid timezone"))
			return
		}
		tz = *req.Timezone
	}

	group, err := h.Store.CreateGroup(r.Context(), db.CreateGroupParams{
		Name:                 name,
		Description:          req.Description,
		Slug:                 slug,
		PerMatchAmount:       req.PerMatchAmount,
		MonthlyAmount:        req.MonthlyAmount,
		IsPublic:             isPublic,
		VoteOpenDelayMinutes: voteDelay,
		VoteDurationHours:    voteDur,
		Timezone:             tz,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	// Creator becomes group admin
	_, _ = h.Store.AddGroupMember(r.Context(), group.ID, player.ID, db.GroupMemberRoleAdmin)
	// Ensure player has subscription
	_ = h.Store.EnsurePlayerSubscription(r.Context(), player.ID)

	renderJSON(w, http.StatusCreated, group)
}

func (h *GroupHandler) getGroup(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	group, err := h.Store.GetGroupByID(r.Context(), groupID)
	if err == db.ErrNotFound {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch group"))
		return
	}

	// Check membership (admin sees all)
	var callerMembership *db.GroupMember
	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil {
			renderError(w, apierror.Forbidden("not a member of this group"))
			return
		}
		callerMembership = m
	}

	members, err := h.Store.GetGroupMembers(r.Context(), groupID)
	if err != nil {
		renderError(w, err)
		return
	}

	isGroupAdmin := player.Role == db.PlayerRoleAdmin ||
		(callerMembership != nil && callerMembership.Role == db.GroupMemberRoleAdmin)

	memberResp := make([]memberResponse, 0, len(members))
	for _, m := range members {
		memberResp = append(memberResp, buildMemberResponse(m, isGroupAdmin))
	}

	renderJSON(w, http.StatusOK, groupDetailResponse{
		Group:        *group,
		Members:      memberResp,
		TotalMembers: len(members),
	})
}

func (h *GroupHandler) updateGroup(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var req updateGroupReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	group, err := h.Store.GetGroupByID(r.Context(), groupID)
	if err != nil {
		renderError(w, err)
		return
	}

	// Auth check
	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can update"))
			return
		}
	}

	// Apply changes
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = req.Description
	}
	if req.PerMatchAmount != nil {
		group.PerMatchAmount = req.PerMatchAmount
	}
	if req.MonthlyAmount != nil {
		group.MonthlyAmount = req.MonthlyAmount
	}
	if req.RecurrenceEnabled != nil {
		group.RecurrenceEnabled = *req.RecurrenceEnabled
	}
	if req.IsPublic != nil {
		group.IsPublic = *req.IsPublic
	}
	if req.VoteOpenDelayMinutes != nil {
		group.VoteOpenDelayMinutes = *req.VoteOpenDelayMinutes
	}
	if req.VoteDurationHours != nil {
		group.VoteDurationHours = *req.VoteDurationHours
	}
	if req.Timezone != nil {
		group.Timezone = *req.Timezone
	}
	if req.TeamSlots != nil {
		group.TeamSlots = req.TeamSlots
	}

	updated, err := h.Store.UpdateGroupFull(r.Context(), groupID, group)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, updated)
}

func (h *GroupHandler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin access required"))
		return
	}
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	if err := h.Store.DeleteGroup(r.Context(), groupID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *GroupHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var callerMembership *db.GroupMember
	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil {
			renderError(w, apierror.Forbidden("not a member of this group"))
			return
		}
		callerMembership = m
	}

	members, err := h.Store.GetGroupMembers(r.Context(), groupID)
	if err != nil {
		renderError(w, err)
		return
	}

	isGroupAdmin := player.Role == db.PlayerRoleAdmin ||
		(callerMembership != nil && callerMembership.Role == db.GroupMemberRoleAdmin)

	resp := make([]memberResponse, 0, len(members))
	for _, m := range members {
		resp = append(resp, buildMemberResponse(m, isGroupAdmin))
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *GroupHandler) addMember(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var req addMemberReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if req.Role == "" {
		req.Role = db.GroupMemberRoleMember
	}

	// Auth check
	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can add members"))
			return
		}
	}

	// Plan members limit
	plan, _ := h.Store.GetPlayerPlan(r.Context(), player.ID)
	limit := db.PlanMembersLimit(plan)
	if limit > 0 {
		count, _ := h.Store.CountGroupMembers(r.Context(), groupID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	// Check already member
	if _, err := h.Store.GetGroupMember(r.Context(), groupID, req.PlayerID); err == nil {
		renderError(w, apierror.Conflict("player already in group"))
		return
	}

	m, err := h.Store.AddGroupMember(r.Context(), groupID, req.PlayerID, req.Role)
	if err != nil {
		renderError(w, err)
		return
	}

	// Add PENDING attendance for open matches
	matchIDs, _ := h.Store.GetOpenMatchesForGroup(r.Context(), groupID)
	for _, mid := range matchIDs {
		_ = h.Store.SetAttendance(r.Context(), mid, req.PlayerID, "pending")
	}

	_ = h.Store.EnsurePlayerSubscription(r.Context(), req.PlayerID)

	// Ensure member appears in current finance period
	if playerInfo, err := h.Store.GetPlayerByID(r.Context(), req.PlayerID); err == nil {
		playerDisplayName := playerInfo.Name
		if playerInfo.Nickname != nil && *playerInfo.Nickname != "" {
			playerDisplayName = *playerInfo.Nickname
		}
		_ = h.Store.EnsureMemberInCurrentPeriod(r.Context(), groupID, req.PlayerID, playerDisplayName)
	}

	renderJSON(w, http.StatusCreated, m)
}

func (h *GroupHandler) updateMyPosition(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var req selfPositionReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if !positionRe.MatchString(req.Position) {
		renderError(w, apierror.Unprocessable("invalid position"))
		return
	}

	m, err := h.Store.UpdateGroupMember(r.Context(), groupID, player.ID, db.UpdateGroupMemberParams{
		Position: &req.Position,
	})
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, m)
}

func (h *GroupHandler) updateMember(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	targetID, err := playerIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	var req updateMemberReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	// Determine caller role
	isAdmin := player.Role == db.PlayerRoleAdmin
	var callerRole db.GroupMemberRole
	if !isAdmin {
		cm, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil {
			renderError(w, apierror.Forbidden("not a member"))
			return
		}
		callerRole = cm.Role
	}

	// Non-admins can only update their own nickname
	isSelf := player.ID == targetID
	if !isAdmin && callerRole != db.GroupMemberRoleAdmin {
		if !isSelf {
			renderError(w, apierror.Forbidden("can only update own profile"))
			return
		}
		// Only nickname allowed for self-service
		req.Role = nil
		req.SkillStars = nil
		req.Position = nil
	}

	if req.Position != nil && !positionRe.MatchString(*req.Position) {
		renderError(w, apierror.Unprocessable("invalid position"))
		return
	}

	m, err := h.Store.UpdateGroupMember(r.Context(), groupID, targetID, db.UpdateGroupMemberParams{
		Role:       req.Role,
		SkillStars: req.SkillStars,
		Position:   req.Position,
		Nickname:   req.Nickname,
	})
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, m)
}

func (h *GroupHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	targetID, err := playerIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can remove members"))
			return
		}
	}

	if err := h.Store.RemoveGroupMember(r.Context(), groupID, targetID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *GroupHandler) lookupMember(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("admin access required"))
			return
		}
	}
	whatsapp := r.URL.Query().Get("whatsapp")
	if whatsapp == "" {
		renderError(w, apierror.Unprocessable("whatsapp is required"))
		return
	}
	whatsapp = normalizePhone(whatsapp)

	found, err := h.Store.GetPlayerByWhatsApp(r.Context(), whatsapp)
	if err != nil {
		renderJSON(w, http.StatusOK, map[string]any{"status": "not_found", "player": nil})
		return
	}

	// Check if already member
	if _, err := h.Store.GetGroupMember(r.Context(), groupID, found.ID); err == nil {
		renderJSON(w, http.StatusOK, map[string]any{
			"status": "already_member",
			"player": map[string]any{
				"id": found.ID, "name": found.Name,
				"nickname": found.Nickname, "avatar_url": found.AvatarURL,
			},
		})
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"status": "found",
		"player": map[string]any{
			"id": found.ID, "name": found.Name,
			"nickname": found.Nickname, "avatar_url": found.AvatarURL,
		},
	})
}

func (h *GroupHandler) addMemberByPhone(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	if player.Role != db.PlayerRoleAdmin {
		m, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("admin access required"))
			return
		}
	}

	var req addMemberByPhoneReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	req.WhatsApp = normalizePhone(req.WhatsApp)

	// Plan limit
	plan, _ := h.Store.GetPlayerPlan(r.Context(), player.ID)
	limit := db.PlanMembersLimit(plan)
	if limit > 0 {
		count, _ := h.Store.CountGroupMembers(r.Context(), groupID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	isNew := false
	target, err := h.Store.GetPlayerByWhatsApp(r.Context(), req.WhatsApp)
	if err != nil {
		// Create new player
		if req.Name == nil || len(strings.TrimSpace(*req.Name)) < 2 {
			renderError(w, apierror.Unprocessable("name is required for new players"))
			return
		}
		hash, _ := hashPassword("temp-" + req.WhatsApp)
		target, err = h.Store.CreatePlayer(r.Context(), db.CreatePlayerParams{
			Name:         strings.TrimSpace(*req.Name),
			WhatsApp:     req.WhatsApp,
			PasswordHash: hash,
		})
		if err != nil {
			renderError(w, err)
			return
		}
		_ = h.Store.EnsurePlayerSubscription(r.Context(), target.ID)
		_ = h.Store.UpdatePlayerMustChangePassword(r.Context(), target.ID, true)
		isNew = true
	}

	// Check already member
	if _, err := h.Store.GetGroupMember(r.Context(), groupID, target.ID); err == nil {
		renderError(w, apierror.Conflict("player already in group"))
		return
	}

	skillStars := 2
	if req.SkillStars != nil {
		skillStars = *req.SkillStars
	}
	position := "mei"
	if req.Position != nil && positionRe.MatchString(*req.Position) {
		position = *req.Position
	}

	m, err := h.Store.AddGroupMember(r.Context(), groupID, target.ID, db.GroupMemberRoleMember)
	if err != nil {
		renderError(w, err)
		return
	}
	// Set skill/position
	_, _ = h.Store.UpdateGroupMember(r.Context(), groupID, target.ID, db.UpdateGroupMemberParams{
		SkillStars: &skillStars,
		Position:   &position,
		Nickname:   req.Nickname,
	})

	// Add to open matches
	matchIDs, _ := h.Store.GetOpenMatchesForGroup(r.Context(), groupID)
	for _, mid := range matchIDs {
		_ = h.Store.SetAttendance(r.Context(), mid, target.ID, "pending")
	}

	// Ensure member appears in current finance period
	playerDisplayName := target.Name
	if target.Nickname != nil && *target.Nickname != "" {
		playerDisplayName = *target.Nickname
	}
	_ = h.Store.EnsureMemberInCurrentPeriod(r.Context(), groupID, target.ID, playerDisplayName)

	renderJSON(w, http.StatusCreated, map[string]any{"member": m, "is_new": isNew})
}

func (h *GroupHandler) groupStats(w http.ResponseWriter, r *http.Request) {
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
	// Simplified stats: return member count for now
	count, _ := h.Store.CountGroupMembers(r.Context(), groupID)
	renderJSON(w, http.StatusOK, map[string]any{
		"total_members": count,
		"players":       []any{},
		"period_label":  "all",
	})
}

func (h *GroupHandler) joinWaitlist(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var req waitlistReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if !req.Agreed {
		renderError(w, apierror.Forbidden("must agree to join waitlist"))
		return
	}

	group, err := h.Store.GetGroupByID(r.Context(), groupID)
	if err != nil {
		renderError(w, err)
		return
	}
	if !group.IsPublic {
		renderError(w, apierror.Forbidden("group is not public"))
		return
	}

	// Check already member
	if _, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID); err == nil {
		renderError(w, apierror.Conflict("already a member"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]any{
		"status":  "pending",
		"message": "waitlist join recorded",
	})
}

func (h *GroupHandler) listWaitlist(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	// Check if caller is admin
	isAdmin := player.Role == db.PlayerRoleAdmin
	if !isAdmin {
		member, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
		if err != nil || member.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can list waitlist"))
			return
		}
	}

	// Get open matches for the group
	matchIDs, err := h.Store.GetOpenMatchesForGroup(r.Context(), groupID)
	if err != nil || len(matchIDs) == 0 {
		renderJSON(w, http.StatusOK, []any{})
		return
	}

	// Return empty list for now (waitlist data model not fully implemented in Go API)
	renderJSON(w, http.StatusOK, []any{})
}

// normalizePhone strips common formatting from a phone number.
func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r == '+' || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) > 0 && s[0] != '+' {
		s = "+" + s
	}
	return s
}
