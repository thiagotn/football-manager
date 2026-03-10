-- Registro de cada voto submetido (um por jogador por partida)
CREATE TABLE match_votes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id     UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    voter_id     UUID NOT NULL REFERENCES players(id),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, voter_id)
);

-- Escolhas do top 5 (1 a 5 registros por voto)
CREATE TABLE match_vote_top5 (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id   UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    position  SMALLINT NOT NULL CHECK (position IN (1, 2, 3, 4, 5)),
    points    SMALLINT NOT NULL CHECK (points IN (2, 4, 6, 8, 10)),
    UNIQUE (vote_id, position),
    UNIQUE (vote_id, player_id)
);

-- Escolha da decepção (0 ou 1 registro por voto)
CREATE TABLE match_vote_flop (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id   UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    UNIQUE (vote_id)
);

-- Flag para controlar envio único da push notification de abertura
ALTER TABLE matches ADD COLUMN vote_notified BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_match_votes_match    ON match_votes (match_id);
CREATE INDEX idx_match_vote_top5_vote ON match_vote_top5 (vote_id);
CREATE INDEX idx_match_vote_flop_vote ON match_vote_flop (vote_id);
