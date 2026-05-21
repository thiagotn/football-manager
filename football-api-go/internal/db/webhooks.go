package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func IsWebhookEventProcessed(ctx context.Context, pool *pgxpool.Pool, eventID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM webhook_events WHERE event_id=$1)`, eventID,
	).Scan(&exists)
	return exists, err
}

func MarkWebhookEventProcessed(ctx context.Context, pool *pgxpool.Pool, eventID, eventType string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO webhook_events (event_id, event_type)
		 VALUES ($1, $2)
		 ON CONFLICT (event_id) DO NOTHING`,
		eventID, eventType,
	)
	return err
}
