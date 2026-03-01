-- ============================================================
-- 005_match_venue_fields.sql
-- Tipo de quadra e quantidade de jogadores por time
-- ============================================================

DO $$ BEGIN
  CREATE TYPE court_type AS ENUM ('campo', 'sintetico', 'terrao', 'quadra');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE matches
  ADD COLUMN IF NOT EXISTS court_type      court_type NULL,
  ADD COLUMN IF NOT EXISTS players_per_team SMALLINT  NULL
    CONSTRAINT chk_players_per_team CHECK (players_per_team BETWEEN 2 AND 15);
