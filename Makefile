.PHONY: build run lint fmt test release-test tag tidy clean help

BINARY=ocd
GO=go
GORELEASER=goreleaser

help: ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GO) build -o $(BINARY) .

run: ## Run the app (pass args after --)
	$(GO) run . -- $(filter-out $@,$(MAKECMDGOALS))

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Run gofmt on all source files
	gofmt -w .

test: ## Run all unit tests
	$(GO) test ./...

release-test: ## Run GoReleaser locally in snapshot mode (no upload)
	CI=true $(GORELEASER) release --clean --snapshot --skip=publish

tag: ## Bump version in config, commit, create and push an annotated git tag
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo "$$CURRENT" | sed 's/^v//' | cut -d. -f1); \
	MINOR=$$(echo "$$CURRENT" | sed 's/^v//' | cut -d. -f2); \
	SUGGEST="v$$MAJOR.$$(($$MINOR + 1)).0"; \
	read -p "Enter version [$$SUGGEST]: " TAG; \
	TAG=$${TAG:-$$SUGGEST}; \
	VER=$$(echo "$$TAG" | sed 's/^v//'); \
	if git rev-parse "$$TAG" >/dev/null 2>&1; then \
		echo "Tag $$TAG already exists, pushing..."; \
	else \
		echo "Version $$VER (no config/config.go to patch)"; \
		git tag -a "$$TAG" -m "Release $$TAG" && echo "Created tag $$TAG."; \
	fi; \
	git push origin "$$TAG"

tidy: ## Tidy Go module dependencies
	$(GO) mod tidy

clean: ## Remove build artifacts and cache
	rm -f $(BINARY)
	rm -rf .obsidian_cache
