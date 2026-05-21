package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GenerateMCPToken creates a new raw token string (rachao_<32 hex bytes>),
// its SHA-256 hash, and the display prefix.
func GenerateMCPToken() (raw, hash, prefix string, err error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", "", "", fmt.Errorf("GenerateMCPToken: %w", e)
	}
	raw = "rachao_" + hex.EncodeToString(b)
	hash = HashToken(raw)
	prefix = raw[:min(len(raw), 14)] // "rachao_" + first 7 chars of hex
	return raw, hash, prefix, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type CreateMCPTokenParams struct {
	PlayerID  uuid.UUID
	Name      string
	TokenHash string
	Prefix    string
	ExpiresAt *time.Time
}

func CreateMCPToken(ctx context.Context, pool *pgxpool.Pool, p CreateMCPTokenParams) (*MCPToken, error) {
	var t MCPToken
	err := pool.QueryRow(ctx,
		`INSERT INTO mcp_tokens (player_id, name, token_hash, token_prefix, expires_at)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, player_id, name, token_hash, token_prefix, expires_at, created_at, last_used_at, revoked_at`,
		p.PlayerID, p.Name, p.TokenHash, p.Prefix, p.ExpiresAt,
	).Scan(&t.ID, &t.PlayerID, &t.Name, &t.TokenHash, &t.TokenPrefix,
		&t.ExpiresAt, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func ListMCPTokens(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]MCPToken, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, player_id, name, token_hash, token_prefix, expires_at, created_at, last_used_at, revoked_at
		 FROM mcp_tokens
		 WHERE player_id=$1 AND revoked_at IS NULL
		 ORDER BY created_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []MCPToken
	for rows.Next() {
		var t MCPToken
		if err := rows.Scan(&t.ID, &t.PlayerID, &t.Name, &t.TokenHash, &t.TokenPrefix,
			&t.ExpiresAt, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func GetMCPToken(ctx context.Context, pool *pgxpool.Pool, tokenID uuid.UUID) (*MCPToken, error) {
	var t MCPToken
	err := pool.QueryRow(ctx,
		`SELECT id, player_id, name, token_hash, token_prefix, expires_at, created_at, last_used_at, revoked_at
		 FROM mcp_tokens WHERE id=$1 AND revoked_at IS NULL`,
		tokenID,
	).Scan(&t.ID, &t.PlayerID, &t.Name, &t.TokenHash, &t.TokenPrefix,
		&t.ExpiresAt, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &t, nil
}

func RevokeMCPToken(ctx context.Context, pool *pgxpool.Pool, tokenID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE mcp_tokens SET revoked_at=NOW() WHERE id=$1`, tokenID,
	)
	return err
}
