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
* containerized deployment;
* continuous integration and delivery;
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

### Account Service

The Account Service is responsible for:

* account creation;
* account retrieval;
* account updates;
* account status management;
* account lifecycle rules.

It does not perform financial operations directly.

### Transactions Service

The Transactions Service is responsible for:

* account balance;
* credit operations;
* debit operations;
* movement history;
* idempotency;
* financial consistency rules.

Financial operations are kept separate from account lifecycle management.

### Service Communication

The services communicate through HTTP when cross-service information is required.

The Transactions Service maintains the information necessary to enforce transaction rules and uses the Account Service as the source of truth for account lifecycle state.

The public API is organized by service:

```text
/api/v1/accounts/*       → Account Service
/api/v1/transactions/*   → Transactions Service
```

This separation allows new endpoints to be added to a service without requiring changes to the reverse proxy configuration.

## API

### Account

```text
POST   /api/v1/accounts
GET    /api/v1/accounts/{id}
PATCH  /api/v1/accounts/{id}
PATCH  /api/v1/accounts/{id}/status
```

### Transactions

```text
GET  /api/v1/transactions/{id}/balance
POST /api/v1/transactions/{id}/credits
POST /api/v1/transactions/{id}/debits
```

Internal service-to-service endpoints are kept under a separate namespace:

```text
/internal/v1/accounts/*
```

These endpoints are not part of the public API.

Each financial operation requires an `Idempotency-Key` to prevent duplicate effects when requests are retried.

## Repository Structure

```text
.
├── AGENTS.md
├── README.md
├── LICENSE
├── .gitignore
├── Makefile
├── go.work
├── docker-compose.yml
├── docker-compose.prod.yml
├── docker-compose.qa.yml
│
├── .github/
│   └── workflows/
│       └── deploy.yml
│
├── docs/
│   ├── architecture.md
│   ├── engineering.md
│   └── conventions.md
│
└── services/
    ├── account/
    │   ├── AGENTS.md
    │   ├── README.md
    │   ├── go.mod
    │   ├── cmd/
    │   ├── internal/
    │   └── docs/
    │       ├── requirements.md
    │       ├── domain.md
    │       └── api.md
    │
    └── transactions/
        ├── go.mod
        ├── cmd/
        ├── internal/
        └── docs/
            └── api.md
```

Each service contains its own application, domain, persistence, transport, tests, documentation, and executable components.

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
     ↓
Containerization
     ↓
CI/CD
     ↓
