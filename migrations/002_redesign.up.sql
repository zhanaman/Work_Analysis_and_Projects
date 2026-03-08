-- Migration 002: Redesign schema for per-center tier architecture
-- Drops old flat schema and creates normalized tables

-- Drop old tables
DROP TABLE IF EXISTS partners CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Core partner identity (shared across centers)
CREATE TABLE partners (
    id                    SERIAL PRIMARY KEY,
    party_id              TEXT UNIQUE NOT NULL,      -- Signing Entity Party ID
    name                  TEXT NOT NULL,              -- Party Name
    theater               TEXT,
    sub_region            TEXT,
    hpe_org               TEXT,                       -- HPE Organization
    country               TEXT,
    country_code          TEXT,
    hq_partner_id_t1      TEXT,
    hq_partner_id_t2      TEXT,
    hq_partner_id_sp      TEXT,
    country_entity_id     TEXT,
    country_entity_name   TEXT,
    global_entity_id      TEXT,
    global_entity_name    TEXT,
    business_rel_sp       TEXT,                       -- Current Business Relationship (Solution Provider)
    business_rel_svc      TEXT,                       -- Service Provider
    business_rel_si       TEXT,                       -- System Integrator
    active_agreement      BOOLEAN DEFAULT false,
    lr_status             TEXT,
    lr_status_start       DATE,
    lr_status_end         DATE,
    sp_agreement          BOOLEAN DEFAULT false,
    membership_compute    TEXT,                       -- "Silver Partner" etc.
    membership_hc         TEXT,
    membership_networking TEXT,
    triple_platinum_plus  BOOLEAN DEFAULT false,
    intl_partner          BOOLEAN DEFAULT false,
    imported_at           TIMESTAMPTZ DEFAULT NOW()
);

-- Per-center, per-tier data
CREATE TABLE partner_tiers (
    id                  SERIAL PRIMARY KEY,
    partner_id          INT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    center              TEXT NOT NULL,                -- 'compute', 'hybrid_cloud', 'networking'
    tier                TEXT NOT NULL,                -- 'business', 'silver', 'gold', 'platinum'

    -- Status flags
    criteria_met        BOOLEAN DEFAULT false,
    volume_status       BOOLEAN DEFAULT false,
    growth_plan_status  BOOLEAN DEFAULT false,
    growth_plan_end     DATE,
    cert_status         BOOLEAN DEFAULT false,

    -- Volume (actuals from Excel, per-partner thresholds)
    volume_actuals      NUMERIC,
    volume_actuals_total NUMERIC,                    -- includes SP direct orders
    threshold           NUMERIC,
    volume_pct          NUMERIC,                     -- actuals / threshold %

    -- SRI (Gold+ only)
    sri                 NUMERIC,
    sri_required        NUMERIC,
    sri_pct             NUMERIC,
    sri_status          BOOLEAN,

    -- Certifications (stored as "x/y" text)
    sales_certified     TEXT,                        -- e.g. "2/3"
    atp_current         TEXT,                        -- e.g. "1/1"
    atp_active          INT,
    atp_at_risk         INT,
    atp_not_current     INT,
    ase_current         TEXT,                        -- e.g. "0/2"
    ase_active          INT,
    ase_at_risk         INT,
    ase_not_current     INT,
    mase_current        TEXT,                        -- e.g. "0/1" (Platinum only)
    mase_active         INT,
    mase_at_risk        INT,
    mase_not_current    INT,

    -- Dates
    tier_start_date     DATE,
    tier_end_date       DATE,

    UNIQUE(partner_id, center, tier)
);

-- Competencies (14 possible per partner)
CREATE TABLE partner_competencies (
    id              SERIAL PRIMARY KEY,
    partner_id      INT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    competency      TEXT NOT NULL,
    criteria_met    BOOLEAN DEFAULT false,
    UNIQUE(partner_id, competency)
);

-- Quarterly compensation levels
CREATE TABLE partner_comp_levels (
    id              SERIAL PRIMARY KEY,
    partner_id      INT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    center          TEXT NOT NULL,
    quarter         TEXT NOT NULL,                    -- 'Q126', 'Q226', etc.
    comp_level      TEXT,                             -- 'Silver', 'Gold', etc.
    at_risk         TEXT,                             -- comp@risk value
    UNIQUE(partner_id, center, quarter)
);

-- Revenue breakdown per center
CREATE TABLE partner_revenue (
    id                  SERIAL PRIMARY KEY,
    partner_id          INT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    center              TEXT NOT NULL,
    total_business      NUMERIC,
    total_products      NUMERIC,
    total_services      NUMERIC,
    ops_lob             NUMERIC,
    attach_sm           NUMERIC,
    ib_sm               NUMERIC,
    refresh_date        DATE,
    UNIQUE(partner_id, center)
);

-- SCS (Service Contract Specialist)
CREATE TABLE partner_scs (
    id                  SERIAL PRIMARY KEY,
    partner_id          INT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    membership          TEXT,
    criteria_met        BOOLEAN DEFAULT false,
    prv_membership_met  BOOLEAN DEFAULT false,
    threshold_met       BOOLEAN DEFAULT false,
    volume_actuals      NUMERIC,
    threshold           NUMERIC,
    volume_pct          NUMERIC,
    UNIQUE(partner_id)
);

-- Users (with region filter for RBAC)
CREATE TABLE users (
    id              SERIAL PRIMARY KEY,
    telegram_id     BIGINT UNIQUE NOT NULL,
    username        TEXT,
    full_name       TEXT,
    role            TEXT DEFAULT 'pending',           -- 'pending', 'user', 'pbm', 'admin'
    region_filter   TEXT,                             -- NULL=all CCA, 'RMC'=RMC only, partner_id=own data
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Data import log
CREATE TABLE import_log (
    id              SERIAL PRIMARY KEY,
    filename        TEXT NOT NULL,
    data_date       DATE,                             -- Date extracted from filename
    sheets_parsed   TEXT[],
    partners_total  INT,
    partners_cca    INT,
    duration_ms     INT,
    imported_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_partners_name_trgm ON partners USING gin (name gin_trgm_ops);
CREATE INDEX idx_partners_party_id ON partners(party_id);
CREATE INDEX idx_partners_region ON partners(country, hpe_org, sub_region);
CREATE INDEX idx_partners_country_code ON partners(country_code);
CREATE INDEX idx_partner_tiers_lookup ON partner_tiers(partner_id, center, tier);
CREATE INDEX idx_partner_tiers_criteria ON partner_tiers(center, tier, criteria_met);
CREATE INDEX idx_partner_competencies_lookup ON partner_competencies(partner_id);
CREATE INDEX idx_users_telegram ON users(telegram_id);
