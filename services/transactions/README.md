# Transactions Service

## Purpose

Owns financial operations and transaction consistency, including:

* credits;
* debits;
* balances;
* movement records;
* idempotency.

The Transactions Service is responsible for enforcing financial business rules and maintaining consistency during concurrent operations.

It reads account lifecycle status through the Account Service HTTP API and does not own account lifecycle management.

## API

Public endpoints are exposed under the `/api/v1/transactions` prefix:

```text
GET  /api/v1/transactions/{id}/balance
POST /api/v1/transactions/{id}/credits
POST /api/v1/transactions/{id}/debits
```

Each financial operation requires an `Idempotency-Key` header.

Example:

```http
POST /api/v1/transactions/{id}/credits
Idempotency-Key: unique-operation-key
Content-Type: application/json
```

Request body:

```json
{
  "amount": 100
}
```

Money is represented using integer minor units.

For example:

```text
R$ 1,00  → 100
R$ 10,50 → 1050
```

## Internal API

The service exposes internal endpoints used for communication with the Account Service:

```text
POST /internal/v1/accounts/{id}/register
POST /internal/v1/accounts/{id}/status
```

These endpoints are intended for service-to-service communication and are not part of the public API.

## Business Rules

The Transactions Service enforces rules including:

* balances cannot become negative;
* only active accounts can perform financial operations;
* monetary operations must be atomic;
* concurrent operations must preserve consistency;
* repeated requests using the same idempotency key must not produce duplicate financial effects.

## Persistence

The service uses MongoDB Atlas for persistence.

Required configuration:

* `MONGODB_URI`: MongoDB Atlas connection URI.
* `MONGODB_DATABASE`: database name.
* `ACCOUNT_SERVICE_URL`: base URL of the Account Service.

Financial operations use MongoDB transactions when atomic updates across multiple documents are required.

No credentials or connection strings are committed to this repository.

## Configuration

`TRANSACTIONS_HTTP_ADDR` is optional and defaults to `:8089`.

Example configuration:

```text
MONGODB_URI=<mongodb-atlas-uri>
MONGODB_DATABASE=transfer_api
ACCOUNT_SERVICE_URL=http://localhost:8088
TRANSACTIONS_HTTP_ADDR=:8089
```

Environment variables already set by Docker, CI or the shell take precedence over `.env`.

For local development, use a local `.env` file based on `.env.example`.

The `.env` file is ignored by Git and must never be committed.

## Local Development

From the repository root:

```bash
go run ./services/transactions/cmd/transactions
```

Run the service tests with:

```bash
go test ./services/transactions/...
```

Static analysis:

```bash
go vet ./services/transactions/...
```

Race detection:

```bash
go test -race ./services/transactions/...
```

## Service Integration

The Transactions Service communicates with the Account Service when account lifecycle information is required.

The public API is intentionally separated by service prefix:

```text
/api/v1/accounts/*       → Account Service
/api/v1/transactions/*   → Transactions Service
```

This allows new Transactions endpoints to be added without requiring changes to the reverse proxy configuration, as long as they remain under the `/api/v1/transactions/*` prefix.

For example:

```text
GET /api/v1/transactions/{id}/history
```

can be introduced without modifying the gateway or reverse proxy routing.

## Technology

* Go;
* Echo;
* MongoDB Atlas;
* MongoDB transactions;
* REST API;
* Docker;
* Docker Compose.

## Documentation

Service-level API documentation is available in:

```text
services/transactions/docs/api.md
```

Repository-level architecture and engineering conventions are documented in:

```text
docs/architecture.md
docs/engineering.md
docs/conventions.md
```
