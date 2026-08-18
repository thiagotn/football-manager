-- Cadastros temporários criados por admin de grupo: flag pending_registration
-- + convites de claim endereçados a um jogador (invite_tokens.purpose/target_player_id).
ALTER TABLE players
  ADD COLUMN IF NOT EXISTS pending_registration BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE invite_tokens
  ADD COLUMN IF NOT EXISTS purpose VARCHAR(20) NOT NULL DEFAULT 'group_join';

ALTER TABLE invite_tokens
  ADD COLUMN IF NOT EXISTS target_player_id UUID REFERENCES players(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_invite_tokens_target_player_id
  ON invite_tokens(target_player_id);

-- Backfill: whatsapp fora do E.164 OU (must_change_password e nunca logou).
UPDATE players p
SET pending_registration = TRUE
WHERE p.pending_registration = FALSE
  AND (
    p.whatsapp !~ '^\+[1-9][0-9]{6,14}$'
    OR (
      p.must_change_password = TRUE
      AND NOT EXISTS (SELECT 1 FROM refresh_tokens rt WHERE rt.player_id = p.id)
    )
  );
