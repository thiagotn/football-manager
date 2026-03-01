-- ============================================================
-- 003_match_number.sql
-- Adiciona número sequencial global às partidas
-- ============================================================

-- Cria a sequence global (idempotente)
CREATE SEQUENCE IF NOT EXISTS matches_number_seq START 1;

-- Adiciona a coluna (nullable temporariamente para popular dados existentes)
ALTER TABLE matches ADD COLUMN IF NOT EXISTS number INTEGER;

-- Popula partidas existentes com números sequenciais ordenados por created_at
UPDATE matches SET number = sub.rn
FROM (
  SELECT id, ROW_NUMBER() OVER (ORDER BY created_at) AS rn
  FROM matches
  WHERE number IS NULL
) sub
WHERE matches.id = sub.id;

-- Avança a sequence para continuar após os valores já atribuídos
SELECT setval('matches_number_seq', COALESCE((SELECT MAX(number) FROM matches), 0) + 1, false);

-- Define NOT NULL e default da sequence
ALTER TABLE matches
  ALTER COLUMN number SET NOT NULL,
  ALTER COLUMN number SET DEFAULT nextval('matches_number_seq');

ALTER SEQUENCE matches_number_seq OWNED BY matches.number;

CREATE UNIQUE INDEX IF NOT EXISTS idx_matches_number ON matches(number);
