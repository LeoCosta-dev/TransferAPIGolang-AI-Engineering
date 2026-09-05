# TransferAPI Go — AI-Assisted Engineering

A financial transaction API built with Go as an experiment in **AI-assisted software engineering**.

The project explores how generative AI and coding agents can participate in a structured software development process while keeping architecture, requirements, business rules, technical decisions, and code review under human responsibility.

> **This project is not intended to demonstrate that AI can generate an application. It is intended to demonstrate how AI can be used as an engineering tool.**

## Objectives

The project has two main objectives:

1. Build a backend system using Go and microservice-oriented architecture.
2. Explore a disciplined workflow for developing software with generative AI and coding agents.

The project emphasizes:

* explicit requirements;
* documented domain rules;
* API contracts;
* separation of responsibilities;
* incremental implementation;
* automated validation;
* human review of AI-generated changes.

## AI-Assisted Engineering

AI agents are used throughout the development process to assist with:

* requirements analysis;
* technical documentation;
* architecture analysis;
* code generation;
* code review;
* test generation;
* refactoring suggestions;
* static analysis and problem investigation.

AI does not replace the engineering decisions of the project.

Human responsibility remains over:

* requirements;
* business rules;
* architecture;
* technology choices;
* acceptance criteria;
* review of generated code;
* validation of the final implementation.

The repository documentation is therefore treated as an important source of context for AI agents working on the project.

## Architecture

The repository follows a monorepo structure in which each directory under `services/` represents an independently deployable service.

Current services:

```text
services/
├── account/
└── transactions/
```

The Account Service is responsible for:

* account creation;
* account retrieval;
* account updates;
* account status management;
* account status management.

The Transactions Service is responsible for balances, credits, debits,
movement history and idempotency. Both services persist in MongoDB Atlas using
`MONGODB_URI` and `MONGODB_DATABASE`; credentials are supplied only through
environment variables.

## Repository Structure

```text
.
├── AGENTS.md
├── README.md
├── LICENSE
├── .gitignore
├── Makefile
├── go.work
│
├── docs/
│   ├── architecture.md
│   ├── engineering.md
│   └── conventions.md
│
└── services/
    └── account/
        ├── AGENTS.md
        ├── README.md
        ├── go.mod
        │
        └── docs/
            ├── requirements.md
            ├── domain.md
            └── api.md
```

As implementation evolves, the service will contain its application, domain, persistence, transport, and executable components.

## Engineering Workflow

Development follows a documented engineering loop:

```text
Requirements
     ↓
Domain Rules
     ↓
API Contract
     ↓
Implementation
     ↓
Tests
     ↓
Formatting
     ↓
Static Analysis
     ↓
Review
```

The project intentionally does not follow a TDD-first workflow.

Tests are added as part of implementation and are used to validate the behavior and contracts of the resulting system.

## Technology

Planned technologies include:

* Go;
* Echo;
* MongoDB Atlas;
* REST API;
* Docker;
* automated tests;
* static analysis.

Additional technologies may be introduced when there is a clear engineering reason.

## Domain

The services model:

* customer accounts;
* account status;
* account lifecycle and status;
* balances, credit operations, debit operations and idempotency in Transactions Service.

Money is represented using integer minor units rather than floating-point numbers.

For example:

```text
R$ 1,00  → 100
R$ 10,50 → 1050
```

The domain also enforces consistency rules such as:

* balances cannot become negative;
* monetary operations must be atomic;
* only active accounts can move money;
* closed accounts are terminal;
* concurrent operations must preserve consistency;
* repeated idempotent operations must not produce duplicate effects.

## Documentation

Repository-level engineering rules are documented in:

* `docs/architecture.md`
* `docs/engineering.md`
* `docs/conventions.md`

Service-specific behavior is documented in:

* `services/account/docs/requirements.md`
* `services/account/docs/domain.md`
* `services/account/docs/api.md`

The documentation is intentionally structured so that both developers and AI agents can use it as project context.

## Development Status

The project is currently under active development.

### Current phase

* [x] Repository structure
* [x] Engineering conventions
* [x] Architecture documentation
* [x] Account requirements
* [x] Account domain rules
* [x] Account API contract
* [ ] Domain implementation
* [ ] Application layer
* [x] MongoDB persistence
* [ ] HTTP transport
* [ ] Integration
* [ ] Automated validation
* [ ] Containerization

## Running the Project

For local MongoDB Atlas development, copy `.env.example` to `.env` and fill
only the local `MONGODB_URI`. The `.env` file is ignored by Git. Run services
from the repository root with `go run ./services/account/cmd/account` and
`go run ./services/transactions/cmd/transactions`. Shell, Docker and CI
environment variables take precedence over `.env` values.

The repository uses a Go workspace to coordinate its services.

For the Account Service:

```bash
go test ./services/account/...
go vet ./services/account/...
go test -race ./services/account/...
```

## Project Philosophy

The central idea of this repository is:

> **AI should accelerate engineering, not replace engineering thinking.**

A generated implementation is not considered correct simply because it compiles.

Changes must remain consistent with:

* documented requirements;
* domain invariants;
* API contracts;
* repository conventions;
* automated validation.

## License

This project is licensed under the MIT License.

See [`LICENSE`](LICENSE) for details.
