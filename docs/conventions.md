# Repository Conventions

## Purpose

This document defines the technical conventions that apply across the repository.

Service-specific documentation may add stricter conventions, but must not contradict these rules.

## Language and Tooling

* Go is the primary backend language.
* The repository uses the Go version declared by each module.
* Code must be compatible with the Go version declared in `go.mod`.
* Go source code must be formatted with `gofmt`.
* Static analysis must be performed with `go vet`.
* Dependencies must be explicitly declared in `go.mod`.
* Dependencies must not be introduced without a clear technical reason.

## Repository Structure

* Each directory under `services/` represents an independently deployable service.
* Each service owns its own business logic, persistence, configuration, and tests.
* Shared code must not be introduced prematurely.
* Service-specific business rules must remain inside the owning service.
* Cross-service communication must use explicit contracts.

## Go Package Organization

* Keep packages small and focused.
* Package names must be short, lowercase, and descriptive.
* Avoid generic package names such as `utils`, `helpers`, or `common`.
* Prefer clear dependencies between packages over hidden global state.
* Keep application entrypoints under `cmd/`.
* Keep service implementation packages under `internal/`.

## Naming

Follow standard Go naming conventions.

* Use `PascalCase` for exported identifiers.
* Use `camelCase` for unexported identifiers.
* Use descriptive names instead of abbreviations when clarity improves.
* Acronyms follow Go conventions, such as `ID`, `HTTP`, `URL`, and `API`.
* Names should describe the responsibility of the value, function, or type.

## Error Handling

* Errors must be handled explicitly.
* Do not ignore returned errors unless there is a documented reason.
* Errors should contain enough context to identify the failed operation.
* Business errors must be distinguishable from infrastructure errors.
* HTTP error responses must follow the service's documented API contract.
* Internal implementation details must not be exposed through public API errors.

## Context

* Request-scoped operations must propagate `context.Context`.
* Do not store `context.Context` in structs.
* Do not create background contexts when the caller's context should be propagated.
* Respect context cancellation and deadlines in I/O operations.

## HTTP APIs

* HTTP APIs must follow the contract documented by the service.
* Request and response structures must be explicit.
* JSON field names must be defined explicitly when they form part of the public contract.
* HTTP status codes must represent the documented outcome.
* Validation errors must be returned consistently.
* API behavior must not be inferred from implementation details.

## Data and Persistence

* Persistence must be encapsulated behind appropriate service boundaries.
* Business logic must not depend directly on database-specific implementation details when unnecessary.
* Database operations must respect transaction boundaries.
* Constraints that protect domain invariants should be enforced at the persistence layer when appropriate.
* Database migrations or schema changes must be explicit and reproducible.

## Money and Numeric Values

When representing monetary values:

* Store amounts as integer minor units, such as cents.
* Do not use floating-point types for monetary values.
* Arithmetic must preserve exact monetary values.
* The API representation of monetary values must follow the service contract.

## Concurrency

* Shared mutable state must be protected appropriately.
* Do not introduce global mutable state without a clear justification.
* Database transactions must be used when multiple operations must be atomic.
* Concurrency-sensitive behavior must be covered by tests.
* Race detection should be used for code involving shared state.

## Configuration

* Configuration must come from explicit configuration sources such as environment variables or configuration files.
* Secrets must not be committed to the repository.
* Production credentials must never be hardcoded.
* Configuration required for local development must be documented.

## Logging and Observability

* Logs should provide useful operational context.
* Do not log secrets, credentials, authentication tokens, or sensitive personal data.
* Log messages should describe the event and relevant context.
* Services should expose appropriate health information.
* Observability must not alter business behavior.

## Testing

* Tests should focus on observable behavior.
* Business rules must have automated tests.
* Important error paths must be tested.
* Tests should be deterministic.
* Tests must not depend on execution order.
* Integration tests must clearly identify their external dependencies.
* Concurrency-sensitive behavior should include race detection where appropriate.

## API and Domain Contracts

* Public behavior must be documented before implementation when the behavior is part of a service contract.
* Domain invariants must be explicit.
* Implementation must not silently introduce undocumented business rules.
* Changes to public contracts require corresponding documentation and tests.

## Dependencies

Before adding a dependency, consider:

* whether the standard library is sufficient;
* whether the dependency is actively maintained;
* whether it introduces unnecessary complexity;
* whether it is appropriate for the service's operational requirements.

Dependencies should be kept to the minimum necessary set.

## Security

* Never commit secrets, tokens, private keys, or credentials.
* Validate untrusted input at system boundaries.
* Do not trust client-provided identifiers or state without validation.
* Avoid exposing internal errors, database details, or stack traces through public APIs.
* Security-sensitive behavior must be covered by tests where applicable.

## Documentation

Documentation must remain consistent with the implementation.

When behavior changes, update the relevant documentation and tests.

Documentation should explain:

* what the system does;
* the contracts it exposes;
* important domain rules;
* operational requirements;
* relevant development workflows.

## Changes

Changes should be:

* small and focused;
* easy to review;
* consistent with existing conventions;
* covered by appropriate tests.

Avoid unrelated refactoring when implementing a specific requirement.
