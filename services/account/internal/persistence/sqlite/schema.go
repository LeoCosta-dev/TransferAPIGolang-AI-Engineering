package sqlite

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    document TEXT NOT NULL UNIQUE CHECK (length(trim(document)) > 0),
    balance INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'BLOCKED', 'CLOSED')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS idempotency_operations (
    idempotency_key TEXT PRIMARY KEY
        CHECK (length(trim(idempotency_key)) > 0)
        CHECK (length(idempotency_key) <= 255),
    account_id TEXT NOT NULL,
    operation_type TEXT NOT NULL
        CHECK (operation_type IN ('CREDIT', 'DEBIT')),
    amount INTEGER NOT NULL CHECK (amount > 0),
    resulting_balance INTEGER NOT NULL CHECK (resulting_balance >= 0),
    created_at TEXT NOT NULL,
    FOREIGN KEY (account_id) REFERENCES accounts(id)
);`
