-- Seller store fee submissions
-- Private sellers: PKR 500; business sellers: PKR 1,000.
CREATE TABLE IF NOT EXISTS seller_store_fees (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  seller_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  amount DECIMAL(12,2) NOT NULL CHECK (amount > 0),
  payment_method TEXT NOT NULL CHECK (payment_method IN ('JAZZCASH', 'EASYPAISA', 'RAAST', 'BANK_TRANSFER')),
  payment_reference TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PAID', 'REJECTED')),
  review_note TEXT,
  reviewed_by UUID REFERENCES users(id),
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_seller_store_fees_status ON seller_store_fees(status);
CREATE INDEX IF NOT EXISTS idx_seller_store_fees_seller ON seller_store_fees(seller_id);
