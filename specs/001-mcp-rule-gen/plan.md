# Implementation Plan: Phase 1 MCP Server for AI-Powered Rule Generation

**Branch**: `001-mcp-rule-gen` | **Date**: 2026-03-19 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-mcp-rule-gen/spec.md`

## Summary

Build a Go MCP server + CLI for AI-powered Konveyor analyzer rule generation. Two entry points, shared internals:

- **MCP server** (`rulegen serve`): 4 deterministic tools (`construct_rule`, `construct_ruleset`, `validate_rules`, `get_help`) over SSE. No server-side LLM needed. The client's LLM (Claude/Cursor/Kai) does the thinking.
- **CLI** (`rulegen generate/test/score`): E2E pipelines with server-side LLM. Require `RULEGEN_LLM_PROVIDER` + provider API key. Supported providers: Anthropic, OpenAI, Gemini, Ollama.

Pipeline capabilities are CLI-only — not exposed as MCP tools. `generate` is the primary pipeline command — takes any input, uses LLM to extract patterns, deterministically constructs YAML rules, saves to disk. `test` includes an autonomous test-fix loop (fixes test data, not rules). Internal functions (ingest, extract, construct, scaffold, fix) are shared packages.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: `github.com/modelcontextprotocol/go-sdk` (official MCP SDK, SSE + Streamable HTTP), `gopkg.in/yaml.v3`, `github.com/JohannesKaufmann/html-to-markdown`, `github.com/spf13/cobra` (CLI), `github.com/anthropics/anthropic-sdk-go`
**Storage**: Filesystem only (rules YAML, test data files, confidence scores)
**Testing**: `go test` with table-driven tests, integration tests against real migration guides
**Target Platform**: Linux, macOS (localhost by default, configurable for remote/container)
**Project Type**: MCP server + CLI tool (dual entry point, single binary)
**Performance Goals**: MCP server starts and responds to tool calls within 2 seconds (excluding LLM inference)
**Constraints**: MCP tools require no server-side API key. CLI pipeline commands require `RULEGEN_LLM_PROVIDER` + provider API key.
**Scale/Scope**: 4 MCP tools + CLI pipeline commands, 12 condition types + combinators, 3+ migration paths demonstrated E2E

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | MCP-First | PASS | 4 deterministic tools exposed as MCP tools over SSE. Pipeline capabilities are CLI-only (not useful as MCP tools since the client LLM already does the thinking). CLI shares same internal packages. |
| II | Dual Mode: MCP + CLI | PASS | Deterministic MCP tools for interactive use (no server LLM). CLI pipeline commands with server-side LLM for CI/automation. Same `Completer` interface abstracts LLM providers. |
| III | Ecosystem Alignment | PASS | Own YAML types for rules (no runtime engine.Rule). Output matches rulesets repo layout. Condition types mirror upstream providers. |
| IV | Template-Driven Generation | PASS | All LLM prompts defined as Go `text/template` files in `templates/` directory. Language-specific templates for test data generation. |
| V | Test-First | PASS | Every internal package has unit tests. Integration tests validate E2E flows. Generated rules validated structurally + via confidence scoring. |
| VI | Simplicity | PASS | 8 focused tools in two clear categories. LLM extracts, deterministic code constructs. Constructor tools are pure functions. No framework magic. |

All gates pass. No violations to track.

## Project Structure

### Documentation (this feature)

```text
specs/001-mcp-rule-gen/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── test-plan.md         # Phase 1 output (per-package test plan)
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
│   │   ├── construct.go            # construct_rule, construct_ruleset tool handlers
│   │   ├── help.go                 # get_help tool handler
│   │   ├── validate.go             # validate_rules tool
│   │   ├── generate.go             # generate_rules tool (LLM + deterministic, saves to disk)
│   │   ├── test_generate.go        # generate_test_data tool (LLM + deterministic)
│   │   ├── test_run.go             # run_tests tool (+ autonomous test-fix loop)
│   │   └── confidence.go           # score_confidence tool (LLM)
│   ├── llm/
│   │   ├── completer.go            # Completer interface + LLMCompleter
│   │   ├── anthropic.go            # Anthropic provider
│   │   ├── openai.go               # OpenAI provider
│   │   ├── gemini.go               # Google Gemini provider
│   │   └── ollama.go               # Ollama provider (local models)
│   ├── rules/
│   │   ├── types.go                # Rule, Ruleset, Condition types (YAML-serializable)
│   │   ├── builder.go              # Condition builders (java, go, csharp, builtin, combinators)
│   │   ├── serializer.go           # YAML read/write
│   │   └── validator.go            # Structural validation
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
│   ├── integration/                # Integration tests (build tag: integration)
│   └── workspace/
│       └── workspace.go            # Output directory management
├── templates/
│   ├── extraction/
│   │   └── extract_patterns.tmpl   # Pattern extraction prompt
│   ├── generation/
│   │   └── generate_message.tmpl   # Message generation prompt
│   ├── testing/
│   │   └── main.tmpl               # Test source generation master prompt
│   └── confidence/
│       └── judge.tmpl              # Adversarial judge prompt
├── testdata/                       # Shared test fixtures
├── test/
│   └── e2e/                        # E2E tests (build tag: e2e)
├── go.mod
├── go.sum
├── Makefile
├── Containerfile
└── README.md
```

## Complexity Tracking

No constitution violations. No complexity justifications needed.

---

## Tasks

### Phase 1: Setup (Shared Infrastructure)

- [ ] T001 Create directory structure
- [ ] T002 Initialize Go module with dependencies
- [ ] T003 [P] Create `Makefile`
- [ ] T004 [P] Create `Containerfile` skeleton

**Checkpoint**: Project compiles with `go build ./...`

---

### Phase 2: Foundation (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 [P] Implement `internal/rules/types.go` — Rule, Ruleset, Condition structs. All 12 condition types + combinators.
- [ ] T006 [P] Implement `internal/rules/builder.go` — Condition builder functions for all condition types + chaining.
- [ ] T007 [P] Implement `internal/rules/serializer.go` — YAML read/write for rules and rulesets.
- [ ] T008 [P] Implement `internal/rules/validator.go` — Structural validation.
- [ ] T009 [P] Implement `internal/workspace/workspace.go` — Output directory management.
- [ ] T010 [P] Implement `internal/llm/completer.go` — `Completer` interface + `LLMCompleter` (delegates to provider). `Provider` interface.
- [ ] T011 Implement `internal/server/server.go` — MCP server setup: register 4 deterministic tools, SSE transport.
- [ ] T012 Implement `internal/tools/validate.go` — `validate_rules` handler.
- [ ] T012a [P] Implement `internal/tools/construct.go` — `construct_rule` and `construct_ruleset` handlers. Takes JSON params, uses `rules/builder.go` to construct Rule, marshals to YAML, validates, returns.
- [ ] T012b [P] Implement `internal/tools/help.go` — `get_help` handler. Returns documentation on condition types, locations, labels, rule format, examples.
- [ ] T013 [P] Implement `cmd/rulegen/main.go` — Cobra root + `serve` subcommand. Placeholder subcommands for `generate`, `validate`, `test`, `score`.
- [ ] T014 [P] Unit tests for `internal/rules/`
- [ ] T015 [P] Unit tests for `internal/workspace/`

**Checkpoint**: All 4 MCP tools (`construct_rule`, `construct_ruleset`, `validate_rules`, `get_help`) work E2E. Server starts, accepts connections, builds valid YAML.

---

### Phase 3: User Story 1b — Generate Rules Pipeline (Priority: P1)

**Goal**: `generate_rules` works E2E with server-side LLM.

- [ ] T016 [P] [US1b] Implement `internal/ingestion/html.go`
- [ ] T017 [P] [US1b] Implement `internal/ingestion/chunker.go`
- [ ] T018 [US1b] Implement `internal/ingestion/ingest.go`
- [ ] T019 [P] [US1b] Implement `internal/extraction/patterns.go`
- [ ] T020 [P] [US1b] Create `templates/extraction/extract_patterns.tmpl`
- [ ] T021 [US1b] Implement `internal/extraction/extractor.go`
- [ ] T022 [P] [US1b] Implement `internal/generation/ruleid.go`
- [ ] T023 [P] [US1b] Create `templates/generation/generate_message.tmpl`
- [ ] T024 [US1b] Implement `internal/generation/generator.go`
- [ ] T026 [US1b] Implement pipeline logic in `internal/tools/generate.go` — CLI-only (not an MCP tool).
- [ ] T027 [US1b] Implement `internal/llm/anthropic.go` — Anthropic provider.
- [ ] T027a [P] [US1b] Implement `internal/llm/openai.go` — OpenAI provider.
- [ ] T027b [P] [US1b] Implement `internal/llm/gemini.go` — Gemini provider.
- [ ] T027c [P] [US1b] Implement `internal/llm/ollama.go` — Ollama provider.
- [ ] T028 [P] Unit tests for `internal/ingestion/`
- [ ] T029 [P] Unit tests for `internal/extraction/`
- [ ] T030 [P] Unit tests for `internal/generation/`
- [ ] T031 Integration test: migration guide → generate_rules → valid output.

**Checkpoint**: `rulegen generate` CLI command works E2E with any configured LLM provider.

---

### Phase 4: User Story 2 — Generate and Run Tests (Priority: P2)

- [ ] T032-T045: Test data generation, kantra runner, fix loop (unchanged from original plan).

**Checkpoint**: `generate_test_data` and `run_tests` work E2E.

---

### Phase 5: User Story 3 — Score Confidence (Priority: P3)

- [ ] T046-T051: Confidence scoring with adversarial rubric (unchanged from original plan).

**Checkpoint**: `score_confidence` works E2E.

---

### Phase 6: User Story 4 — CLI Pipeline (Priority: P4)

- [ ] T054 Complete `cmd/rulegen/main.go` — Wire up CLI subcommands (`generate`, `validate`, `test`, `score`) with LLM provider selection via `RULEGEN_LLM_PROVIDER`. Pipeline capabilities are CLI-only.
- [ ] T055 Integration test: CLI `generate` → verify output.

**Checkpoint**: CLI produces rules + tests + confidence scores from a single command.

---

### Phase 7: Polish & Cross-Cutting Concerns

- [ ] T056 [P] Edge case handling
- [ ] T057 [P] README.md
- [ ] T058 E2E validation: 3 migration paths
- [ ] T059 Performance check
- [ ] T060 Quickstart validation

---

## Dependencies & Execution Order

```
Phase 1 (Setup)     → no deps
Phase 2 (Foundation) → depends on Phase 1, BLOCKS all user stories
  - T012a, T012b (constructor tools) can run in parallel with T012 (validate)
Phase 3 (US1b)      → depends on Phase 2
  - T027, T027a-c (LLM providers) can run in parallel
Phase 4 (US2)       → depends on Phase 2 + Phase 3
Phase 5 (US3)       → depends on Phase 2 + Phase 3, can run parallel with Phase 4
Phase 6 (US4)       → depends on Phases 3, 4, 5
Phase 7 (Polish)    → depends on all
```

## Implementation Strategy

### MVP First (User Story 1a + 1b)

1. Complete Phase 1 + Phase 2 → Constructor tools + validate work E2E
2. **Test interactive workflow**: Connect Claude Code/Cursor, verify construct_rule works
3. Complete Phase 3 → Pipeline tools work E2E
4. **Test pipeline**: Run generate_rules with API key
5. Demo / get feedback before proceeding
