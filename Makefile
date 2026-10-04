DOCS_PORT ?= 8000

.PHONY: docs test lint

docs: ## serve the course at http://localhost:$(DOCS_PORT)
	@echo "Course: http://localhost:$(DOCS_PORT)"
	python3 -m http.server $(DOCS_PORT) --bind 127.0.0.1 --directory docs

test: ## fast tests (domain, application, http, contract, arch)
	go test ./...

lint: ## golangci-lint (v2)
	golangci-lint run
