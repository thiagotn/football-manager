CREATE TABLE IF NOT EXISTS android_beta_signups (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_email TEXT NOT NULL CHECK (char_length(google_email) <= 254),
    player_id    UUID REFERENCES players(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_android_beta_signups_created
    ON android_beta_signups (created_at DESC);
