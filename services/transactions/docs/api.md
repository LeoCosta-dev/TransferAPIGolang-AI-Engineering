#  Service — API

## Base Path

`/api/v1`

## Resources

Balances and monetary movements for account IDs.

## Endpoints

`GET /api/v1/accounts/{id}/balance` returns the current balance.

`POST /api/v1/accounts/{id}/credits` and
`POST /api/v1/accounts/{id}/debits` accept `{ "amount": 100 }` plus a required
`Idempotency-Key` header. They return the account ID, operation type, amount
and resulting integer balance.

Account Service calls `POST /internal/v1/accounts/{id}/register` and
`POST /internal/v1/accounts/{id}/status` to synchronize the financial status
projection. The latter returns `409` when a close is requested with non-zero
balance.

## Validation

Document transport-level validation rules.

## Errors

All errors use:

```json
{
  "error": "Descrição detalhada do erro em português"
}
```

## HTTP Status Codes

| Status | Meaning |
|---|---|
| 200 | Successful operation |
| 201 | Resource created |
| 400 | Invalid request |
| 404 | Resource not found |
| 409 | Business conflict |
| 500 | Internal server error |

## API and Domain Separation

HTTP handlers must not contain business rules or access persistence directly.
