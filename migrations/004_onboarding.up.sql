-- Migration 004: Onboarding flow
-- Adds stateful onboarding fields for 3-step partner registration

ALTER TABLE users ADD COLUMN IF NOT EXISTS onboard_step TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS company_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS onboard_msg_id INT;
