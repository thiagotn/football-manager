-- Migration 038: match_player_stats
-- Registros de gols e assistências por jogador por partida.
-- Somente admin do grupo pode inserir/editar.

CREATE TABLE IF NOT EXISTS match_player_stats (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    player_id   UUID        NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    goals       INTEGER     NOT NULL DEFAULT 0 CHECK (goals >= 0 AND goals <= 20),
    assists     INTEGER     NOT NULL DEFAULT 0 CHECK (assists >= 0 AND assists <= 20),
    recorded_by UUID        NOT NULL REFERENCES players(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_mps_match_id  ON match_player_stats (match_id);
CREATE INDEX IF NOT EXISTS idx_mps_player_id ON match_player_stats (player_id);
