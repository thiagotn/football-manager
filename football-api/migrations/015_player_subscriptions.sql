-- 015_player_subscriptions.sql
-- Plano gratuito: 1 grupo como organizador, 30 membros por grupo
-- Fase 1: apenas plano 'free'. Planos pagos serão adicionados em fases futuras.

CREATE TABLE player_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE,
    plan        VARCHAR(20) NOT NULL DEFAULT 'free',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Back-fill: todos os players existentes recebem o plano gratuito
INSERT INTO player_subscriptions (player_id, plan)
SELECT id, 'free' FROM players
ON CONFLICT (player_id) DO NOTHING;
