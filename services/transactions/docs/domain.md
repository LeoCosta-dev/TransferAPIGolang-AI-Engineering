# Transactions Service — Domain

## Domain Model

An account financial state contains an account ID and an integer balance. A
movement contains account ID, CREDIT or DEBIT type, amount, resulting balance
and idempotency key.

## Invariants

Balance is never negative; amounts are integer minor units and strictly
positive; only ACTIVE accounts move money.

## Business Operations

Credit adds the amount. Debit subtracts only when sufficient funds exist.

## State Transitions

Movements are immutable once persisted.

## Errors

Invalid amount, blocked account, closed account and insufficient balance are
business errors.