Deployment
```

The project intentionally does not follow a TDD-first workflow.

Tests are added as part of implementation and are used to validate the behavior and contracts of the resulting system.

## Technology

Current technologies include:

* Go 1.27;
* Echo;
* MongoDB Atlas;
* MongoDB transactions;
* REST API;
* Docker;
* Docker Compose;
* GitHub Actions;
* GitHub Container Registry;
* Caddy;
* Oracle Cloud Infrastructure;
* DuckDNS;
* Let's Encrypt.

The services use MongoDB Atlas as their persistence layer.

Money is represented using integer minor units rather than floating-point numbers.

## Domain

The system models:

* customer accounts;
* account status;
* account lifecycle;
* balances;
* credit operations;
* debit operations;
* transaction idempotency;
* transaction consistency.

### Monetary Representation

Money is stored as integer minor units:

```text
R$ 1,00  → 100
R$ 10,50 → 1050
```

Floating-point numbers are not used for monetary values.

### Business Rules

The domain enforces consistency rules such as:

* balances cannot become negative;
* monetary operations must be atomic;
* only active accounts can move money;
* closed accounts are terminal;
* concurrent operations must preserve consistency;
* repeated idempotent operations must not produce duplicate effects.

## Data Persistence

Both services use MongoDB Atlas.

Configuration is provided through environment variables:

```text
MONGODB_URI
MONGODB_DATABASE
```

Credentials and connection strings are never stored in the repository.

Different environments use separate database names while sharing the configured MongoDB infrastructure.

For example:

```text
Production → transfer_api
QA         → transfer_api_qa
```

## Containerization

Each service has its own Dockerfile.

The services are independently containerized and published as Docker images.

Images are published to GitHub Container Registry:

```text
ghcr.io/leocosta-dev/transferapigolang-account
ghcr.io/leocosta-dev/transferapigolang-transactions
```

Docker Compose is used for local development and for the deployed environments.

## Environments

The project currently maintains two deployed environments on Oracle Cloud Infrastructure.

### Production

```text
Account       → localhost:8088
Transactions  → localhost:8089
Caddy         → HTTPS :443
```

### QA

```text
Account       → localhost:9088
Transactions  → localhost:9089
Caddy         → HTTPS :443
```

The reverse proxy routes requests according to the service prefix:

```text
/api/v1/accounts/*       → Account
/api/v1/transactions/*   → Transactions
```

The services themselves are not exposed directly to the public internet.

### Public API

The API is publicly accessible through the following domains:

| Environment | Domain                                     |
| ----------- | ------------------------------------------ |
| Production  | `https://transferapigolang.duckdns.org`    |
| QA          | `https://transferapigolang-qa.duckdns.org` |

### Production

Health check:

```text
GET https://transferapigolang.duckdns.org/health
```

Account example:

```text
POST https://transferapigolang.duckdns.org/api/v1/accounts
```

Transactions example:

```text
GET https://transferapigolang.duckdns.org/api/v1/transactions/{id}/balance
```

### QA

Health check:

```text
GET https://transferapigolang-qa.duckdns.org/health
```

Account example:

```text
POST https://transferapigolang-qa.duckdns.org/api/v1/accounts
```

Transactions example:

```text
GET https://transferapigolang-qa.duckdns.org/api/v1/transactions/{id}/balance
```

Both environments use HTTPS with certificates automatically managed by Caddy through Let's Encrypt.

The public domains are routed by Caddy according to the API service prefix:

```text
/api/v1/accounts/*       → Account Service
/api/v1/transactions/*   → Transactions Service
```

This means that adding a new endpoint under an existing service prefix does not require a reverse-proxy configuration change.

For example, adding:

```text
GET /api/v1/transactions/{id}/history
```

only requires implementing the endpoint in the Transactions Service. Caddy automatically forwards the request to the correct service.

## CI/CD

The project uses GitHub Actions to automate validation, container image publication, and deployment.

The pipeline includes:

```text
Push / Tag
    ↓
Tests
    ↓
Static Analysis
    ↓
Build Docker Images
    ↓
Push to GHCR
    ↓
Deploy to OCI
    ↓
Health Checks
```

### Validation

The pipeline executes tests and static analysis for each service.

Examples:

```bash
go test ./services/account/...
go test ./services/transactions/...

go vet ./services/account/...
go vet ./services/transactions/...
```

### QA Releases

QA releases use dedicated tags following the pattern:

```text
v-qa*.*.*
```

For example:

```text
v-qa0.0.2
```

QA images use the `qa` tag and are deployed independently from production.

### Production Releases

Production releases use semantic version tags:

```text
v*.*.*
```

Production images are published with both the release version and `latest`.

Production deployment also verifies that the release commit belongs to the `main` branch history.

## Development Status

The project is currently in an **implemented and deployed** stage.

### Completed

* [x] Repository structure
* [x] Engineering conventions
* [x] Architecture documentation
* [x] Account requirements
* [x] Account domain rules
* [x] Account API contract
* [x] Account implementation
* [x] Transactions implementation
* [x] Application layer
* [x] MongoDB persistence
* [x] HTTP transport
* [x] Inter-service communication
* [x] Idempotency
* [x] Automated tests
* [x] Race detection
* [x] Static analysis
* [x] Docker containerization
* [x] Docker Compose environments
* [x] GitHub Actions CI/CD
* [x] GitHub Container Registry
* [x] OCI deployment
* [x] QA environment
* [x] Production environment
* [x] HTTPS
* [x] Health checks

### Current Focus

The project continues to evolve through incremental improvements in:

* architecture;
* API design;
* observability;
* security;
* testing;
* CI/CD;
* infrastructure;
* AI-assisted engineering practices.

## Running Locally

The repository uses a Go workspace to coordinate its services.

### Account Service

```bash
go test ./services/account/...
go vet ./services/account/...
go test -race ./services/account/...
```

### Transactions Service

```bash
go test ./services/transactions/...
go vet ./services/transactions/...
go test -race ./services/transactions/...
```

### Running the services

The services can be started independently:

```bash
go run ./services/account/cmd/account
```

```bash
go run ./services/transactions/cmd/transactions
```

Docker Compose is also available for running the complete local environment.

Local configuration should be provided through environment variables or a local `.env` file derived from the example configuration.

The `.env` file is ignored by Git and must never contain values committed to the repository.

## Code Quality

Static analysis is enforced with `golangci-lint` v2 (fixed version `v2.13.2`, configuration in [`.golangci.yml`](.golangci.yml)), alongside `gofmt`, `go vet`, `go test` and `go test -race`.

Run the linter locally for both services:

```bash
make lint
```

A Git pre-commit hook runs the `gofmt` check, `golangci-lint` and `go test ./...` for both services before every commit (`go test -race` is reserved for CI to keep the hook fast). Activate it once per clone with:

```bash
make hooks
```

The pipeline runs the same pinned `golangci-lint` version as a mandatory gate in the `Test` job, before any build or deployment.

## Validation Philosophy

The project treats compilation as only one part of validation.

A change is considered acceptable when it remains consistent with:

* documented requirements;
* domain invariants;
* API contracts;
* repository conventions;
* automated tests;
* static analysis;
* deployment requirements.

The validation process includes:

```text
Compile
  ↓
Unit Tests
  ↓
Race Detection
  ↓
Static Analysis
  ↓
Container Build
  ↓
Deployment
  ↓
Health Check
```

## Project Philosophy

The central idea of this repository is:

> **AI should accelerate engineering, not replace engineering thinking.**

A generated implementation is not considered correct simply because it compiles.

AI-generated changes must remain subject to:

* human-defined requirements;
* explicit business rules;
* architectural constraints;
* API contracts;
* automated validation;
* human review.

The project therefore uses AI as part of an engineering process rather than treating AI-generated code as the final product.

## License

This project is licensed under the MIT License.

See [`LICENSE`](LICENSE) for details.
