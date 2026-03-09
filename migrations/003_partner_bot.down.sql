-- Migration 003 rollback: Remove partner bot additions

DROP TABLE IF EXISTS email_verifications;

ALTER TABLE users DROP COLUMN IF EXISTS partner_id;
ALTER TABLE users DROP COLUMN IF EXISTS email;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS lang;
ALTER TABLE users DROP COLUMN IF EXISTS bot_type;
