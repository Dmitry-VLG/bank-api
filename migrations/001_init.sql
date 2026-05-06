CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (balance_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);

CREATE TABLE IF NOT EXISTS cards (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    number_encrypted BYTEA NOT NULL,
    expiry_encrypted BYTEA NOT NULL,
    cvv_hash TEXT NOT NULL,
    data_hmac TEXT NOT NULL,
    last4 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cards_user_id ON cards(user_id);
CREATE INDEX IF NOT EXISTS idx_cards_account_id ON cards(account_id);

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_account_id BIGINT REFERENCES accounts(id),
    to_account_id BIGINT REFERENCES accounts(id),
    type TEXT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_created ON transactions(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions(to_account_id);

CREATE TABLE IF NOT EXISTS credits (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    principal_cents BIGINT NOT NULL CHECK (principal_cents > 0),
    annual_rate NUMERIC(8, 4) NOT NULL,
    term_months INT NOT NULL CHECK (term_months > 0),
    monthly_payment_cents BIGINT NOT NULL CHECK (monthly_payment_cents > 0),
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_credits_user_id ON credits(user_id);
CREATE INDEX IF NOT EXISTS idx_credits_account_id ON credits(account_id);

CREATE TABLE IF NOT EXISTS payment_schedules (
    id BIGSERIAL PRIMARY KEY,
    credit_id BIGINT NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    due_date DATE NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    penalty_cents BIGINT NOT NULL DEFAULT 0 CHECK (penalty_cents >= 0),
    status TEXT NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payment_schedules_due ON payment_schedules(status, due_date);
CREATE INDEX IF NOT EXISTS idx_payment_schedules_credit_id ON payment_schedules(credit_id);
CREATE INDEX IF NOT EXISTS idx_payment_schedules_account_id ON payment_schedules(account_id);