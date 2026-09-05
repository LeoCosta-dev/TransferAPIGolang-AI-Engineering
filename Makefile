GO_MODULE_PREFIX := github.com/lcosta/TransferAPIGolang
GO_VERSION := 1.27.1

.PHONY: service

GO_MODULE_PREFIX := github.com/lcosta/TransferAPIGolang
GO_VERSION := 1.27.1

.PHONY: service

service:
	@test -n "$(name)" || (echo "Uso: make service name=nome-do-servico"; exit 1)
	@test ! -d "services/$(name)" || (echo "Erro: services/$(name) já existe"; exit 1)

	mkdir -p services/$(name)/cmd/$(name)
	mkdir -p services/$(name)/internal/domain
	mkdir -p services/$(name)/internal/application
	mkdir -p services/$(name)/internal/persistence/mongodb
	mkdir -p services/$(name)/internal/transport/http
	mkdir -p services/$(name)/docs

	touch services/$(name)/AGENTS.md
	touch services/$(name)/cmd/$(name)/main.go

	printf '# %s Service — Requirements\n\n## Purpose\n\nDescribe the purpose of the service.\n\n## Responsibilities\n\nDefine what this service owns.\n\n## Non-responsibilities\n\nDefine what this service must not own.\n\n## Business Rules\n\nDescribe the business rules.\n\n## Validation\n\nDescribe required validations.\n\n## Persistence\n\nDescribe persistence requirements.\n\n## Consistency and Concurrency\n\nDescribe consistency and concurrency requirements.\n\n## Idempotency\n\nDescribe idempotency requirements, when applicable.\n\n## Testing Requirements\n\nDescribe the minimum required test coverage.\n\n## Acceptance Criteria\n\nDefine when the service is considered complete.\n' > services/$(name)/docs/requirements.md

	printf '# %s Service — Domain\n\n## Domain Model\n\nDescribe the entities and value objects owned by the service.\n\n## Invariants\n\nDescribe the invariants that must always be preserved.\n\n## Business Operations\n\nDescribe the domain operations and their rules.\n\n## State Transitions\n\nDescribe valid state transitions, when applicable.\n\n## Errors\n\nDescribe domain errors and their meaning.\n' > services/$(name)/docs/domain.md

	printf '# %s Service — API\n\n## Base Path\n\n`/api/v1`\n\n## Resources\n\nDescribe the resources exposed by the service.\n\n## Endpoints\n\nDocument each endpoint, request, response and HTTP status.\n\n## Validation\n\nDocument transport-level validation rules.\n\n## Errors\n\nAll errors use:\n\n```json\n{\n  \"error\": \"Descrição detalhada do erro em português\"\n}\n```\n\n## HTTP Status Codes\n\n| Status | Meaning |\n|---|---|\n| 200 | Successful operation |\n| 201 | Resource created |\n| 400 | Invalid request |\n| 404 | Resource not found |\n| 409 | Business conflict |\n| 500 | Internal server error |\n\n## API and Domain Separation\n\nHTTP handlers must not contain business rules or access persistence directly.\n' > services/$(name)/docs/api.md

	printf '# %s Service\n\n## Purpose\n\nDescribe the purpose of this service.\n\n## Local Development\n\nDocument how to run the service locally.\n\n## Configuration\n\nDocument required configuration.\n' > services/$(name)/README.md

	printf '# %s Service — Agent Instructions\n\n## Required Reading\n\nBefore making changes, read:\n\n- `AGENTS.md` at the repository root;\n- `docs/engineering.md`;\n- `docs/conventions.md`;\n- `services/%s/docs/requirements.md`;\n- `services/%s/docs/domain.md`;\n- `services/%s/docs/api.md`.\n\n## Rules\n\nFollow the repository engineering workflow.\nDo not invent requirements.\nDo not change architectural boundaries without explicit approval.\nDo not introduce unnecessary dependencies.\nDo not refactor unrelated code.\n' > services/$(name)/AGENTS.md $(name) $(name) $(name)

	printf 'module $(GO_MODULE_PREFIX)/services/$(name)\n\ngo $(GO_VERSION)\n' > services/$(name)/go.mod

	go work use ./services/$(name)

	@echo "Microserviço '$(name)' criado com sucesso."


.PHONY: run-account run-transactions run

run-account:
	@go run ./services/account/cmd/account

run-transactions:
	@go run ./services/transactions/cmd/transactions

run:
	@trap 'kill 0' INT TERM EXIT; \
	go run ./services/account/cmd/account & \
	go run ./services/transactions/cmd/transactions & \
	wait