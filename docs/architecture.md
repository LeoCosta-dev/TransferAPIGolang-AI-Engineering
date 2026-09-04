# Architecture

## Repository

This repository follows a monorepo architecture.

Each directory under `services/` represents an independently deployable
microservice.

## Services

Current services:

- Account Service

Additional services may be added in the future.

## Service boundaries

Each service owns its own business logic and persistence.

A service must not directly access another service's database.

Communication between services must occur through explicit contracts.

## Shared code

Shared code should only be introduced when there is a clear need.

Business rules specific to a service must remain inside that service.

## Local development

Each service should be independently executable during local development.