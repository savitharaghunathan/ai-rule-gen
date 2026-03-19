# Implementation Plan: Phase 1 MCP Server for AI-Powered Rule Generation

**Branch**: `001-mcp-rule-gen` | **Date**: 2026-03-19 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-mcp-rule-gen/spec.md`

## Summary

Build a Go MCP server exposing 5 tools over SSE transport for AI-powered Konveyor analyzer rule generation. `generate_rules` is the primary tool — it takes any input (migration guide URL, code snippets, changelogs, text), uses LLM to extract patterns, deterministically constructs valid YAML rules, and saves to disk. `run_tests` includes an autonomous test-fix loop (fixes test data, not rules) matching ARG's approach. Internal functions (ingest, extract, construct, scaffold, fix) are shared packages, not separate MCP tools. Two entry points share identical internal packages: MCP server (sampling) and CLI (server-side LLM). Condition schemas are hardcoded structs mirroring upstream providers. OpenRewrite recipe ingestion deferred to Phase 1.5.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: `github.com/mark3labs/mcp-go` (MCP SDK, SSE), `github.com/konveyor/analyzer-lsp` (output/v1/konveyor types, parser.CreateSchema()), `github.com/konveyor-ecosystem/kantra` (pkg/testing types), `gopkg.in/yaml.v3`, `github.com/JohannesKaufmann/html-to-markdown`, `github.com/spf13/cobra` (CLI), `github.com/anthropics/anthropic-sdk-go`, `github.com/openai/openai-go`
**Storage**: Filesystem only (rules YAML, test data files, confidence scores)
**Testing**: `go test` with table-driven tests, integration tests against real migration guides
**Target Platform**: Linux, macOS (localhost by default, configurable for remote/container)
**Project Type**: MCP server + CLI tool (dual entry point, single binary)
**Performance Goals**: MCP server starts and responds to tool calls within 2 seconds (excluding LLM inference)
**Constraints**: No server-side API key required for interactive MCP use; CLI requires API key via env vars
**Scale/Scope**: 5 MCP tools, 12 condition types + combinators, 3+ migration paths demonstrated E2E

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | MCP-First | PASS | All 5 user-facing capabilities exposed as MCP tools over SSE. Internal functions (ingest, extract, construct, scaffold, fix) are shared packages used by tools. CLI shares same internal packages. |
| II | Sampling Over Server LLM | PASS | MCP path uses sampling for all LLM-requiring tools (generate_rules, generate_test_data, run_tests fix loop, score_confidence). Server-side LLM only for CLI path. Same `Completer` interface abstracts both. |
| III | Ecosystem Alignment | PASS | Imports `analyzer-lsp/output/v1/konveyor` (Category, Link) and `kantra/pkg/testing` (TestsFile, Runner, Result). Own YAML types for rules (no runtime engine.Rule). Output matches rulesets repo layout. Rule schema base from `parser.CreateSchema()`. |
| IV | Template-Driven Generation | PASS | All LLM prompts defined as Go `text/template` files in `templates/` directory. Language-specific templates for test data generation. |
| V | Test-First | PASS | Every internal package has unit tests. Integration tests validate E2E flows. Generated rules validated structurally + via confidence scoring. Test data follows ARG-style pipeline. |
| VI | Simplicity | PASS | 5 focused user-facing tools (down from 11). LLM extracts, deterministic code constructs. Tools save to disk automatically. Hardcoded condition structs (Approach B). No framework magic. |

All gates pass. No violations to track.

## Project Structure

### Documentation (this feature)

```text
specs/001-mcp-rule-gen/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (MCP tool schemas)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
ai-rule-gen/
├── cmd/
│   └── rulegen/
│       └── main.go                 # Entry point: MCP server + CLI (Cobra)
├── internal/
│   ├── server/
│   │   └── server.go               # MCP server setup, tool registration, SSE transport
│   ├── tools/
│   │   ├── generate.go             # generate_rules tool (LLM + deterministic, saves to disk)
│   │   ├── validate.go             # validate_rules tool
│   │   ├── test_generate.go        # generate_test_data tool (LLM + deterministic)
│   │   ├── test_run.go             # run_tests tool (+ autonomous test-fix loop)
│   │   └── confidence.go           # score_confidence tool (LLM)
│   ├── llm/
│   │   ├── completer.go            # Completer interface + SamplingCompleter + LLMCompleter
│   │   ├── anthropic.go            # Anthropic provider
│   │   ├── openai.go               # OpenAI provider
│   │   └── google.go               # Google provider
│   ├── rules/
│   │   ├── types.go                # Rule, Ruleset, Condition types (YAML-serializable)
│   │   ├── builder.go              # Condition builders (java, go, csharp, builtin, combinators)
│   │   ├── serializer.go           # YAML read/write
│   │   └── validator.go            # Structural validation (fields, regex, labels, duplicates)
│   ├── ingestion/
│   │   ├── ingest.go               # URL/file/text ingestion (internal, called by generate_rules)
│   │   ├── html.go                 # HTML to Markdown conversion
│   │   └── chunker.go              # Content chunking for LLM context limits
│   ├── extraction/
│   │   ├── extractor.go            # Migration pattern extraction via LLM (internal)
│   │   └── patterns.go             # MigrationPattern type
│   ├── generation/
│   │   ├── generator.go            # Pattern to Rule deterministic construction (internal)
│   │   └── ruleid.go               # Rule ID generation (increment by 10)
│   ├── testing/
│   │   ├── scaffold.go             # Test project scaffolding + .test.yaml (internal)
│   │   ├── testgen.go              # Test source generation (ARG-style pipeline)
│   │   ├── fixer.go                # Test failure analysis + LLM-driven code hint regen
│   │   ├── runner.go               # kantra test runner wrapper
│   │   └── langconfig.go           # Per-language project structure config
│   ├── confidence/
│   │   ├── scorer.go               # LLM-as-judge scorer
│   │   └── rubric.go               # Scoring rubric definition
│   └── workspace/
│       └── workspace.go            # Output directory management
├── templates/
│   ├── extraction/
│   │   └── extract_patterns.tmpl   # Pattern extraction prompt
│   ├── generation/
│   │   ├── generate_rules.tmpl     # Rule generation prompt
│   │   └── generate_message.tmpl   # Message generation prompt
│   ├── testing/
│   │   ├── main.tmpl               # Test source generation master prompt
│   │   ├── java.tmpl               # Java-specific instructions
│   │   ├── go.tmpl                 # Go-specific instructions
│   │   ├── csharp.tmpl             # C#-specific instructions
│   │   └── typescript.tmpl         # TypeScript-specific instructions
│   └── confidence/
│       └── judge.tmpl              # Adversarial judge prompt
├── go.mod
├── go.sum
├── Makefile
├── Containerfile
└── README.md
```

**Structure Decision**: Single Go project with `cmd/` + `internal/` layout following Go conventions. `internal/` packages organized by domain concern (rules, ingestion, extraction, generation, testing, confidence). Templates in `templates/` at repo root. Both MCP server and CLI are entry points into the same binary. 5 tool handlers in `tools/`, internal functions in domain packages.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
