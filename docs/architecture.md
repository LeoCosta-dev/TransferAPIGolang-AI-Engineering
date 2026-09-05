# Architecture

## Repository

This repository follows a monorepo architecture.

Each directory under `services/` represents an independently deployable
microservice.

## Services

Current services:

- Account Service: customer registration and account lifecycle/status.
- Transactions Service: balances and financial movements.

Additional services may be added in the future.

## Service boundaries

Each service owns its own business logic and persistence. The Account Service
does not store or modify balances. It publishes account creation and lifecycle
changes to Transactions through an explicit HTTP contract. Transactions keeps
its own status projection with the balance, so a status change and a financial
movement are serialized by a MongoDB transaction; it never accesses the
Account database.

A service must not directly access another service's database.

Communication between services must occur through explicit contracts.

## Shared code

Shared code should only be introduced when there is a clear need.

Business rules specific to a service must remain inside that service.

## Local development

Each service should be independently executable during local development.
