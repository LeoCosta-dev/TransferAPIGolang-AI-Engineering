# Transactions Service — Requirements

## Purpose

Own balances and monetary movements for existing accounts.

## Responsibilities

Credits, debits, current balance, movement records, idempotency and financial
consistency. Account lifecycle is received through the Account Service HTTP
contract and kept as a local financial-status projection.

## Non-responsibilities

Account registration, customer data and account-status mutation.

## Business Rules

Amounts are positive integer minor units. Only ACTIVE accounts may move money,
balances never become negative, and CLOSED accounts cannot move money.

## Validation

Describe required validations.

## Persistence

MongoDB Atlas is configured only with `MONGODB_URI` and `MONGODB_DATABASE`.
Balance and idempotency/movement records are written in one MongoDB transaction.
The status projection is changed in the same transactional boundary as financial
movements, so a non-ACTIVE account cannot accept a concurrent movement.

## Consistency and Concurrency

Concurrent operations must not lose updates, duplicate idempotent effects or
make a balance negative.

## Idempotency

`Idempotency-Key` is required for credits and debits. It is scoped to an
account. Equal request data replays the original result; a changed amount or
type is a conflict. Failed business operations do not retain a key.

## Testing Requirements

Describe the minimum required test coverage.

## Acceptance Criteria

Define when the service is considered complete.
