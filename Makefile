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
	$(GO) build -o $(BINARY) ./cmd/agent

# Run tests
.PHONY: test
test:
	$(GO) test ./... -v

# Refresh project context docs (see AGENTS.md §7)
.PHONY: context-refresh
context-refresh:
	@echo "[context-refresh] Analyze recent changes and update docs/CONTEXT.md." \
	 && echo "Use: agent_tool_call: update-context --auto (preferred in agent flows)."

# Run the server
.PHONY: run
run:
	$(GO) run ./cmd/agent

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
