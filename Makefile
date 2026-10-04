DOCS_PORT ?= 8000

.PHONY: docs test lint infra down reset

docs: ## serve the course at http://localhost:$(DOCS_PORT)
	@echo "Course: http://localhost:$(DOCS_PORT)"
	python3 -m http.server $(DOCS_PORT) --bind 127.0.0.1 --directory docs

test: ## fast tests (domain, application, http, contract, arch)
	go test ./...

lint: ## golangci-lint (v2)
	golangci-lint run

infra: ## only the infrastructure (postgres), for running the app from your IDE
	docker compose up -d --wait postgres

down: ## stop everything (keep data)
	docker compose down

reset: ## stop everything and wipe volumes
	docker compose down -v
