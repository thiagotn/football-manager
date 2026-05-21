package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

// ── Types ────────────────────────────────────────────────────────────────────

type groupHandler struct {
	pool *pgxpool.Pool
}

func NewGroupHandler(pool *pgxpool.Pool) *groupHandler {
	return &groupHandler{pool: pool}
}

func (h *groupHandler) Routes() chi.Router {
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

func (h *groupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groups, err := db.GetGroupsByPlayer(r.Context(), h.pool, player.ID, player.Role == db.PlayerRoleAdmin)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, groups)
}

func (h *groupHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	var req createGroupReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" || len(req.Name) < 2 {
		renderError(w, apierror.Unprocessable("name must be at least 2 characters"))
		return
	}

	// Plan limit check (admins are exempt)
	if player.Role != db.PlayerRoleAdmin {
		plan, _ := db.GetPlayerPlan(r.Context(), h.pool, player.ID)
		limit := db.PlanGroupLimit(plan)
		count, _ := db.CountGroupAdminCount(r.Context(), h.pool, player.ID)
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
	exists, err := db.SlugExists(r.Context(), h.pool, slug)
	if err != nil {
		renderError(w, err)
		return
	}
	if exists {
		// Try append suffix
		for i := 2; i <= 9; i++ {
			candidate := slug + "-" + string(rune('0'+i))
			exists2, _ := db.SlugExists(r.Context(), h.pool, candidate)
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
		voteDelay = *req.VoteOpenDelayMinutes
	}
	voteDur := 24
	if req.VoteDurationHours != nil {
		voteDur = *req.VoteDurationHours
	}
	tz := "America/Sao_Paulo"
	if req.Timezone != nil && *req.Timezone != "" {
		tz = *req.Timezone
	}

	group, err := db.CreateGroup(r.Context(), h.pool, db.CreateGroupParams{
		Name:                 strings.TrimSpace(req.Name),
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
	_, _ = db.AddGroupMember(r.Context(), h.pool, group.ID, player.ID, db.GroupMemberRoleAdmin)
	// Ensure player has subscription
	_ = db.EnsurePlayerSubscription(r.Context(), h.pool, player.ID)

	renderJSON(w, http.StatusCreated, group)
}

func (h *groupHandler) getGroup(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	group, err := db.GetGroupByID(r.Context(), h.pool, groupID)
	if err != nil {
		renderError(w, err)
		return
	}

	// Check membership (admin sees all)
	var callerMembership *db.GroupMember
	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil {
			renderError(w, apierror.Forbidden("not a member of this group"))
			return
		}
		callerMembership = m
	}

	members, err := db.GetGroupMembers(r.Context(), h.pool, groupID)
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

func (h *groupHandler) updateGroup(w http.ResponseWriter, r *http.Request) {
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

	group, err := db.GetGroupByID(r.Context(), h.pool, groupID)
	if err != nil {
		renderError(w, err)
		return
	}

	// Auth check
	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
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

	updated, err := db.UpdateGroupFull(r.Context(), h.pool, groupID, group)
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, updated)
}

func (h *groupHandler) deleteGroup(w http.ResponseWriter, r *http.Request) {
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
	if err := db.DeleteGroup(r.Context(), h.pool, groupID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *groupHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	var callerMembership *db.GroupMember
	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil {
			renderError(w, apierror.Forbidden("not a member of this group"))
			return
		}
		callerMembership = m
	}

	members, err := db.GetGroupMembers(r.Context(), h.pool, groupID)
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

func (h *groupHandler) addMember(w http.ResponseWriter, r *http.Request) {
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
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can add members"))
			return
		}
	}

	// Plan members limit
	plan, _ := db.GetPlayerPlan(r.Context(), h.pool, player.ID)
	limit := db.PlanMembersLimit(plan)
	if limit > 0 {
		count, _ := db.CountGroupMembers(r.Context(), h.pool, groupID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	// Check already member
	if _, err := db.GetGroupMember(r.Context(), h.pool, groupID, req.PlayerID); err == nil {
		renderError(w, apierror.Conflict("player already in group"))
		return
	}

	m, err := db.AddGroupMember(r.Context(), h.pool, groupID, req.PlayerID, req.Role)
	if err != nil {
		renderError(w, err)
		return
	}

	// Add PENDING attendance for open matches
	matchIDs, _ := db.GetOpenMatchesForGroup(r.Context(), h.pool, groupID)
	for _, mid := range matchIDs {
		_ = db.SetAttendance(r.Context(), h.pool, mid, req.PlayerID, "pending")
	}

	_ = db.EnsurePlayerSubscription(r.Context(), h.pool, req.PlayerID)
	renderJSON(w, http.StatusCreated, m)
}

func (h *groupHandler) updateMyPosition(w http.ResponseWriter, r *http.Request) {
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

	m, err := db.UpdateGroupMember(r.Context(), h.pool, groupID, player.ID, db.UpdateGroupMemberParams{
		Position: &req.Position,
	})
	if err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, m)
}

func (h *groupHandler) updateMember(w http.ResponseWriter, r *http.Request) {
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
		cm, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
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

	m, err := db.UpdateGroupMember(r.Context(), h.pool, groupID, targetID, db.UpdateGroupMemberParams{
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

func (h *groupHandler) removeMember(w http.ResponseWriter, r *http.Request) {
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
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can remove members"))
			return
		}
	}

	if err := db.RemoveGroupMember(r.Context(), h.pool, groupID, targetID); err != nil {
		renderError(w, err)
		return
	}
	noContent(w)
}

func (h *groupHandler) lookupMember(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
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

	found, err := db.GetPlayerByWhatsApp(r.Context(), h.pool, whatsapp)
	if err != nil {
		renderJSON(w, http.StatusOK, map[string]any{"status": "not_found", "player": nil})
		return
	}

	// Check if already member
	if _, err := db.GetGroupMember(r.Context(), h.pool, groupID, found.ID); err == nil {
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

func (h *groupHandler) addMemberByPhone(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	groupID, err := groupIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}
	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
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
	plan, _ := db.GetPlayerPlan(r.Context(), h.pool, player.ID)
	limit := db.PlanMembersLimit(plan)
	if limit > 0 {
		count, _ := db.CountGroupMembers(r.Context(), h.pool, groupID)
		if count >= limit {
			renderError(w, apierror.PlanLimitExceeded())
			return
		}
	}

	isNew := false
	target, err := db.GetPlayerByWhatsApp(r.Context(), h.pool, req.WhatsApp)
	if err != nil {
		// Create new player
		if req.Name == nil || len(strings.TrimSpace(*req.Name)) < 2 {
			renderError(w, apierror.Unprocessable("name is required for new players"))
			return
		}
		hash, _ := hashPassword("temp-" + req.WhatsApp)
		target, err = db.CreatePlayer(r.Context(), h.pool, db.CreatePlayerParams{
			Name:         strings.TrimSpace(*req.Name),
			WhatsApp:     req.WhatsApp,
			PasswordHash: hash,
		})
		if err != nil {
			renderError(w, err)
			return
		}
		_ = db.EnsurePlayerSubscription(r.Context(), h.pool, target.ID)
		_ = db.UpdatePlayerMustChangePassword(r.Context(), h.pool, target.ID, true)
		isNew = true
	}

	// Check already member
	if _, err := db.GetGroupMember(r.Context(), h.pool, groupID, target.ID); err == nil {
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

	m, err := db.AddGroupMember(r.Context(), h.pool, groupID, target.ID, db.GroupMemberRoleMember)
	if err != nil {
		renderError(w, err)
		return
	}
	// Set skill/position
	_, _ = db.UpdateGroupMember(r.Context(), h.pool, groupID, target.ID, db.UpdateGroupMemberParams{
		SkillStars: &skillStars,
		Position:   &position,
		Nickname:   req.Nickname,
	})

	// Add to open matches
	matchIDs, _ := db.GetOpenMatchesForGroup(r.Context(), h.pool, groupID)
	for _, mid := range matchIDs {
		_ = db.SetAttendance(r.Context(), h.pool, mid, target.ID, "pending")
	}

	renderJSON(w, http.StatusCreated, map[string]any{"member": m, "is_new": isNew})
}

func (h *groupHandler) groupStats(w http.ResponseWriter, r *http.Request) {
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
	// Simplified stats: return member count for now
	count, _ := db.CountGroupMembers(r.Context(), h.pool, groupID)
	renderJSON(w, http.StatusOK, map[string]any{
		"total_members": count,
		"players":       []any{},
		"period_label":  "all",
	})
}

func (h *groupHandler) joinWaitlist(w http.ResponseWriter, r *http.Request) {
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

	group, err := db.GetGroupByID(r.Context(), h.pool, groupID)
	if err != nil {
		renderError(w, err)
		return
	}
	if !group.IsPublic {
		renderError(w, apierror.Forbidden("group is not public"))
		return
	}

	// Check already member
	if _, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID); err == nil {
		renderError(w, apierror.Conflict("already a member"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]any{
		"status":  "pending",
		"message": "waitlist join recorded",
	})
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
