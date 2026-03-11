-- Migration: 021_match_teams.sql
-- Cria as tabelas de times gerados por sorteio para cada partida.

CREATE TABLE match_teams (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    color      VARCHAR(7),              -- hex opcional, ex: '#e63946'
    position   SMALLINT NOT NULL,       -- ordem do time (1, 2, ...)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE match_team_players (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID NOT NULL REFERENCES match_teams(id) ON DELETE CASCADE,
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    is_reserve  BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (team_id, player_id)
);

CREATE INDEX idx_match_teams_match_id ON match_teams(match_id);
