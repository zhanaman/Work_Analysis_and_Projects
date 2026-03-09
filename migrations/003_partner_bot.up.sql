-- Migration 003: Partner Self-Service Bot
-- Adds partner_id, email, language, and email verification support

-- Extend users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS lang TEXT DEFAULT 'ru';
ALTER TABLE users ADD COLUMN IF NOT EXISTS bot_type TEXT DEFAULT 'pbm';

-- Email verification codes
CREATE TABLE IF NOT EXISTS email_verifications (
    id          SERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    code        TEXT NOT NULL,
    partner_id  INT REFERENCES partners(id),
    expires_at  TIMESTAMPTZ NOT NULL,
    verified    BOOLEAN DEFAULT false,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_code ON email_verifications(code, verified);
CREATE INDEX IF NOT EXISTS idx_users_partner ON users(partner_id);
CREATE INDEX IF NOT EXISTS idx_users_bot_type ON users(bot_type);
