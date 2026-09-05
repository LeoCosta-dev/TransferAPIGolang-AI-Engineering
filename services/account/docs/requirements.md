# Account Service Requirements

The Account Service creates and retrieves customer accounts, updates the
mutable customer name, and changes account status. It is the authoritative
source for account ID, document, customer data and lifecycle status.

Each account has a generated UUID, non-empty name and document, unique
document, `ACTIVE`, `BLOCKED` or `CLOSED` status, and UTC creation/update
timestamps. A closed account is terminal. Invalid input and invalid status
transitions leave persisted state unchanged.

Persistence uses MongoDB Atlas through `MONGODB_URI` and `MONGODB_DATABASE`.
No secret is committed. The `document` uniqueness constraint is enforced by a
MongoDB unique index.

Balances, credits, debits, movement history, idempotency and all financial
atomicity rules are exclusively owned by Transactions Service. Account Service
does not store a balance or expose monetary endpoints. It synchronizes account
creation and status changes with Transactions Service through its HTTP contract;
Transactions rejects closing a non-zero balance and blocks movements atomically.

Tests cover account lifecycle, validation, duplicate documents, HTTP contract
and errors. Run `go test ./...`, `go test -race ./...` and `go vet ./...` from
this service directory.
