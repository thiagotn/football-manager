-- Migration: 026_group_finance.sql
-- Tabelas de controle financeiro por grupo

CREATE TABLE IF NOT EXISTS finance_periods (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    year        SMALLINT NOT NULL,
    month       SMALLINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, year, month)
);

CREATE INDEX IF NOT EXISTS idx_finance_periods_group
    ON finance_periods (group_id, year DESC, month DESC);

CREATE TABLE IF NOT EXISTS finance_payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id       UUID NOT NULL REFERENCES finance_periods(id) ON DELETE CASCADE,
    player_id       UUID NOT NULL REFERENCES players(id),
    player_name     VARCHAR(100) NOT NULL,
    payment_type    VARCHAR(20),           -- 'monthly' | 'per_match' | null enquanto pendente
    amount_due      INT,                   -- centavos; null enquanto pendente
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (period_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_finance_payments_period
    ON finance_payments (period_id);
