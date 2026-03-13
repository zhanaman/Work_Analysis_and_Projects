-- Activity log for admin reporting
-- Tracks search queries and partner card views per user

CREATE TABLE IF NOT EXISTS activity_log (
    id           SERIAL PRIMARY KEY,
    user_id      INT REFERENCES users(id) ON DELETE SET NULL,
    telegram_id  BIGINT NOT NULL,
    event_type   TEXT NOT NULL,   -- 'search' | 'partner_view'
    query        TEXT,            -- search string (for 'search' events)
    partner_id   INT,             -- partner DB ID (for 'partner_view' events)
    partner_name TEXT,            -- partner name snapshot at time of view
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_log_telegram_id ON activity_log(telegram_id);
CREATE INDEX IF NOT EXISTS idx_activity_log_created_at  ON activity_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_log_user_id     ON activity_log(user_id);
