-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (token_hash, player_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetValidRefreshToken :one
SELECT token_hash, player_id, expires_at, revoked, created_at
FROM refresh_tokens
WHERE token_hash = $1
  AND revoked = false
  AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = true
WHERE token_hash = $1;

-- name: RevokeAllRefreshTokensForPlayer :exec
UPDATE refresh_tokens
SET revoked = true
WHERE player_id = $1 AND revoked = false;

-- name: GetPlayerByMCPToken :one
SELECT p.id, p.name, p.whatsapp, p.password_hash, p.role, p.active,
       p.must_change_password, p.avatar_url, p.plan, p.plan_expires_at,
       p.stripe_customer_id, p.stripe_subscription_id, p.created_at, p.updated_at
FROM players p
JOIN mcp_tokens m ON m.player_id = p.id
WHERE m.token_hash = $1
  AND m.revoked = false
  AND p.active = true;

-- name: UpdateMCPTokenLastUsed :exec
UPDATE mcp_tokens
SET last_used_at = NOW()
WHERE token_hash = $1;
