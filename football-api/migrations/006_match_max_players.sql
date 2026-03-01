-- ============================================================
-- 006_match_max_players.sql
-- Quantidade máxima de jogadores por partida
-- ============================================================

ALTER TABLE matches
  ADD COLUMN IF NOT EXISTS max_players SMALLINT NULL
    CONSTRAINT chk_max_players CHECK (max_players >= 2);
