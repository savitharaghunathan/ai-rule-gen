# ai-rule-gen Development Guidelines

Skill-first architecture. Last updated: 2026-04-15

## Architecture

**Skill-first design**: All LLM orchestration lives in agent skills (subagents).
Go code is purely deterministic — no LLM calls, no API keys, no prompt templates.

The agent (Claude Code, Cursor, Goose, etc.) reads migration guides, calls CLI
commands, and orchestrates the pipeline. The CLI does the deterministic heavy
lifting.

## Active Technologies

- Go 1.25+ with stdlib `flag` (CLI), `gopkg.in/yaml.v3`, `github.com/JohannesKaufmann/html-to-markdown`

## Project Structure

```text
cmd/
  construct/main.go       # patterns.json → rule YAML + ruleset.yaml
  validate/main.go        # Validate rule YAML files
  ingest/main.go          # Fetch migration guide → clean markdown
  scaffold/main.go        # Create test dirs, .test.yaml, manifest.json
  sanitize/main.go        # Fix illegal XML comments in a directory
  stamp/main.go           # Update rule files with kantra pass/fail labels
  report/main.go          # Generate YAML summary report
  internal/cli/           # Shared JSON output helper
internal/
  construct/              # patterns.json → rule YAML + ruleset.yaml
  ingestion/              # URL/file/text → clean markdown, chunking
  kantraparser/           # Parse kantra test/analyze output
  rules/                  # Rule/Condition types, builders, serializer, validator,
                          #   patterns.go (ExtractOutput/MigrationPattern),
                          #   labels.go (StampTestResults), ruleid.go (IDGenerator)
  sanitize/               # Fix illegal XML comments (LLM-generated '--')
  scaffold/               # test-scaffold: create dirs, .test.yaml, manifest.json
  workspace/              # Output directory management, report generation
agents/                   # Agent skills (agentskills.io format, SKILL.md + references/)
```

## CLI Commands

Each command is a standalone Go file invoked via `go run cmd/<name>.go`.
All commands are deterministic. No LLM, no API keys required.

```bash
# Run individual commands (no build step needed)
go run ./cmd/ingest    --input <url-or-file> --output guide.md
go run ./cmd/construct --patterns patterns.json --output rules/
go run ./cmd/validate  --rules rules/
go run ./cmd/scaffold  --rules rules/ --output tests/
go run ./cmd/sanitize  --dir tests/data/
go run ./cmd/stamp     --rules rules/ --kantra-output "..."
go run ./cmd/report    --source src --target tgt --output report.yaml

# Tests
go test ./internal/...  # Unit tests

# Coverage
go test -coverprofile=coverage.out ./internal/...
```

## Key Concepts

### patterns.json (ExtractOutput)
Intermediate JSON format between agent pattern extraction and `go run ./cmd/construct`.
Agent writes this; CLI reads it. Contains source, target, language, and a list of
MigrationPattern objects with fields like source_fqn, provider_type, location_type,
alternative_fqns, complexity, category, concern.

### manifest.json
Output of `go run ./cmd/scaffold`. Tells the agent what source files to generate
per test group (build file + main source file per group, with language config).

## Code Style

Go 1.25+: Follow standard conventions. No LLM dependencies anywhere in Go code.
