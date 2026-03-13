-- Adiciona colunas de integração Stripe na tabela player_subscriptions.
-- Necessário para: POST /api/v1/subscriptions (checkout) e
--                  POST /api/v1/webhooks/payment (ativação de plano).

ALTER TABLE player_subscriptions
    ADD COLUMN IF NOT EXISTS gateway_customer_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS gateway_sub_id       VARCHAR(255),
    ADD COLUMN IF NOT EXISTS status               VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS grace_period_end     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS current_period_end   TIMESTAMPTZ;
