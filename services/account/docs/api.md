# Account Service API

Base path: `/api/v1`. JSON requests with a body require
`Content-Type: application/json`. Errors are `{ "error": "mensagem" }`.

An account response contains `id`, `name`, `document`, `status`, `created_at`
and `updated_at`. It deliberately has no balance.

- `POST /api/v1/accounts` accepts `{ "name": "...", "document": "..." }`
  and returns `201`. Duplicate documents return `409`.
- `GET /api/v1/accounts/{id}` returns `200`, or `404` for a missing account.
- `PATCH /api/v1/accounts/{id}` accepts `{ "name": "..." }`.
- `PATCH /api/v1/accounts/{id}/status` accepts `ACTIVE`, `BLOCKED` or
  `CLOSED`; invalid transitions return `409`.
- `GET /health` returns `{ "status": "ok" }`.

Financial endpoints are exposed by Transactions Service under the same
`/api/v1/accounts/{id}` resource path, not by this service.
