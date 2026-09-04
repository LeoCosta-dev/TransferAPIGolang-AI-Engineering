# Account Service Domain

## Purpose

This document defines the domain model and business rules of the Account Service.

These rules are independent of HTTP, JSON, Echo, SQLite, or any other infrastructure concern.

The implementation must preserve all domain invariants regardless of how an operation is invoked.

## Account

An account represents a customer account managed by the Account Service.

An account contains:

* `id`;
* `name`;
* `document`;
* `balance`;
* `status`;
* `created_at`;
* `updated_at`.

### Identity

`id` uniquely identifies an account and is a UUID.

The service generates the UUID account identifier.

`name` is required and must not be empty.

`document` identifies the customer associated with the account, must not be
empty, and must be unique across accounts. It is treated as a textual
identifier; CPF/CNPJ-specific validation is not required.

Two accounts cannot have the same document.

## Monetary Values

Account balances are represented in integer minor units.

For Brazilian currency, one real is represented as `100` cents.

Examples:

* `R$ 1,00` → `100`;
* `R$ 10,50` → `1050`;
* `R$ 0,01` → `1`.

Floating-point numbers must not be used to represent monetary values.

## Initial State

When an account is created:

* `status` must be `ACTIVE`;
* `balance` must be `0`.

The initial balance cannot be supplied by the client during account creation.

## Account Status

An account can have exactly one of the following statuses:

```text
ACTIVE
BLOCKED
CLOSED
```

### ACTIVE

An active account:

* may receive credits;
* may perform debits when sufficient balance exists;
* may have its mutable cadastral data updated;
* may transition to `BLOCKED`;
* may transition to `CLOSED` only when its balance is zero.

### BLOCKED

A blocked account:

* must not perform credit operations;
* must not perform debit operations;
* must not have its balance modified;
* may have permitted cadastral data updated;
* may transition back to `ACTIVE`;
* may transition to `CLOSED` only when its balance is zero.

### CLOSED

A closed account:

* must not perform credit operations;
* must not perform debit operations;
* must not have its balance modified;
* must not return to another status;
* must not be reactivated.

`CLOSED` is a terminal state.

## Status Transitions

The following transitions are allowed:

```text
ACTIVE  → BLOCKED
BLOCKED → ACTIVE
ACTIVE  → CLOSED
BLOCKED → CLOSED
```

The following transitions are forbidden:

```text
CLOSED  → ACTIVE
CLOSED  → BLOCKED
ACTIVE  → ACTIVE
BLOCKED → BLOCKED
CLOSED  → CLOSED
```

An operation that requests a forbidden transition must fail without modifying the account.

An attempt to close an account with a non-zero balance must fail without
modifying the account.

## Balance Invariants

The following invariants must always hold:

1. Balance must never be negative.
2. Balance must be represented using integer minor units.
3. A failed monetary operation must not modify the balance.
4. A successful credit increases the balance by exactly the requested amount.
5. A successful debit decreases the balance by exactly the requested amount.
6. Monetary operations must be atomic.
7. Concurrent monetary operations must preserve the final correct balance.

## Credit

A credit operation receives a positive monetary amount.

The amount must be greater than zero.

For an account with balance `B` and credit amount `A`:

```text
new_balance = B + A
```

Credit is allowed only when the account status is `ACTIVE`.

A credit operation must either:

* complete entirely and persist the new balance; or
* fail without changing the balance.

## Debit

A debit operation receives a positive monetary amount.

The amount must be greater than zero.

For an account with balance `B` and debit amount `A`:

```text
new_balance = B - A
```

Debit is allowed only when the account status is `ACTIVE`.

The debit must be rejected when:

```text
A > B
```

A rejected debit must not modify the balance.

## Zero and Negative Amounts

The following monetary amounts are invalid:

```text
0
negative values
```

Only strictly positive monetary amounts may be used for credit and debit operations.

## Account Data Mutability

The account identifier is immutable.

The customer document is immutable after account creation.

The balance cannot be modified through general account update operations.

The status must be modified only through an operation explicitly intended to change account status.

The customer name may be updated when allowed by the API contract.

Updating mutable account data must not change unrelated domain state.

## Timestamps

`created_at` records when the account was created.

`updated_at` records the last successful modification of the account.

Creating an account sets both timestamps.

A successful modification updates `updated_at`.

A successful credit or debit updates `updated_at` after the new balance is
persisted.

A failed operation must not update timestamps.

## Atomicity

Any operation that changes account state must be atomic.

For a monetary operation, the following must be treated as a single state transition:

```text
validate operation
    ↓
verify account state
    ↓
calculate new balance
    ↓
persist new balance
```

If any step fails, the account must remain in its previous valid state.

## Concurrency

The domain must remain consistent when multiple operations are executed concurrently.

Concurrent operations must not result in:

* lost updates;
* duplicated monetary effects;
* negative balances;
* inconsistent status;
* partially applied operations.

The persistence mechanism is responsible for providing the necessary transactional guarantees.

## Idempotency

Idempotency is a property of monetary operations and is scoped to an account
and operation type (`credit` or `debit`).

An operation identified by an `Idempotency-Key` must not produce its monetary
effect more than once. Within one account, a key cannot identify different
operation types.

Repeated execution with the same account, operation type, and equivalent
payload must return the original result without applying the effect again.
Reusing the key for the same account with a different payload or operation type
is a conflict.

When a first attempt fails because of a business rule before producing a
monetary effect, the key must not be retained, allowing a later valid attempt
to use it.

The HTTP representation and persistence strategy for idempotency are defined outside the domain model.

## Domain Errors

The domain must be able to distinguish business rule violations such as:

* account not found;
* duplicate document;
* invalid status transition;
* account blocked;
* account closed;
* invalid monetary amount;
* insufficient balance.

Infrastructure failures must remain distinguishable from domain rule violations.

The HTTP layer is responsible for translating these failures into HTTP status codes and JSON responses.

## Domain Invariants Summary

At all times, the following must be true:

```text
Account ID is unique
Account ID is a UUID
Name is non-empty
Document is non-empty and unique
Balance >= 0
Balance uses integer minor units
CLOSED is terminal
Only ACTIVE accounts can move money
Credit amount > 0
Debit amount > 0
Debit amount <= current balance
Failed operations do not modify state
Successful monetary operations are atomic
Concurrent operations preserve consistency
```

Any implementation that violates one of these invariants is considered incorrect.
