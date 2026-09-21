-- Allow administrators to temporarily revoke access without deleting an account.
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;
ALTER TABLE users ADD CONSTRAINT chk_users_status
  CHECK (status IN ('ACTIVE', 'IN_REVIEW', 'REJECTED', 'SUSPENDED'));