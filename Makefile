# Makefile for Manglekit

GO := go
GOLANGCI_LINT := golangci-lint
BINARY := manglekit

# Default target
.PHONY: all
all: fmt lint build test

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Lint code
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

# Build the project
.PHONY: build
build:
	$(GO) build -o $(BINARY) ./cmd/mkit

# Run tests
.PHONY: test
test:
	$(GO) test -race -count=1 ./... -v

# Run tests with coverage
.PHONY: coverage
coverage:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@THRESHOLD=60; \
	COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Current coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < $${THRESHOLD}" | bc -l) -eq 1 ]; then \
		echo "FAIL: Coverage $${COVERAGE}% is below threshold $${THRESHOLD}%"; \
		exit 1; \
	fi

# Refresh project context docs (see workspace AGENTS.md § Context maintenance)
.PHONY: context-refresh
context-refresh:
	@echo "[context-refresh] Analyze recent changes and update the context docs at ../docs/context/." \
	 && echo "Use: agent_tool_call: update-context --auto (preferred in agent flows)."

# Run the server
.PHONY: run
run:
	$(GO) run ./cmd/mkit

# Install the mkit CLI
.PHONY: install-cli
install-cli:
	$(GO) install ./cmd/mkit

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY)

# Update dependencies
.PHONY: deps
deps:
	$(GO) get -u ./...
	$(GO) mod tidy
