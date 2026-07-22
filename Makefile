.PHONY: build run lint fmt clean help

BINARY=obsi-css-diff
GO=go

help:
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GO) build -o $(BINARY) .

run: ## Run the app (pass args after --)
	$(GO) run . -- $(filter-out $@,$(MAKECMDGOALS))

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Run gofmt on all source files
	gofmt -w .

tidy: ## Tidy Go module dependencies
	$(GO) mod tidy

clean: ## Remove build artifacts and cache
	rm -f $(BINARY)
	rm -rf .obsidian_cache
