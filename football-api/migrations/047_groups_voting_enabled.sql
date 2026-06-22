-- Issue #10: toggle por grupo para ligar/desligar a votação pós-rachão.
-- Default true preserva o comportamento atual de todos os grupos existentes.
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS voting_enabled BOOLEAN NOT NULL DEFAULT TRUE;
