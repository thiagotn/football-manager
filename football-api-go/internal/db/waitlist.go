package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WaitlistEntry mirrors the match_waitlist table (migration 027).
type WaitlistEntry struct {
	ID            uuid.UUID  `json:"id"`
	MatchID       uuid.UUID  `json:"match_id"`
	PlayerID      uuid.UUID  `json:"player_id"`
	Intro         *string    `json:"intro"`
	Status        string     `json:"status"` // pending | accepted | rejected
	ReviewedBy    *uuid.UUID `json:"-"`
	ReviewedAt    *time.Time `json:"-"`
	CreatedAt     time.Time  `json:"created_at"`
	PlayerName    string     `json:"player_name"`
	PlayerNick    *string    `json:"player_nickname"`
}

// CreateWaitlistEntryParams collects the inputs for inserting a new entry.
type CreateWaitlistEntryParams struct {
	MatchID  uuid.UUID
	PlayerID uuid.UUID
	Intro    *string
}

// CreateWaitlistEntry inserts a new pending entry.
// Returns ErrConflict-like behaviour via the underlying UNIQUE constraint:
// callers should treat duplicate key errors as "already in waitlist".
func CreateWaitlistEntry(ctx context.Context, pool *pgxpool.Pool, p CreateWaitlistEntryParams) (*WaitlistEntry, error) {
	var e WaitlistEntry
	err := pool.QueryRow(ctx, `
		INSERT INTO match_waitlist (match_id, player_id, intro)
		VALUES ($1, $2, $3)
		RETURNING id, match_id, player_id, intro, status::TEXT, reviewed_by, reviewed_at, created_at`,
		p.MatchID, p.PlayerID, p.Intro,
	).Scan(&e.ID, &e.MatchID, &e.PlayerID, &e.Intro, &e.Status, &e.ReviewedBy, &e.ReviewedAt, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetWaitlistEntryForPlayer returns the player's entry for a specific match (if any).
func GetWaitlistEntryForPlayer(ctx context.Context, pool *pgxpool.Pool, matchID, playerID uuid.UUID) (*WaitlistEntry, error) {
	var e WaitlistEntry
	err := pool.QueryRow(ctx, `
		SELECT w.id, w.match_id, w.player_id, w.intro, w.status::TEXT,
		       w.reviewed_by, w.reviewed_at, w.created_at,
		       p.name, p.nickname
		FROM match_waitlist w
		JOIN players p ON p.id = w.player_id
		WHERE w.match_id = $1 AND w.player_id = $2`,
		matchID, playerID,
	).Scan(&e.ID, &e.MatchID, &e.PlayerID, &e.Intro, &e.Status,
		&e.ReviewedBy, &e.ReviewedAt, &e.CreatedAt,
		&e.PlayerName, &e.PlayerNick)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetWaitlistEntryByID returns a single entry by ID with player name+nickname joined.
func GetWaitlistEntryByID(ctx context.Context, pool *pgxpool.Pool, entryID uuid.UUID) (*WaitlistEntry, error) {
	var e WaitlistEntry
	err := pool.QueryRow(ctx, `
		SELECT w.id, w.match_id, w.player_id, w.intro, w.status::TEXT,
		       w.reviewed_by, w.reviewed_at, w.created_at,
		       p.name, p.nickname
		FROM match_waitlist w
		JOIN players p ON p.id = w.player_id
		WHERE w.id = $1`,
		entryID,
	).Scan(&e.ID, &e.MatchID, &e.PlayerID, &e.Intro, &e.Status,
		&e.ReviewedBy, &e.ReviewedAt, &e.CreatedAt,
		&e.PlayerName, &e.PlayerNick)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetPendingWaitlistForMatch returns all PENDING entries for one match, joined
// with the player table for display.
func GetPendingWaitlistForMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]WaitlistEntry, error) {
	rows, err := pool.Query(ctx, `
		SELECT w.id, w.match_id, w.player_id, w.intro, w.status::TEXT,
		       w.reviewed_by, w.reviewed_at, w.created_at,
		       p.name, p.nickname
		FROM match_waitlist w
		JOIN players p ON p.id = w.player_id
		WHERE w.match_id = $1 AND w.status = 'pending'
		ORDER BY w.created_at ASC`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WaitlistEntry
	for rows.Next() {
		var e WaitlistEntry
		if err := rows.Scan(&e.ID, &e.MatchID, &e.PlayerID, &e.Intro, &e.Status,
			&e.ReviewedBy, &e.ReviewedAt, &e.CreatedAt,
			&e.PlayerName, &e.PlayerNick); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateWaitlistEntryStatus marks an entry as accepted or rejected and records the reviewer.
func UpdateWaitlistEntryStatus(ctx context.Context, pool *pgxpool.Pool, entryID uuid.UUID, status string, reviewerID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE match_waitlist
		SET status = $2::waitlist_status, reviewed_by = $3, reviewed_at = now()
		WHERE id = $1`,
		entryID, status, reviewerID,
	)
	return err
}

// GetGroupAdminIDs returns the player IDs of admins of a group.
func GetGroupAdminIDs(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
		SELECT player_id FROM group_members
		WHERE group_id = $1 AND role = 'admin'`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
