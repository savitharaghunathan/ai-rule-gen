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
  testgen/          # Test data generation, kantra runner, ARG-style fix loop
  confidence/       # Functional scoring (kantra test) + optional LLM-as-judge
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
- **CLI** (`rulegen generate/test/score`): Pipeline commands. `generate` and `test` require RULEGEN_LLM_PROVIDER + API key. `score` runs kantra tests (no LLM needed) with optional LLM-as-judge via `--provider`. Not exposed as MCP tools.
- LLM providers: Anthropic, OpenAI, Gemini, Ollama (local models)

## Recent Changes

- 001-mcp-rule-gen: Official MCP SDK (`modelcontextprotocol/go-sdk`), dual tool approach (constructor + pipeline), multi-provider LLM support, own YAML rule types (not engine.Rule)
- Confidence scoring: Primary signal is functional (kantra test pass/fail, matching ARG's approach). Optional secondary signal is LLM-as-judge with adversarial rubric. `rulegen score --tests <dir>` runs kantra; add `--provider` + `--rules` for LLM judge.
- Test-fix loop: `rulegen test` now generates data, runs kantra, and auto-fixes failing test data (not rules) via LLM code hints, up to `--max-iterations` (default 3). Matches ARG's pipeline. Two-phase fix: Phase A checks compilation (Go/Java/Node.js/C#) with API doc lookup, Phase B runs kantra and generates code hints for failing rules.
- Kantra Go workaround: kantra v0.9.0-alpha.6 container lacks Go toolchain, so `go.referenced` rules fail with "no views". Runner falls back to `kantra analyze --run-local` (uses host toolchain) when kantra test reports 0/total. TODO: remove fallback once kantra ships Go in the container.
