-- name: GetPlayerByWhatsApp :one
SELECT id, name, whatsapp, password_hash, role, active, api_v2_enabled,
       must_change_password, avatar_url, plan, plan_expires_at,
       stripe_customer_id, stripe_subscription_id, created_at, updated_at
FROM players
WHERE whatsapp = $1 AND active = true;

-- name: GetPlayerByID :one
SELECT id, name, whatsapp, password_hash, role, active, api_v2_enabled,
       must_change_password, avatar_url, plan, plan_expires_at,
       stripe_customer_id, stripe_subscription_id, created_at, updated_at
FROM players
WHERE id = $1 AND active = true;

-- name: CreatePlayer :one
INSERT INTO players (name, whatsapp, password_hash, role, active, api_v2_enabled)
VALUES ($1, $2, $3, 'player', true, false)
RETURNING id, name, whatsapp, password_hash, role, active, api_v2_enabled,
          must_change_password, avatar_url, plan, plan_expires_at,
          stripe_customer_id, stripe_subscription_id, created_at, updated_at;

-- name: UpdatePlayerPassword :exec
UPDATE players
SET password_hash = $2, must_change_password = false, updated_at = NOW()
WHERE id = $1;

-- name: UpdatePlayerMustChangePassword :exec
UPDATE players
SET must_change_password = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdatePlayerApiV2Enabled :one
UPDATE players
SET api_v2_enabled = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, name, whatsapp, password_hash, role, active, api_v2_enabled,
          must_change_password, avatar_url, plan, plan_expires_at,
          stripe_customer_id, stripe_subscription_id, created_at, updated_at;

-- name: ListPlayersForApiV2 :many
SELECT id, name, whatsapp, password_hash, role, active, api_v2_enabled,
       must_change_password, avatar_url, plan, plan_expires_at,
       stripe_customer_id, stripe_subscription_id, created_at, updated_at
FROM players
WHERE role != 'admin'
ORDER BY name
LIMIT $1 OFFSET $2;
