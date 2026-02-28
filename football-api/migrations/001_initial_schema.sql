-- ============================================================
-- 001_initial_schema.sql
-- Esquema inicial da API de Futebol
-- ============================================================

-- ── Extensões ─────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── ENUMs ─────────────────────────────────────────────────────
DO $$ BEGIN CREATE TYPE player_role AS ENUM ('admin', 'player'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE group_member_role AS ENUM ('admin', 'member'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE match_status AS ENUM ('open', 'closed'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE attendance_status AS ENUM ('pending', 'confirmed', 'declined'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ── Função: updated_at automático ─────────────────────────────
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$;

-- ── players ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS players (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          VARCHAR(100) NOT NULL,
  nickname      VARCHAR(50),
  whatsapp      VARCHAR(20) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role          player_role NOT NULL DEFAULT 'player',
  active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_players_whatsapp ON players(whatsapp);
DROP TRIGGER IF EXISTS trg_players_updated_at ON players;
CREATE TRIGGER trg_players_updated_at BEFORE UPDATE ON players
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── groups ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS groups (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  slug        VARCHAR(60) NOT NULL UNIQUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_groups_slug ON groups(slug);
DROP TRIGGER IF EXISTS trg_groups_updated_at ON groups;
CREATE TRIGGER trg_groups_updated_at BEFORE UPDATE ON groups
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── group_members ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS group_members (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  player_id  UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  role       group_member_role NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_group_player UNIQUE (group_id, player_id)
);
DROP TRIGGER IF EXISTS trg_group_members_updated_at ON group_members;
CREATE TRIGGER trg_group_members_updated_at BEFORE UPDATE ON group_members
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── matches ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS matches (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  group_id       UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  match_date     DATE NOT NULL,
  start_time     TIME NOT NULL,
  location       VARCHAR(200) NOT NULL,
  notes          TEXT,
  hash           VARCHAR(12) NOT NULL UNIQUE,
  status         match_status NOT NULL DEFAULT 'open',
  created_by_id  UUID REFERENCES players(id) ON DELETE SET NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_matches_hash ON matches(hash);
CREATE INDEX IF NOT EXISTS idx_matches_group_id ON matches(group_id);
DROP TRIGGER IF EXISTS trg_matches_updated_at ON matches;
CREATE TRIGGER trg_matches_updated_at BEFORE UPDATE ON matches
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── attendances ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS attendances (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  player_id  UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  status     attendance_status NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_match_player UNIQUE (match_id, player_id)
);
DROP TRIGGER IF EXISTS trg_attendances_updated_at ON attendances;
CREATE TRIGGER trg_attendances_updated_at BEFORE UPDATE ON attendances
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── invite_tokens ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS invite_tokens (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  group_id       UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  token          VARCHAR(64) NOT NULL UNIQUE,
  expires_at     TIMESTAMPTZ NOT NULL,
  used           BOOLEAN NOT NULL DEFAULT FALSE,
  created_by_id  UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  used_by_id     UUID REFERENCES players(id) ON DELETE SET NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invite_tokens_token ON invite_tokens(token);
DROP TRIGGER IF EXISTS trg_invite_tokens_updated_at ON invite_tokens;
CREATE TRIGGER trg_invite_tokens_updated_at BEFORE UPDATE ON invite_tokens
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Seed: admin inicial ────────────────────────────────────────
-- Senha: admin123 (trocar em produção!)
INSERT INTO players (name, nickname, whatsapp, password_hash, role) VALUES
  (
    'Administrador',
    'Admin',
    '11999990000',
    '$2b$12$dZQ.U8Uh1.OdmsbPQSL55efhbdS3QeAntsqOEie3GpzMMSSQLuXPO',
    'admin'
  )
ON CONFLICT (whatsapp) DO NOTHING;
