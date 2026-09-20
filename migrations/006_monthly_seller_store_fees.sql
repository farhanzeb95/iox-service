-- Convert seller store fees from one lifetime record to one record per seller/month.
ALTER TABLE seller_store_fees
  ADD COLUMN IF NOT EXISTS billing_period_start DATE,
  ADD COLUMN IF NOT EXISTS billing_period_end DATE;

UPDATE seller_store_fees
SET billing_period_start = date_trunc('month', submitted_at)::date,
    billing_period_end = (date_trunc('month', submitted_at) + INTERVAL '1 month')::date
WHERE billing_period_start IS NULL OR billing_period_end IS NULL;

ALTER TABLE seller_store_fees
  ALTER COLUMN billing_period_start SET NOT NULL,
  ALTER COLUMN billing_period_end SET NOT NULL;

ALTER TABLE seller_store_fees
  DROP CONSTRAINT IF EXISTS seller_store_fees_seller_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_seller_store_fees_seller_period
  ON seller_store_fees(seller_id, billing_period_start);