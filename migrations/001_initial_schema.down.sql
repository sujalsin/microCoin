-- Drop triggers and functions
DROP TRIGGER IF EXISTS create_user_accounts_trigger ON users;
DROP FUNCTION IF EXISTS create_user_accounts();

-- Drop tables
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS users;

-- Drop types
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS order_type;
DROP TYPE IF EXISTS order_side;
DROP TYPE IF EXISTS currency;
