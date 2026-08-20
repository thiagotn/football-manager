-- Migration 050: vídeos curtos verticais por rachão (PRD 052)
-- Feature flag experimental por conta (players.videos_enabled): habilita vídeos
-- nos grupos onde o player é admin. Gerenciada pelo super admin.

ALTER TABLE players
ADD COLUMN IF NOT EXISTS videos_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN players.videos_enabled IS
  'Feature flag experimental de vídeos: habilita vídeos nos grupos onde este player é admin. Gerenciado pelo super admin. Padrão: FALSE.';

CREATE TABLE IF NOT EXISTS match_videos (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id         UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    uploaded_by      UUID        NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    status           TEXT        NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','uploaded','processing','ready','failed')),
    original_key     TEXT        NOT NULL,           -- videos/original/{match_id}/{video_id}.mp4
    video_url        TEXT,                           -- https://cdn.rachao.app/videos/{match_id}/{video_id}.mp4
    poster_url       TEXT,                           -- https://cdn.rachao.app/videos/{match_id}/{video_id}.webp
    duration_seconds NUMERIC(6,2),
    size_bytes       BIGINT,
    error            TEXT,
    attempts         INT         NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_videos_match_id ON match_videos (match_id);
CREATE INDEX IF NOT EXISTS idx_match_videos_uploaded_by ON match_videos (uploaded_by);
-- Índice parcial para o poll do worker (a fila é sempre pequena)
CREATE INDEX IF NOT EXISTS idx_match_videos_queue ON match_videos (created_at)
  WHERE status IN ('uploaded','processing');
