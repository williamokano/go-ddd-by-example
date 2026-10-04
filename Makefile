DOCS_PORT ?= 8000

.PHONY: docs test drills lint up infra down reset migrate generate test-integration test-e2e run

docs: ## serve the course at http://localhost:$(DOCS_PORT)
	@echo "Course: http://localhost:$(DOCS_PORT)"
	python3 -m http.server $(DOCS_PORT) --bind 127.0.0.1 --directory docs

test: ## fast tests (domain, application, http, contract, arch)
	go test ./...

test-integration: ## adapter tests against real Postgres (testcontainers; needs Docker)
	go test -tags=integration ./...

test-e2e: export HOLD_TTL = 3s
test-e2e: export HOLD_SWEEP_INTERVAL = 500ms
test-e2e: up ## the full stack in Docker Compose, driven over HTTP
	go test -tags=e2e -count=1 ./test/e2e/...

drills: up ## failure drills (8.5): stops and starts Compose services on purpose
	go test -tags=e2e,drills -count=1 -run Drill -v ./test/e2e/...

lint: ## golangci-lint (v2)
	golangci-lint run

up: ## start everything: postgres, migrate, app (http://localhost:8080)
	docker compose up -d --build --wait

infra: ## only the infrastructure (postgres, kafka + topics, kafka-ui), for running the app from your IDE
	docker compose up -d --wait postgres kafka kafka-ui
	docker compose up kafka-init

down: ## stop everything (keep data)
	docker compose down

reset: ## stop everything and wipe volumes
	docker compose down -v

DATABASE_URL ?= postgres://stagehand:stagehand@localhost:5432/stagehand?sslmode=disable
export DATABASE_URL

migrate: ## apply migrations against $$DATABASE_URL
	go run ./cmd/stagehand migrate up

generate: ## sqlc: SQL in db/queries → Go in each context's sqlcgen package
	go tool sqlc generate

run: ## the app on the host against `make infra`
	go run ./cmd/stagehand serve
