#  Service — API

## Base Path

`/api/v1`

## Resources

Describe the resources exposed by the service.

## Endpoints

Document each endpoint, request, response and HTTP status.

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
