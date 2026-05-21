// Package db contains hand-crafted query functions for Phase 1.
// These will be replaced by sqlc-generated code when `make generate` runs.
package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlayerRole mirrors the player_role PostgreSQL enum.
type PlayerRole string

const (
	PlayerRoleAdmin  PlayerRole = "admin"
	PlayerRolePlayer PlayerRole = "player"
)

// Player mirrors the players table.
type Player struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Nickname           *string    `json:"nickname"`
	WhatsApp           string     `json:"whatsapp"`
	PasswordHash       string     `json:"-"`
	Role               PlayerRole `json:"role"`
	Active             bool       `json:"active"`
	MustChangePassword bool       `json:"must_change_password"`
	AvatarURL          *string    `json:"avatar_url"`
	ChatEnabled        bool       `json:"chat_enabled"`
	ChatReqCount       int32      `json:"-"`
	ChatReqWindow      *time.Time `json:"-"`
	ApiV2Enabled       bool       `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// RefreshToken mirrors the refresh_tokens table.
type RefreshToken struct {
	ID        uuid.UUID
	PlayerID  uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// MCPToken mirrors the mcp_tokens table.
type MCPToken struct {
	ID          uuid.UUID
	PlayerID    uuid.UUID
	Name        string
	TokenHash   string
	TokenPrefix string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
}

var ErrNotFound = pgx.ErrNoRows

// PlayerSelectCols is the column list for SELECT queries on the players table.
const PlayerSelectCols = `
	id, name, nickname, whatsapp, password_hash,
	role, active, must_change_password, avatar_url,
	chat_enabled, chat_req_count, chat_req_window,
	api_v2_enabled, created_at, updated_at`

// ScanPlayer scans player fields from any pgx scan function (Row or Rows).
func ScanPlayer(scanFn func(dest ...any) error) (*Player, error) {
	var p Player
	err := scanFn(
		&p.ID, &p.Name, &p.Nickname, &p.WhatsApp, &p.PasswordHash,
		&p.Role, &p.Active, &p.MustChangePassword, &p.AvatarURL,
		&p.ChatEnabled, &p.ChatReqCount, &p.ChatReqWindow,
		&p.ApiV2Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// scanPlayer wraps ScanPlayer for the legacy pgx.Row interface used in auth queries.
func scanPlayer(row pgx.Row) (*Player, error) {
	return ScanPlayer(row.Scan)
}

// CreatePlayerParams is the canonical params type for creating players.
// (Alias of CreatePlayerArgs for backward compat.)
type CreatePlayerParams = CreatePlayerArgs

const playerColumns = `
	id, name, nickname, whatsapp, password_hash,
	role, active, must_change_password, avatar_url,
	chat_enabled, chat_req_count, chat_req_window,
	api_v2_enabled, created_at, updated_at`

// GetPlayerByWhatsApp fetches an active player by their WhatsApp number.
func GetPlayerByWhatsApp(ctx context.Context, pool *pgxpool.Pool, whatsapp string) (*Player, error) {
	row := pool.QueryRow(ctx,
		`SELECT`+playerColumns+` FROM players WHERE whatsapp = $1 AND active = true`,
		whatsapp,
	)
	p, err := scanPlayer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPlayerByWhatsApp: %w", err)
	}
	return p, nil
}

// GetPlayerByID fetches an active player by UUID.
func GetPlayerByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Player, error) {
	row := pool.QueryRow(ctx,
		`SELECT`+playerColumns+` FROM players WHERE id = $1 AND active = true`,
		id,
	)
	p, err := scanPlayer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPlayerByID: %w", err)
	}
	return p, nil
}

// CreatePlayerArgs holds parameters for CreatePlayer.
type CreatePlayerArgs struct {
	Name         string
	Nickname     *string
	WhatsApp     string
	PasswordHash string
}

// CreatePlayer inserts a new player and returns the created record.
func CreatePlayer(ctx context.Context, pool *pgxpool.Pool, args CreatePlayerArgs) (*Player, error) {
	row := pool.QueryRow(ctx,
		`INSERT INTO players (name, nickname, whatsapp, password_hash, role)
		 VALUES ($1, $2, $3, $4, 'player')
		 RETURNING`+playerColumns,
		args.Name, args.Nickname, args.WhatsApp, args.PasswordHash,
	)
	p, err := scanPlayer(row)
	if err != nil {
		return nil, fmt.Errorf("CreatePlayer: %w", err)
	}
	return p, nil
}

// UpdatePlayerPassword updates the password hash and clears must_change_password.
func UpdatePlayerPassword(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, hash string) error {
	_, err := pool.Exec(ctx,
		`UPDATE players SET password_hash = $1, must_change_password = false WHERE id = $2`,
		hash, id,
	)
	return err
}

// UpdatePlayerMustChangePassword sets the must_change_password flag.
func UpdatePlayerMustChangePassword(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, val bool) error {
	_, err := pool.Exec(ctx,
		`UPDATE players SET must_change_password = $1 WHERE id = $2`,
		val, id,
	)
	return err
}

// GetPlayerByMCPToken validates an MCP token hash and returns the associated player.
// Updates last_used_at as a side-effect.
func GetPlayerByMCPToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (*Player, error) {
	row := pool.QueryRow(ctx,
		`SELECT p.`+playerColumns+`
		 FROM players p
		 JOIN mcp_tokens m ON m.player_id = p.id
		 WHERE m.token_hash = $1
		   AND m.revoked_at IS NULL
		   AND (m.expires_at IS NULL OR m.expires_at > now())
		   AND p.active = true`,
		tokenHash,
	)
	p, err := scanPlayer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPlayerByMCPToken: %w", err)
	}
	// Update last_used_at asynchronously (best-effort)
	go func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE mcp_tokens SET last_used_at = now() WHERE token_hash = $1`, tokenHash)
	}()
	return p, nil
}

