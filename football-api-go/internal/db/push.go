package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PushSubscription struct {
	ID        int       `json:"id"` // SERIAL (int), not UUID
	PlayerID  uuid.UUID `json:"player_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	UserAgent *string   `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func UpsertPushSubscription(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, endpoint, p256dh, auth string, userAgent *string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO push_subscriptions (player_id, endpoint, p256dh, auth, user_agent)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (player_id, endpoint)
		 DO UPDATE SET p256dh=EXCLUDED.p256dh, auth=EXCLUDED.auth, user_agent=EXCLUDED.user_agent`,
		playerID, endpoint, p256dh, auth, userAgent,
	)
	return err
}

func DeletePushSubscriptions(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM push_subscriptions WHERE player_id=$1`, playerID,
	)
	return err
}

func GetPushSubscriptionsForPlayer(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]PushSubscription, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, player_id, endpoint, p256dh, auth, user_agent, created_at
		 FROM push_subscriptions WHERE player_id=$1`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.PlayerID, &s.Endpoint, &s.P256dh, &s.Auth, &s.UserAgent, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
