-- Adiciona billing_cycle à tabela player_subscriptions
-- Preenchido a partir do metadata do checkout Stripe (monthly | yearly)

ALTER TABLE player_subscriptions
    ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly';
