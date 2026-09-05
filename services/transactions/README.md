# Transactions Service

## Purpose

Owns credits, debits, balances, movement records and idempotency. It reads
account status through the Account Service HTTP API.

## Local Development

From the repository root, copy `.env.example` to `.env`, fill in the MongoDB
Atlas connection string locally, then run `go run ./services/transactions/cmd/transactions`.
Environment variables already set by Docker, CI or the shell take precedence
over `.env`.

## Configuration

Required configuration:

- `MONGODB_URI`: MongoDB Atlas connection URI.
- `MONGODB_DATABASE`: database name.
- `ACCOUNT_SERVICE_URL`: base URL of the Account Service.

`TRANSACTIONS_HTTP_ADDR` is optional and defaults to `:8089`. No credentials
or connection strings are committed to this repository.
