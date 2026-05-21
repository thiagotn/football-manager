package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func InsertAndroidBetaSignup(ctx context.Context, pool *pgxpool.Pool, email string, playerID *uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO android_beta_signups (google_email, player_id) VALUES ($1, $2)`,
		email, playerID,
	)
	return err
}
