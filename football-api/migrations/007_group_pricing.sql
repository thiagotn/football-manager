DO $$ BEGIN
  CREATE TYPE pricing_model AS ENUM ('free', 'per_match', 'monthly');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS pricing_model pricing_model NOT NULL DEFAULT 'free',
  ADD COLUMN IF NOT EXISTS pricing_amount NUMERIC(10,2) NULL
    CONSTRAINT chk_pricing_amount CHECK (pricing_amount IS NULL OR pricing_amount >= 0);
