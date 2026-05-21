package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Invite represents a group invite token.
type Invite struct {
	ID          uuid.UUID  `json:"id"`
	GroupID     uuid.UUID  `json:"group_id"`
	Token       string     `json:"token"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Used        bool       `json:"used"`
	CreatedByID uuid.UUID  `json:"created_by_id"`
	UsedByID    *uuid.UUID `json:"used_by_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

// InviteWithGroup extends Invite with the group name.
type InviteWithGroup struct {
	Invite
	GroupName string `json:"group_name"`
}

func CreateInvite(ctx context.Context, pool *pgxpool.Pool, groupID, createdByID uuid.UUID, token string, expiresAt time.Time) (*Invite, error) {
	var inv Invite
	err := pool.QueryRow(ctx, `
		INSERT INTO invite_tokens (group_id, token, expires_at, created_by_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, group_id, token, expires_at, used, created_by_id, used_by_id, created_at`,
		groupID, token, expiresAt, createdByID).
		Scan(&inv.ID, &inv.GroupID, &inv.Token, &inv.ExpiresAt,
			&inv.Used, &inv.CreatedByID, &inv.UsedByID, &inv.CreatedAt)
	return &inv, err
}

func GetInviteByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*InviteWithGroup, error) {
	var inv InviteWithGroup
	err := pool.QueryRow(ctx, `
		SELECT
			it.id, it.group_id, it.token, it.expires_at,
			it.used, it.created_by_id, it.used_by_id, it.created_at,
			g.name
		FROM invite_tokens it
		JOIN groups g ON g.id = it.group_id
		WHERE it.token = $1`, token).
		Scan(&inv.ID, &inv.GroupID, &inv.Token, &inv.ExpiresAt,
			&inv.Used, &inv.CreatedByID, &inv.UsedByID, &inv.CreatedAt,
			&inv.GroupName)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return &inv, err
}

func UseInvite(ctx context.Context, pool *pgxpool.Pool, token string, usedByID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE invite_tokens SET used=TRUE, used_by_id=$1
		WHERE token=$2`, usedByID, token)
	return err
}
