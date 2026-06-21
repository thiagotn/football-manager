-- Issue #6: lembrete push 30 min antes de fechar a votação para confirmados
-- que ainda não votaram. A coluna garante idempotência do cron (a mesma
-- partida não recebe mais de um lembrete).
ALTER TABLE matches
  ADD COLUMN IF NOT EXISTS vote_reminder_sent_at TIMESTAMPTZ;
