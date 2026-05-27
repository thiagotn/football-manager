package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GroupMemberRole is the role of a player within a group.
type GroupMemberRole string

const (
	GroupMemberRoleAdmin  GroupMemberRole = "admin"
	GroupMemberRoleMember GroupMemberRole = "member"
)

// TeamSlot defines optional custom name/color for a team.
type TeamSlot struct {
	Color *string `json:"color"`
	Name  *string `json:"name"`
}

// Group represents a football group.
type Group struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Description          *string    `json:"description"`
	Slug                 string     `json:"slug"`
	PerMatchAmount       *float64   `json:"per_match_amount"`
	MonthlyAmount        *float64   `json:"monthly_amount"`
	RecurrenceEnabled    bool       `json:"recurrence_enabled"`
	IsPublic             bool       `json:"is_public"`
	VoteOpenDelayMinutes int        `json:"vote_open_delay_minutes"`
	VoteDurationHours    int        `json:"vote_duration_hours"`
	Timezone             string     `json:"timezone"`
	TeamSlots            []TeamSlot `json:"team_slots"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// GroupMember represents a player's membership in a group.
type GroupMember struct {
	ID         uuid.UUID       `json:"id"`
	GroupID    uuid.UUID       `json:"group_id"`
	PlayerID   uuid.UUID       `json:"player_id"`
	Role       GroupMemberRole `json:"role"`
	SkillStars int             `json:"skill_stars"`
	Position   string          `json:"position"`
	Nickname   *string         `json:"nickname"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// GroupMemberWithPlayer extends GroupMember with joined player data.
type GroupMemberWithPlayer struct {
	GroupMember
	PlayerName      string     `json:"player_name"`
	PlayerNickname  *string    `json:"player_nickname"`
	PlayerWhatsApp  string     `json:"player_whatsapp"`
	PlayerAvatarURL *string    `json:"player_avatar_url"`
	PlayerRole      PlayerRole `json:"player_role"`
}

const groupSelectCols = `
	g.id, g.name, g.description, g.slug,
	g.per_match_amount::FLOAT8,
	g.monthly_amount::FLOAT8,
	g.recurrence_enabled, g.is_public,
	g.vote_open_delay_minutes, g.vote_duration_hours,
	g.timezone, g.team_slots,
	g.created_at, g.updated_at`

const groupReturnCols = `
	id, name, description, slug,
	per_match_amount::FLOAT8,
	monthly_amount::FLOAT8,
	recurrence_enabled, is_public,
	vote_open_delay_minutes, vote_duration_hours,
	timezone, team_slots,
	created_at, updated_at`

func scanGroup(scanFn func(dest ...any) error) (*Group, error) {
	var g Group
	var slotsJSON []byte
	err := scanFn(
		&g.ID, &g.Name, &g.Description, &g.Slug,
		&g.PerMatchAmount, &g.MonthlyAmount,
		&g.RecurrenceEnabled, &g.IsPublic,
		&g.VoteOpenDelayMinutes, &g.VoteDurationHours,
		&g.Timezone, &slotsJSON,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if slotsJSON != nil {
		_ = json.Unmarshal(slotsJSON, &g.TeamSlots)
	}
	return &g, nil
}

func GetGroupsByPlayer(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, isAdmin bool) ([]Group, error) {
	var rows pgx.Rows
	var err error
	if isAdmin {
		rows, err = pool.Query(ctx, `SELECT `+groupSelectCols+` FROM groups g ORDER BY g.name`)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT `+groupSelectCols+`
			FROM groups g
			JOIN group_members gm ON gm.group_id = g.id
			WHERE gm.player_id = $1
			ORDER BY g.name`, playerID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]Group, 0)
	for rows.Next() {
		g, err := scanGroup(rows.Scan)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *g)
	}
	return groups, rows.Err()
}

func GetGroupByID(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (*Group, error) {
	row := pool.QueryRow(ctx, `SELECT `+groupSelectCols+` FROM groups g WHERE g.id = $1`, groupID)
	return scanGroup(row.Scan)
}

type CreateGroupParams struct {
	Name                 string
	Description          *string
	Slug                 string
	PerMatchAmount       *float64
	MonthlyAmount        *float64
	IsPublic             bool
	VoteOpenDelayMinutes int
	VoteDurationHours    int
	Timezone             string
}

func CreateGroup(ctx context.Context, pool *pgxpool.Pool, p CreateGroupParams) (*Group, error) {
	row := pool.QueryRow(ctx, `
		INSERT INTO groups
			(name, description, slug, per_match_amount, monthly_amount,
			 is_public, vote_open_delay_minutes, vote_duration_hours, timezone)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+groupReturnCols,
		p.Name, p.Description, p.Slug,
		p.PerMatchAmount, p.MonthlyAmount,
		p.IsPublic, p.VoteOpenDelayMinutes, p.VoteDurationHours, p.Timezone,
	)
	return scanGroup(row.Scan)
}

func UpdateGroupFull(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID, g *Group) (*Group, error) {
	var slotsJSON *[]byte
	if g.TeamSlots != nil {
		b, _ := json.Marshal(g.TeamSlots)
		slotsJSON = &b
	}
	row := pool.QueryRow(ctx, `
		UPDATE groups SET
			name=$1, description=$2,
			per_match_amount=$3, monthly_amount=$4,
			recurrence_enabled=$5, is_public=$6,
			vote_open_delay_minutes=$7, vote_duration_hours=$8,
			timezone=$9, team_slots=$10
		WHERE id=$11
		RETURNING `+groupReturnCols,
		g.Name, g.Description,
		g.PerMatchAmount, g.MonthlyAmount,
		g.RecurrenceEnabled, g.IsPublic,
		g.VoteOpenDelayMinutes, g.VoteDurationHours,
		g.Timezone, slotsJSON, groupID,
	)
	return scanGroup(row.Scan)
}

func DeleteGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM groups WHERE id = $1`, groupID)
	return err
}

func SlugExists(ctx context.Context, pool *pgxpool.Pool, slug string) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM groups WHERE slug=$1`, slug).Scan(&n)
	return n > 0, err
}

