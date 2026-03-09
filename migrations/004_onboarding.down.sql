-- Migration 004 down: Remove onboarding fields

ALTER TABLE users DROP COLUMN IF EXISTS onboard_step;
ALTER TABLE users DROP COLUMN IF EXISTS company_name;
ALTER TABLE users DROP COLUMN IF EXISTS onboard_msg_id;
