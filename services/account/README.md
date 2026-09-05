# Account Service

## Purpose

Owns account lifecycle management, including:

* account creation;
* account lookup;
* customer-name updates;
* account status management.

The Account Service is the source of truth for account lifecycle state.

It does not persist balances or process monetary movements.

Financial operations such as credits, debits and balance management belong to the Transactions Service.

## API

Public endpoints are exposed under the `/api/v1/accounts` prefix:

```text id="n2g2zq"
POST  /api/v1/accounts
GET   /api/v1/accounts/{id}
PATCH /api/v1/accounts/{id}
PATCH /api/v1/accounts/{id}/status
```

Health check:

```text id="9n3d9x"
GET /health
```

## Internal Integration

The Account Service communicates with the Transactions Service when account lifecycle changes need to be propagated.

The service architecture separates public APIs by responsibility:

```text id="s9p5yb"
/api/v1/accounts/*       → Account Service
/api/v1/transactions/*   → Transactions Service
```

The Transactions Service queries account lifecycle information through the Account Service HTTP API.

## Account Lifecycle

Account status is managed by the Account Service.

Lifecycle rules include:

* accounts can be created in an active state;
* account status changes are controlled by the Account Service;
* closed accounts are terminal;
* financial operations must respect the current account status.

The Account Service does not modify transaction balances as part of lifecycle management.

## Persistence

The service uses MongoDB Atlas for persistence.

Required configuration:

* `MONGODB_URI`: MongoDB Atlas connection URI.
* `MONGODB_DATABASE`: database name.

No credentials or connection strings are committed to this repository.

## Configuration

`ACCOUNT_HTTP_ADDR` is optional and defaults to `:8088`.

`TRANSACTIONS_SERVICE_URL` configures the base URL used to communicate with the Transactions Service.

Example configuration:

```text id="1xq9dk"
MONGODB_URI=<mongodb-atlas-uri>
MONGODB_DATABASE=transfer_api
ACCOUNT_HTTP_ADDR=:8088
TRANSACTIONS_SERVICE_URL=http://localhost:8089
```

Environment variables already set by Docker, CI or the shell take precedence over values in `.env`.

For local development, use a local `.env` file based on `.env.example`.

The `.env` file is ignored by Git and must never be committed.

## Local Development

From the repository root:

```bash id="t0o3lo"
go run ./services/account/cmd/account
```

Run the service tests with:

```bash id="z5wqkq"
go test ./services/account/...
```

Static analysis:

```bash id="3z1ay5"
go vet ./services/account/...
```

Race detection:

```bash id="1dr9fs"
go test -race ./services/account/...
```

## Service Boundaries

The Account Service owns account identity and lifecycle.

The Transactions Service owns financial state and monetary operations.

```text id="v2b5t7"
Account Service
    │
    ├── Account identity
    ├── Customer information
    └── Account lifecycle
              │
              │ HTTP
              ▼
Transactions Service
    │
    ├── Balance
    ├── Credits
    ├── Debits
    ├── Movement records
    └── Idempotency
```

This separation prevents financial concerns from being coupled directly to account lifecycle management.

## Technology

* Go;
* Echo;
* MongoDB Atlas;
* REST API;
* Docker;
* Docker Compose.

## Documentation

Service-specific documentation is available in:

```text id="4mt0cy"
services/account/docs/
├── requirements.md
├── domain.md
└── api.md
```

Repository-level architecture and engineering conventions are documented in:

```text id="j7n4qf"
docs/architecture.md
docs/engineering.md
docs/conventions.md
```
