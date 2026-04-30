# ai-rule-gen Development Guidelines

Skill-first architecture. Last updated: 2026-04-16

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
  test/main.go            # Run kantra tests, stamp rules, generate report (all-in-one)
  stamp/main.go           # Update rule files with kantra pass/fail labels
  report/main.go          # Generate YAML summary report
  coverage/main.go        # Post-extraction coverage check (guide artifacts vs patterns.json)
  internal/cli/           # Shared JSON output helper
internal/
  construct/              # patterns.json → rule YAML + ruleset.yaml
  coverage/               # Coverage check: scan guide sections for artifacts, cross-ref patterns.json
  ingestion/              # URL/file/text → clean markdown, chunking
  kantraparser/           # Parse kantra test/analyze output
  testrunner/             # Run kantra tests per group, stamp, report (used by cmd/test)
  rules/                  # Rule/Condition types, builders, serializer, validator,
                          #   patterns.go (ExtractOutput/MigrationPattern),
                          #   labels.go (StampTestResults), ruleid.go (IDGenerator)
  sanitize/               # Fix illegal XML comments (LLM-generated '--')
  scaffold/               # test-scaffold: create dirs, .test.yaml, manifest.json
  workspace/              # Output directory management, report generation
agents/                   # Agent skills (agentskills.io format, SKILL.md + references/)
```

## Skill Composition

`generate-rules` is the orchestrator skill. It delegates to three sub-skills
using **invoke blocks** — declarative sections that name the skill, pass
inputs, and state expected return fields.

```text
generate-rules (orchestrator)
  ├── rule-writer        — extract patterns, produce rule YAML
  ├── test-generator     — generate test source code (3x parallel)
  └── rule-validator     — fix failing tests, verify loop
```

Each sub-skill has `## Inputs` and `## Returns` sections defining its
contract. Sub-skills are independently invocable (e.g. `/rule-writer`
works standalone).

How runtimes interpret invoke blocks: spawn a sub-agent, tell it
"read and follow `agents/<skill-name>/SKILL.md`", pass the inputs.
If the runtime supports parallel sub-agents, blocks marked
`Parallel: yes` should run concurrently.

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
go run ./cmd/test      --rules rules/ --tests tests/ [--files a.test.yaml,b.test.yaml] [--timeout 5m] [--retry-timeouts]
go run ./cmd/stamp     --rules rules/ --kantra-output "..."
go run ./cmd/report    --source src --target tgt --output report.yaml
go run ./cmd/coverage  --guide guide.md --patterns patterns.json [--language java|go]

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
