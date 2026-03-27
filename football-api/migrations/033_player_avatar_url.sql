-- Migration 033: Avatar de jogador
-- Adiciona avatar_url em players e cria tabela de logs de upload

ALTER TABLE players
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

CREATE TABLE IF NOT EXISTS avatar_upload_logs (
    id          BIGSERIAL PRIMARY KEY,
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    ip_address  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_avatar_upload_logs_player_id
    ON avatar_upload_logs (player_id);