// --- group_members ---

func GetGroupMember(ctx context.Context, pool *pgxpool.Pool, groupID, playerID uuid.UUID) (*GroupMember, error) {
	var m GroupMember
	err := pool.QueryRow(ctx, `
		SELECT id, group_id, player_id, role::TEXT, skill_stars, position, nickname, created_at, updated_at
		FROM group_members
		WHERE group_id=$1 AND player_id=$2`, groupID, playerID).
		Scan(&m.ID, &m.GroupID, &m.PlayerID, &m.Role,
			&m.SkillStars, &m.Position, &m.Nickname,
			&m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return &m, err
}

func GetGroupMembers(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]GroupMemberWithPlayer, error) {
	// Super admins (role='admin') are excluded from group member listings.
	// See: feedback_super_admin_exclusion.md — admins must not appear in any business metric
	// or interaction surface (e.g. "confirmable absent members" in match details).
	rows, err := pool.Query(ctx, `
		SELECT
			gm.id, gm.group_id, gm.player_id, gm.role::TEXT,
			gm.skill_stars, gm.position, gm.nickname,
			gm.created_at, gm.updated_at,
			p.name, p.nickname, p.whatsapp, p.avatar_url, p.role::TEXT
		FROM group_members gm
		JOIN players p ON p.id = gm.player_id
		WHERE gm.group_id = $1
		  AND p.role != 'admin'
		ORDER BY p.name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []GroupMemberWithPlayer
	for rows.Next() {
		var m GroupMemberWithPlayer
		if err := rows.Scan(
			&m.ID, &m.GroupID, &m.PlayerID, &m.Role,
			&m.SkillStars, &m.Position, &m.Nickname,
			&m.CreatedAt, &m.UpdatedAt,
			&m.PlayerName, &m.PlayerNickname, &m.PlayerWhatsApp,
			&m.PlayerAvatarURL, &m.PlayerRole,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if members == nil {
		members = []GroupMemberWithPlayer{}
	}
	return members, rows.Err()
}

func AddGroupMember(ctx context.Context, pool *pgxpool.Pool, groupID, playerID uuid.UUID, role GroupMemberRole) (*GroupMember, error) {
	var m GroupMember
	err := pool.QueryRow(ctx, `
		INSERT INTO group_members (group_id, player_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, group_id, player_id, role::TEXT, skill_stars, position, nickname, created_at, updated_at`,
		groupID, playerID, string(role)).
		Scan(&m.ID, &m.GroupID, &m.PlayerID, &m.Role,
			&m.SkillStars, &m.Position, &m.Nickname,
			&m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type UpdateGroupMemberParams struct {
	Role       *GroupMemberRole
	SkillStars *int
	Position   *string
	Nickname   *string
}

func UpdateGroupMember(ctx context.Context, pool *pgxpool.Pool, groupID, playerID uuid.UUID, p UpdateGroupMemberParams) (*GroupMember, error) {
	// Fetch current, apply changes, write back
	m, err := GetGroupMember(ctx, pool, groupID, playerID)
	if err != nil {
		return nil, err
	}
	if p.Role != nil {
		m.Role = *p.Role
	}
	if p.SkillStars != nil {
		m.SkillStars = *p.SkillStars
	}
	if p.Position != nil {
		m.Position = *p.Position
	}
	if p.Nickname != nil {
		m.Nickname = p.Nickname
	}
	err = pool.QueryRow(ctx, `
		UPDATE group_members
		SET role=$1, skill_stars=$2, position=$3, nickname=$4
		WHERE group_id=$5 AND player_id=$6
		RETURNING id, group_id, player_id, role::TEXT, skill_stars, position, nickname, created_at, updated_at`,
		string(m.Role), m.SkillStars, m.Position, m.Nickname, groupID, playerID).
		Scan(&m.ID, &m.GroupID, &m.PlayerID, &m.Role,
			&m.SkillStars, &m.Position, &m.Nickname,
			&m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func RemoveGroupMember(ctx context.Context, pool *pgxpool.Pool, groupID, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM group_members WHERE group_id=$1 AND player_id=$2`, groupID, playerID)
	return err
}

func CountGroupAdminCount(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM group_members
		WHERE player_id=$1 AND role='admin'`, playerID).Scan(&n)
	return n, err
}

func CountGroupMembers(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id=$1`, groupID).Scan(&n)
	return n, err
}

func GetPlayerPlan(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (string, error) {
	var plan string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(plan, 'free')
		FROM player_subscriptions
		WHERE player_id=$1`, playerID).Scan(&plan)
	if err == pgx.ErrNoRows {
		return "free", nil
	}
	return plan, err
}

func EnsurePlayerSubscription(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO player_subscriptions (player_id, plan)
		VALUES ($1, 'free')
		ON CONFLICT (player_id) DO NOTHING`, playerID)
	return err
}

// PlanGroupLimit returns the max number of groups a player can admin.
func PlanGroupLimit(plan string) int {
	switch plan {
	case "basic":
		return 3
	case "pro":
		return 10
	default:
		return 1
	}
}

// PlanMembersLimit returns the max members per group (0 = unlimited).
func PlanMembersLimit(plan string) int {
	switch plan {
	case "basic":
		return 50
	case "pro":
		return 0 // unlimited
	default:
		return 30
	}
}
