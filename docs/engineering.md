# Engineering Workflow

## Objective

All changes must be developed through a repeatable engineering loop.

The goal is to keep changes small, understandable, testable, and aligned with the repository contracts.

## Loop

### 1. Analyze

Understand:

* the requirement;
* affected service;
* existing implementation;
* existing tests;
* relevant contracts;
* dependencies.

Do not modify code yet.

### 2. Plan

Define:

* files that need to change;
* behavior that must be implemented;
* tests that must be added or modified;
* possible risks;
* assumptions that must be resolved before implementation.

Do not implement behavior that is not defined by the relevant requirements or contracts.

### 3. Implement

Implement the smallest coherent change that satisfies the requirement.

Avoid:

* unrelated refactoring;
* unnecessary abstractions;
* speculative features;
* changing public contracts without explicit justification;
* duplicating business rules across layers.

Keep business rules explicit and testable.

### 4. Format

Format the code using the project's standard tooling.

For Go:

```bash
gofmt -w .
```

### 5. Test

Run the relevant tests after implementation.

For a service:

```bash
go test ./...
```

When concurrency or shared state is involved:

```bash
go test -race ./...
```

Tests must verify behavior, not implementation details.

### 6. Static Analysis

Run:

```bash
go vet ./...
```

Any new warning or failure must be investigated before the change is considered complete.

### 7. Review

Before finishing, verify:

* the implementation satisfies the requirement;
* public behavior matches the documented contract;
* tests cover the relevant success and failure cases;
* errors are handled explicitly;
* no unrelated files were modified;
* no undocumented behavior was introduced;
* formatting and static analysis pass.

### 8. Complete

A change is considered complete only when:

1. the implementation is consistent with the requirements;
2. relevant tests pass;
3. `go vet` passes;
4. the code is formatted;
5. the documented contracts remain accurate.

## Agent Rules

Agents must:

* read the relevant `AGENTS.md` files before modifying code;
* inspect existing code before proposing changes;
* follow repository and service-level documentation;
* prefer existing patterns over introducing new ones;
* ask for clarification when requirements conflict or are insufficient;
* avoid inventing business rules.

Agents must not:

* silently change requirements;
* weaken tests to make an implementation pass;
* remove validation without explicit justification;
* introduce dependencies without a clear reason;
* perform unrelated refactoring;
* consider a task complete solely because the code compiles.

## Scope

The repository-level workflow applies to all services.

Service-specific documentation may add stricter rules, but must not contradict the repository-level engineering workflow.
