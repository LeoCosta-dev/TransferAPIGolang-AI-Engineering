# Account Service

Owns account creation, lookup, customer-name updates and lifecycle status. It
does not persist balances or process monetary movements.

From the repository root, copy `.env.example` to `.env`, then fill in the
MongoDB Atlas connection string locally. Run
`go run ./services/account/cmd/account`. `ACCOUNT_HTTP_ADDR` is optional and
defaults to `:8088`. Existing environment variables take precedence over
values in `.env`; no connection credentials are stored in the repository.
