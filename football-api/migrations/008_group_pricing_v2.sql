-- Substitui o modelo de cobrança único (pricing_model enum + pricing_amount)
-- por dois campos independentes e opcionais: per_match_amount e monthly_amount.
-- Um grupo é gratuito quando ambos são NULL.

ALTER TABLE groups
  DROP COLUMN IF EXISTS pricing_model,
  DROP COLUMN IF EXISTS pricing_amount;

DROP TYPE IF EXISTS pricing_model;

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS per_match_amount NUMERIC(10,2) NULL
    CONSTRAINT chk_per_match_amount CHECK (per_match_amount IS NULL OR per_match_amount >= 0),
  ADD COLUMN IF NOT EXISTS monthly_amount NUMERIC(10,2) NULL
    CONSTRAINT chk_monthly_amount CHECK (monthly_amount IS NULL OR monthly_amount >= 0);
