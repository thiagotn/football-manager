-- Tabela para idempotência de webhooks (RNF-02).
-- O event_id único garante que cada evento Stripe seja processado exatamente uma vez.

CREATE TABLE IF NOT EXISTS webhook_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    VARCHAR(255) NOT NULL UNIQUE,  -- Stripe event ID (evt_xxx)
    event_type  VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
