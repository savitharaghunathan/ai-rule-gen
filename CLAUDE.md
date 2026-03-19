# ai-rule-gen Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-19

## Active Technologies

- Go 1.22+ + `github.com/modelcontextprotocol/go-sdk` (official MCP SDK, SSE + Streamable HTTP), `gopkg.in/yaml.v3`, `github.com/JohannesKaufmann/html-to-markdown`, `github.com/spf13/cobra` (CLI), `github.com/anthropics/anthropic-sdk-go` (001-mcp-rule-gen)

## Project Structure

```text
cmd/rulegen/        # Entry point (Cobra CLI + MCP server)
internal/           # All internal packages
  server/           # MCP server setup, 4 tool registrations
  tools/            # Tool handlers (construct, validate, help, generate, test, confidence)
  llm/              # Completer interface, LLM providers (Anthropic, OpenAI, Gemini, Ollama)
  rules/            # Rule/Condition types, builders, serializer, validator
  ingestion/        # URL/file/text ingestion, HTML→markdown, chunking
  extraction/       # MigrationPattern type, LLM pattern extraction
  generation/       # Pattern→Rule construction, rule ID generation
  testing/          # Test scaffolding, ARG-style code gen, kantra runner, fix loop
  confidence/       # LLM-as-judge scorer, rubric
  workspace/        # Output directory management
templates/          # LLM prompt templates (extraction, generation, testing, confidence)
testdata/           # Test fixtures
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

# Run MCP server
./rulegen serve --port 8080
```

## Code Style

Go 1.22+: Follow standard conventions

## Architecture

Two entry points, shared internals:
- **MCP server** (`rulegen serve`): 4 deterministic tools (construct_rule, construct_ruleset, validate_rules, get_help). No server LLM needed.
- **CLI** (`rulegen generate/test/score`): Pipeline commands with server-side LLM. Require RULEGEN_LLM_PROVIDER + API key. Not exposed as MCP tools.
- LLM providers: Anthropic, OpenAI, Gemini, Ollama (local models)

## Recent Changes

- 001-mcp-rule-gen: Official MCP SDK (`modelcontextprotocol/go-sdk`), dual tool approach (constructor + pipeline), multi-provider LLM support, own YAML rule types (not engine.Rule)
