-- Seller account status and identity document URLs
-- Status: ACTIVE (default), IN_REVIEW (new seller signup), REJECTED
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ACTIVE';
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;
ALTER TABLE users ADD CONSTRAINT chk_users_status CHECK (status IN ('ACTIVE', 'IN_REVIEW', 'REJECTED'));

-- Document URLs (filled at signup for sellers; used for admin review)
ALTER TABLE users ADD COLUMN IF NOT EXISTS business_registration_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS id_card_front_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS id_card_back_url TEXT;

-- Backfill existing users: sellers could be set to ACTIVE (already approved) or leave as-is
-- Default ACTIVE is already set for new column, existing rows get ACTIVE via DEFAULT.
