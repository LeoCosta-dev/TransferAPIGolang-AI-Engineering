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
);`
