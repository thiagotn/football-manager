-- Add is_public flag to groups
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT TRUE;

-- Create waitlist_status enum
DO $$ BEGIN
  CREATE TYPE waitlist_status AS ENUM ('pending', 'accepted', 'rejected');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Create match_waitlist table
CREATE TABLE IF NOT EXISTS match_waitlist (
  id           UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id     UUID            NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  player_id    UUID            NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  intro        TEXT,
  agreed_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),
  status       waitlist_status NOT NULL DEFAULT 'pending',
  reviewed_by  UUID            REFERENCES players(id),
  reviewed_at  TIMESTAMPTZ,
  created_at   TIMESTAMPTZ     NOT NULL DEFAULT now(),
  UNIQUE (match_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_match_waitlist_match   ON match_waitlist (match_id);
CREATE INDEX IF NOT EXISTS idx_match_waitlist_player  ON match_waitlist (player_id);
CREATE INDEX IF NOT EXISTS idx_match_waitlist_status  ON match_waitlist (status);
