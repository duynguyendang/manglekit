# Contributing to Manglekit

Thank you for your interest in contributing to Manglekit! We welcome contributions that improve the framework, fix bugs, add features, or enhance documentation. All contributors are expected to follow this guide.

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct.html). By participating, you are expected to uphold this code. Reports of unacceptable behavior can be sent to the project maintainers.

## Reporting Bugs

- Before opening an issue, search existing issues to avoid duplicates.
- Use the [Bug report template](https://github.com/duynguyend/manglekit/issues/new?template=bug_report.md) for new issues.
- Provide steps to reproduce, expected vs. actual behavior, and environment details (Go version, OS, etc.).
- For security vulnerabilities, email the maintainers privately instead of opening a public issue.

## Development Setup

1. **Fork and Clone**:
   ```
   git clone https://github.com/your-username/manglekit.git
   cd manglekit
   git remote add upstream https://github.com/duynguyend/manglekit.git
   ```

2. **Install Dependencies**:
   ```
   go mod tidy
   # Optional: Install golangci-lint for linting
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

3. **Set Environment Variables** (see README.md for details):
   ```
   export OPENAI_API_KEY=your-key
   # Add others as needed (e.g., QDRANT_URL)
   ```

4. **Build and Run**:
   ```
   make build  # Or: go build ./...
   make run    # Or: go run ./cmd/agent
   ```

## Building and Testing

- **Build**: `make build` or `go build ./...`
- **Lint**: `make lint` or `golangci-lint run`
- **Format**: `go fmt ./...`
- **Test**: `make test` or `go test ./... -v`
  - Run with coverage: `go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out`
- **Examples**: Test examples with `go run examples/simple/main.go`

Ensure all tests pass and linting succeeds before submitting changes.

## Development Workflow

1. **Create a Branch**: Use descriptive names, e.g., `feature/add-kg-adapter` or `fix/retrieval-bug`.
   ```
   git checkout -b your-branch-name
   ```

2. **Make Changes**: Follow the [code style guidelines](#code-style) below.

3. **Commit Messages**: Use conventional commits:
   - `feat: add new KG adapter`
   - `fix: resolve retrieval timeout`
   - `docs: update README example`
   - Keep messages under 72 characters; explain "why" for non-trivial changes.

4. **Push and Open PR**: 
   ```
   git push origin your-branch-name
   ```
   - Open a Pull Request against the `main` branch.
   - Reference related issues (e.g., "Fixes #123").
   - Ensure the PR description includes what was changed and why.

## Pull Requests

- PRs must pass CI (linting, tests).
- Keep PRs focused: one feature or bug fix per PR.
- Update documentation if your changes affect the API or usage.
- For new features, discuss in an issue first.
- Maintainers will review; address feedback iteratively.
- Once approved, squash commits if needed and merge via GitHub.

## Code Style

Follow Go best practices as outlined in [AGENTS.md](AGENTS.md):
- Use `gofmt` and `go vet`.
- Wrap errors: `fmt.Errorf("context: %w", err)`.
- Keep packages focused; no global mutable state.
- Interfaces for dependencies; prefer composition over inheritance.
- Add tests for new code; aim for >80% coverage.
- Document public APIs with godoc comments.

For Mangle rules (Datalog): Keep them readable, add comments for complex logic.

## Questions

If you have questions about the codebase or contribution process, open a [Discussion](https://github.com/duynguyend/manglekit/discussions) or ask in an existing issue.

We appreciate your contributions—let's make Manglekit better together!

---

*Last updated: 2025-10-02*