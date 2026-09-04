# AGENTS.md

## Repository

This repository is a monorepo containing multiple microservices.

Each microservice must be independently maintainable and testable.

## Before making changes

Before modifying code:

1. Identify the target microservice.
2. Read the root documentation relevant to the task.
3. Read the target service's `AGENTS.md`.
4. Inspect existing code and tests.
5. Understand dependencies affected by the change.

## Scope

Keep changes limited to the target microservice whenever possible.

Do not modify unrelated services.

If a cross-service change is necessary, identify the dependency and
validate the impact.

## Engineering principles

- Prefer simple solutions.
- Follow idiomatic Go.
- Keep responsibilities separated.
- Do not introduce unnecessary abstractions.
- Do not silently ignore errors.
- Preserve existing contracts.
- Prefer explicit behavior over implicit behavior.

## Engineering loop

Every implementation task must follow:

1. Analyze
2. Plan
3. Implement
4. Format
5. Test
6. Static analysis
7. Inspect failures
8. Fix root cause
9. Regression test
10. Verify contract

A task is not complete while required validations are failing.

## Failure rules

Never:

- delete a failing test;
- weaken a test to make it pass;
- disable validation;
- ignore errors;
- hide failures;
- modify unrelated services without justification.

When a validation fails, identify and fix the root cause.

## Completion

Do not report a task as complete unless the relevant tests and
validations pass.