-- Migration 051: curtidas em vídeos de rachão (PRD 052)

CREATE TABLE IF NOT EXISTS match_video_likes (
    video_id   UUID        NOT NULL REFERENCES match_videos(id) ON DELETE CASCADE,
    player_id  UUID        NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (video_id, player_id)
);
