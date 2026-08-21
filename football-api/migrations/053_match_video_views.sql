-- Migration 053: contador de visualizações no feed do rachão (PRD 052)

ALTER TABLE match_videos
ADD COLUMN IF NOT EXISTS view_count INT NOT NULL DEFAULT 0;
