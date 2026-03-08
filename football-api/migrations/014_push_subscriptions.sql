CREATE TABLE push_subscriptions (
    id          SERIAL PRIMARY KEY,
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    endpoint    TEXT NOT NULL,
    p256dh      TEXT NOT NULL,
    auth        TEXT NOT NULL,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (player_id, endpoint)
);
