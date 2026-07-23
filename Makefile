.PHONY: build run lint fmt test cover cover-html release-test tag tidy clean help watch

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
	$(MAKE) cover
	@go-test-coverage -p coverage.out -b coverage.svg 2>/dev/null || true

cover: ## Run tests with code coverage and print per-function breakdown
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./... -count=1
	@go tool cover -func=coverage.out

cover-html: ## Run tests with coverage and open HTML report in browser
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./... -count=1
	go tool cover -html=coverage.out

release-test: ## Run GoReleaser locally in snapshot mode (no upload)
	CI=true $(GORELEASER) release --clean --snapshot --skip=publish

tag: ## Bump version, commit, create and push an annotated git tag
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo "$$CURRENT" | sed 's/^v//' | cut -d. -f1); \
	MINOR=$$(echo "$$CURRENT" | sed 's/^v//' | cut -d. -f2); \
	SUGGEST="v$$MAJOR.$$(($$MINOR + 1)).0"; \
	read -p "Enter version [$$SUGGEST]: " TAG; \
	TAG=$${TAG:-$$SUGGEST}; \
	if git rev-parse "$$TAG" >/dev/null 2>&1; then \
		echo "Tag $$TAG already exists, pushing..."; \
	else \
		git tag -a "$$TAG" -m "Release $$TAG" && echo "Created tag $$TAG."; \
	fi; \
	git push origin "$$TAG"

watch: ## Start gowatch for hot-reload development
	@gowatch 2>/dev/null || echo "gowatch not installed (install with: go install github.com/silentred/gowatch@latest)"

tidy: ## Tidy Go module dependencies
	$(GO) mod tidy

clean: ## Remove build artifacts and cache
	rm -f $(BINARY)
	rm -f coverage.out
	rm -rf .obsidian_cache

snapshot: ## Test goreleaser locally (builds all platforms)
	goreleaser release --snapshot --clean
