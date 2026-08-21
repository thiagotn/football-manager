-- Migration 052: fotos no feed do rachão (PRD 052)
-- match_videos passa a aceitar imagens: media_type distingue o pipeline
-- ('video' = transcode ffmpeg; 'image' = resize+JPEG no worker).

ALTER TABLE match_videos
ADD COLUMN IF NOT EXISTS media_type TEXT NOT NULL DEFAULT 'video';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'match_videos_media_type_check'
    ) THEN
        ALTER TABLE match_videos
        ADD CONSTRAINT match_videos_media_type_check
        CHECK (media_type IN ('video', 'image'));
    END IF;
END $$;