// HashToken returns the SHA-256 hex digest of a token string.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateRefreshToken inserts a new refresh token (stored as hash).
func CreateRefreshToken(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, tokenHash string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO refresh_tokens (player_id, token_hash, expires_at)
		 VALUES ($1, $2, now() + interval '30 days')`,
		playerID, tokenHash,
	)
	return err
}

// GetValidRefreshToken fetches a non-revoked, non-expired refresh token by hash.
func GetValidRefreshToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (*RefreshToken, error) {
	var rt RefreshToken
	err := pool.QueryRow(ctx,
		`SELECT id, player_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens
		 WHERE token_hash = $1
		   AND revoked_at IS NULL
		   AND expires_at > now()`,
		tokenHash,
	).Scan(&rt.ID, &rt.PlayerID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetValidRefreshToken: %w", err)
	}
	return &rt, nil
}

// RevokeRefreshToken marks a single refresh token as revoked.
func RevokeRefreshToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) error {
	_, err := pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}

// RevokeAllRefreshTokensForPlayer revokes every refresh token for a player.
func RevokeAllRefreshTokensForPlayer(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now()
		 WHERE player_id = $1 AND revoked_at IS NULL`,
		playerID,
	)
	return err
}

// UpdatePlayerApiV2Enabled toggles the api_v2_enabled flag.
func UpdatePlayerApiV2Enabled(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, enabled bool) error {
	_, err := pool.Exec(ctx,
		`UPDATE players SET api_v2_enabled = $1 WHERE id = $2`,
		enabled, id,
	)
	return err
}

// ListPlayersForApiV2 returns all non-admin players with their api_v2_enabled status.
type PlayerApiV2Row struct {
	ID           uuid.UUID
	Name         string
	WhatsApp     string
	ApiV2Enabled bool
}

func ListPlayersForApiV2(ctx context.Context, pool *pgxpool.Pool, limit, offset int) ([]PlayerApiV2Row, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, name, whatsapp, api_v2_enabled
		 FROM players
		 WHERE role != 'admin' AND active = true
		 ORDER BY name ASC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PlayerApiV2Row
	for rows.Next() {
		var r PlayerApiV2Row
		if err := rows.Scan(&r.ID, &r.Name, &r.WhatsApp, &r.ApiV2Enabled); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
