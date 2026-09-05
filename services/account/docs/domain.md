# Account Service Domain

The Account Service owns customer identity and account lifecycle, independent
of HTTP and MongoDB.

An account has a UUID `id`, non-empty `name` and `document`, a status and
creation/update timestamps. Documents are unique. The service creates accounts
as `ACTIVE` and permits only `ACTIVE → BLOCKED`, `BLOCKED → ACTIVE`,
`ACTIVE → CLOSED` and `BLOCKED → CLOSED`. `CLOSED` is terminal.

The service does not contain a balance or a monetary operation. Balances,
credits, debits and their idempotency are Transactions Service domain state.
That service uses this account status through the documented HTTP contract to
permit movements only for `ACTIVE` accounts.
