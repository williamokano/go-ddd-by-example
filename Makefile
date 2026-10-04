DOCS_PORT ?= 8000

.PHONY: docs
docs: ## serve the course at http://localhost:$(DOCS_PORT)
	@echo "Course: http://localhost:$(DOCS_PORT)"
	python3 -m http.server $(DOCS_PORT) --bind 127.0.0.1 --directory docs
