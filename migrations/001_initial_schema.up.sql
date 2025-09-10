-- Users & auth
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Currency enum
CREATE TYPE currency AS ENUM ('USD', 'BTC', 'ETH');

-- Accounts & ledger (double-entry)
CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  currency currency NOT NULL,
  balance_available NUMERIC(30,10) NOT NULL DEFAULT 0,
  balance_hold NUMERIC(30,10) NOT NULL DEFAULT 0,
  UNIQUE (user_id, currency)
);

-- Each business operation creates a balanced journal (sum=0)
CREATE TABLE ledger_entries (
  id BIGSERIAL PRIMARY KEY,
  journal_id UUID NOT NULL,
  account_id UUID NOT NULL REFERENCES accounts(id),
  amount NUMERIC(30,10) NOT NULL, -- positive=credit, negative=debit
  currency currency NOT NULL,
  ref_type TEXT NOT NULL,          -- 'TOPUP'|'ORDER_FILL'|'FEE'...
  ref_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON ledger_entries (journal_id);

-- Orders
CREATE TYPE order_side AS ENUM ('BUY','SELL');
CREATE TYPE order_type AS ENUM ('MARKET','LIMIT');
CREATE TYPE order_status AS ENUM ('NEW','PARTIALLY_FILLED','FILLED','CANCELED','REJECTED');

CREATE TABLE orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  symbol TEXT NOT NULL CHECK (symbol IN ('BTC-USD','ETH-USD')),
  side order_side NOT NULL,
  type order_type NOT NULL,
  price NUMERIC(30,10),         -- null for MARKET
  qty NUMERIC(30,10) NOT NULL,  -- base quantity (BTC/ETH)
  filled_qty NUMERIC(30,10) NOT NULL DEFAULT 0,
  status order_status NOT NULL DEFAULT 'NEW',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotency & outbox
CREATE TABLE idempotency_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  idem_key TEXT NOT NULL,
  request_fingerprint TEXT NOT NULL, -- hash(body + critical headers)
  response_code INT NOT NULL,
  response_body JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, idem_key)
);

CREATE TABLE outbox (
  id BIGSERIAL PRIMARY KEY,
  topic TEXT NOT NULL,      -- e.g., 'trades:BTC-USD'
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ
);

-- Create system accounts for each user (USD, BTC, ETH)
CREATE OR REPLACE FUNCTION create_user_accounts()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO accounts (user_id, currency) VALUES
    (NEW.id, 'USD'),
    (NEW.id, 'BTC'),
    (NEW.id, 'ETH');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_user_accounts_trigger
  AFTER INSERT ON users
  FOR EACH ROW
  EXECUTE FUNCTION create_user_accounts();
