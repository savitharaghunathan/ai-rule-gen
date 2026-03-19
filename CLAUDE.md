# ai-rule-gen Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-19

## Active Technologies

- Go 1.22+ + `github.com/mark3labs/mcp-go` (MCP SDK, SSE), `github.com/konveyor/analyzer-lsp` (output/v1/konveyor types, parser.CreateSchema()), `github.com/konveyor-ecosystem/kantra` (pkg/testing types), `gopkg.in/yaml.v3`, `github.com/JohannesKaufmann/html-to-markdown`, `github.com/spf13/cobra` (CLI), `github.com/anthropics/anthropic-sdk-go`, `github.com/openai/openai-go` (001-mcp-rule-gen)

## Project Structure

```text
src/
tests/
```

## Commands

```bash
# Build
go build -o rulegen ./cmd/rulegen/

# Unit tests (fast, no external deps)
go test ./internal/...

# Integration tests (mock LLM, no API key)
go test -tags=integration ./internal/integration/...

# E2E tests (real LLM + kantra required)
go test -tags=e2e ./test/e2e/...

# Coverage
go test -coverprofile=coverage.out ./...

# Lint
golangci-lint run ./...
```

## Code Style

Go 1.22+: Follow standard conventions

## Recent Changes

- 001-mcp-rule-gen: Added Go 1.22+ + `github.com/mark3labs/mcp-go` (MCP SDK, SSE), `github.com/konveyor/analyzer-lsp` (output/v1/konveyor types, parser.CreateSchema()), `github.com/konveyor-ecosystem/kantra` (pkg/testing types), `gopkg.in/yaml.v3`, `github.com/JohannesKaufmann/html-to-markdown`, `github.com/spf13/cobra` (CLI), `github.com/anthropics/anthropic-sdk-go`, `github.com/openai/openai-go`

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
