DOCS_PORT ?= 8000

.PHONY: docs
docs: ## serve the course at http://localhost:$(DOCS_PORT)
	@echo "Course: http://localhost:$(DOCS_PORT)"
	python3 -m http.server $(DOCS_PORT) --bind 127.0.0.1 --directory docs

.PHONY: docs-snippets docs-check
docs-snippets: ## regenerate the reference-solution snippets in docs/ from the solution tags
	python3 scripts/docsnippets.py

docs-check: ## fail if a snippet in docs/ no longer matches the solution tags
	python3 scripts/docsnippets.py --check
