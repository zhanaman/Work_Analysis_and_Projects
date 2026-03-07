-- Enable pg_trgm extension for fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Partners table
CREATE TABLE IF NOT EXISTS partners (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    partner_id_hpe   TEXT UNIQUE,
    tier             TEXT,
    country          TEXT,
    city             TEXT,

    -- Certifications
    compute_cert     TEXT,
    networking_cert  TEXT,
    hybrid_cloud_cert TEXT,
    storage_cert     TEXT,

    -- Volumes
    revenue_ytd      NUMERIC DEFAULT 0,
    target           NUMERIC DEFAULT 0,

    -- Contact
    contact_name     TEXT,
    contact_email    TEXT,
    contact_phone    TEXT,

    -- Raw backup
    raw_data         JSONB,

    -- Timestamps
    imported_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

-- Trigram index for fuzzy search on partner name
CREATE INDEX IF NOT EXISTS idx_partners_name_trgm
    ON partners USING gin (name gin_trgm_ops);

-- Index for HPE partner ID lookups
CREATE INDEX IF NOT EXISTS idx_partners_hpe_id
    ON partners (partner_id_hpe);

-- Index for tier-based filtering
CREATE INDEX IF NOT EXISTS idx_partners_tier
    ON partners (tier);

-- Index for country filtering
CREATE INDEX IF NOT EXISTS idx_partners_country
    ON partners (country);


-- Users table
CREATE TABLE IF NOT EXISTS users (
    id              SERIAL PRIMARY KEY,
    telegram_id     BIGINT UNIQUE NOT NULL,
    username        TEXT,
    full_name       TEXT,
    role            TEXT DEFAULT 'pending',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast Telegram ID lookups
CREATE INDEX IF NOT EXISTS idx_users_telegram_id
    ON users (telegram_id);
