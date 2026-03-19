# Tasks: Phase 1 MCP Server for AI-Powered Rule Generation

**Input**: Design documents from `/specs/001-mcp-rule-gen/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project scaffold and Go module initialization

- [ ] T001 Create directory structure: `cmd/rulegen/`, `internal/{server,tools,llm,rules,ingestion,extraction,generation,testing,confidence,workspace}/`, `templates/{extraction,generation,testing,confidence}/`
- [ ] T002 Initialize Go module (`go.mod`) with dependencies: mcp-go, yaml.v3, cobra, html-to-markdown
- [ ] T003 [P] Create `Makefile` with build, test, lint targets
- [ ] T004 [P] Create `Containerfile` skeleton

**Checkpoint**: Project compiles with `go build ./...`

---

## Phase 2: Foundation (Blocking Prerequisites)

**Purpose**: Core types, validation, LLM abstraction, MCP server skeleton — MUST be complete before any user story

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 [P] Implement `internal/rules/types.go` — Rule, Ruleset, Condition, Link, CustomVariable structs (YAML-serializable, per data-model.md). All 12 condition types + and/or combinators.
- [ ] T006 [P] Implement `internal/rules/builder.go` — Condition builder functions for java.referenced (14 locations + annotated), java.dependency, go.referenced, go.dependency, nodejs.referenced, csharp.referenced (4 locations), builtin.filecontent, builtin.file, builtin.xml, builtin.json, builtin.hasTags, builtin.xmlPublicID, and/or combinators. Chaining fields: from, as, ignore, not.
- [ ] T007 [P] Implement `internal/rules/serializer.go` — YAML read/write for rules and rulesets. Read single file, read directory, write rules grouped by concern.
- [ ] T008 [P] Implement `internal/rules/validator.go` — Structural validation: valid YAML, required fields (ruleID, when, message or tag), valid category enum (mandatory/optional/potential), effort range, regex pattern syntax, label format (konveyor.io/source=, konveyor.io/target=), duplicate ruleIDs. Returns errors and warnings.
- [ ] T009 [P] Implement `internal/workspace/workspace.go` — Output directory management: create `output/<source>-to-<target>/` with `rules/`, `tests/`, `tests/data/`, `confidence/` subdirectories.
- [ ] T010 [P] Implement `internal/llm/completer.go` — `Completer` interface with `Complete(ctx, prompt) (string, error)`. `SamplingCompleter` (MCP sampling path). `LLMCompleter` (server-side LLM path, delegates to provider).
- [ ] T011 Implement `internal/server/server.go` — MCP server setup with mcp-go: create server, register 5 tools (generate_rules, validate_rules, generate_test_data, run_tests, score_confidence), SSE transport on configurable host:port (default localhost:8080).
- [ ] T012 Implement `internal/tools/validate.go` — `validate_rules` MCP tool handler. Input: `rules_path`. Calls `rules.Validate()`. Returns JSON with valid, errors, warnings, rule_count. (Deterministic, no LLM.)
- [ ] T013 [P] Implement `cmd/rulegen/main.go` — Cobra root command with `serve` subcommand (starts MCP server). Placeholder subcommands for `generate`, `validate`, `test`, `score`.
- [ ] T014 [P] Unit tests for `internal/rules/` — types serialization roundtrip, all builder functions, validator (valid rules, missing fields, bad regex, duplicate IDs, bad categories, bad labels).
- [ ] T015 [P] Unit tests for `internal/workspace/` — directory creation, path resolution.

**Checkpoint**: `validate_rules` works end-to-end via MCP. Server starts, accepts SSE connections, validates rule YAML files.

---

## Phase 3: User Story 1 — Generate Rules from Any Input (Priority: P1) 🎯 MVP

**Goal**: A user provides any input (URL, code snippets, changelog, text) and gets valid Konveyor rules saved to disk.

**Independent Test**: Point at a real migration guide URL, call `generate_rules`, verify output rules are structurally valid.

### Implementation

- [ ] T016 [P] [US1] Implement `internal/ingestion/html.go` — HTML to Markdown conversion using html-to-markdown library.
- [ ] T017 [P] [US1] Implement `internal/ingestion/chunker.go` — Split content into chunks that fit LLM context limits. Preserve section boundaries.
- [ ] T018 [US1] Implement `internal/ingestion/ingest.go` — Unified ingestion: detect input type (URL, file path, raw text), fetch URL content, convert HTML→markdown, read files. Returns cleaned text content. Error handling: 404, auth-required, empty content.
- [ ] T019 [P] [US1] Implement `internal/extraction/patterns.go` — `MigrationPattern` struct per data-model.md (source_pattern, target_pattern, source_fqn, location_type, rationale, complexity, category, concern, provider_type, file_pattern, example_before, example_after, documentation_url, alternative_fqns).
- [ ] T020 [P] [US1] Create `templates/extraction/extract_patterns.tmpl` — Go template for pattern extraction prompt. Input: ingested content, source, target, language. Output: structured JSON array of MigrationPatterns.
- [ ] T021 [US1] Implement `internal/extraction/extractor.go` — Pattern extraction via LLM: render template with content, call Completer, parse JSON response into []MigrationPattern. Handle chunked content (extract from each chunk, deduplicate). Error handling: no patterns found, malformed LLM response.
- [ ] T022 [P] [US1] Implement `internal/generation/ruleid.go` — Rule ID generation: `<prefix>-<number>` where number increments by 10 (00010, 00020, ...). Prefix derived from source/target.
- [ ] T023 [P] [US1] Create `templates/generation/generate_message.tmpl` — Go template for rule message generation. Input: MigrationPattern. Output: markdown message with Before/After code examples.
- [ ] T024 [US1] Implement `internal/generation/generator.go` — Deterministic rule construction: MigrationPattern → Rule. Maps provider_type + source_fqn → condition (using builders), complexity → effort, category pass-through. Groups rules by concern. Generates ruleset.yaml metadata. Calls Completer for message generation (template-driven).
- [ ] T025 [P] [US1] Create `templates/generation/generate_rules.tmpl` — Go template for rule generation guidance (used when LLM needs to help with complex condition mapping).
- [ ] T026 [US1] Implement `internal/tools/generate.go` — `generate_rules` MCP tool handler. Full pipeline: parse input JSON → ingest → extract patterns (LLM) → construct rules (deterministic) → validate → save to workspace. Returns JSON with output_path, files_written, rule_count, concerns, patterns_extracted.
- [ ] T027 [US1] Implement `internal/llm/anthropic.go` — Anthropic API client implementing the provider interface for `LLMCompleter`. Uses anthropic-sdk-go.
- [ ] T028 [P] [US1] Unit tests for `internal/ingestion/` — URL fetch mock, HTML→markdown, chunking, file reading.
- [ ] T029 [P] [US1] Unit tests for `internal/extraction/` — pattern parsing from mock LLM response, deduplication.
- [ ] T030 [P] [US1] Unit tests for `internal/generation/` — rule ID generation, pattern→rule mapping for each provider type, ruleset construction, concern grouping.
- [ ] T031 [US1] Integration test: sample migration guide text → generate_rules → validate output rules are structurally valid YAML.

**Checkpoint**: `generate_rules` works E2E. User provides a migration guide URL (or text), gets valid rules saved to `output/<source>-to-<target>/rules/`.

---

## Phase 4: User Story 2 — Generate and Run Tests for Rules (Priority: P2)

**Goal**: Generate compilable test source code and run `kantra test` with autonomous fix loop.

**Independent Test**: Take generated Java rules, run test pipeline, verify ≥70% pass on first attempt.

### Implementation

- [ ] T032 [P] [US2] Implement `internal/testing/langconfig.go` — Per-language project config: Java (pom.xml, src/main/java/...), Go (go.mod, main.go), TypeScript (package.json, src/), C# (*.csproj, Program.cs). Build file templates, source paths, import injection rules.
- [ ] T033 [P] [US2] Implement `internal/testing/scaffold.go` — Test project scaffolding: create `.test.yaml` (rulesPath, providers, test cases with atLeast:1), create data directory structure per langconfig.
- [ ] T034 [P] [US2] Create `templates/testing/main.tmpl` — Master test source generation prompt. Input: rules, language config, code hints. Output: fenced code blocks (build file, source file, config files).
- [ ] T035 [P] [US2] Create `templates/testing/java.tmpl` — Java-specific instructions: pom.xml format, package structure, import requirements, location-type-specific code patterns.
- [ ] T036 [P] [US2] Create `templates/testing/go.tmpl` — Go-specific instructions: go.mod format, package main, symbol reference patterns.
- [ ] T037 [P] [US2] Create `templates/testing/typescript.tmpl` — TypeScript-specific: package.json, import patterns.
- [ ] T038 [P] [US2] Create `templates/testing/csharp.tmpl` — C#-specific: .csproj, namespace/class patterns.
- [ ] T039 [US2] Implement `internal/testing/testgen.go` — ARG-style test source generation pipeline: extract code hints from rule patterns + message Before/After blocks → build prompt from templates → call Completer → extract fenced code blocks → validate language → inject missing imports → create config files → write to data directory.
- [ ] T040 [US2] Implement `internal/testing/runner.go` — kantra test runner: shell out to `kantra test <test-file>`, parse output for pass/fail per rule, capture structured results (ruleID, status, incidents, failure reason, debug path). Graceful error if kantra not installed.
- [ ] T041 [US2] Implement `internal/testing/fixer.go` — Test failure analysis + fix: parse kantra debug output for each failing rule, identify which pattern didn't match, use LLM to generate improved code hints, return patched hints for re-generation.
- [ ] T042 [US2] Implement `internal/tools/test_generate.go` — `generate_test_data` MCP tool handler. Input: rules_path, language. Pipeline: load rules → scaffold → testgen → write files. Returns JSON with test_yaml_path, files_written, post_processing stats.
- [ ] T043 [US2] Implement `internal/tools/test_run.go` — `run_tests` MCP tool handler. Input: test_file, max_iterations (default 3). Pipeline: run kantra → if failures and iterations remain → analyze failures → LLM generates improved hints → regenerate test data → re-run. Returns JSON with passed, failed, total, iterations_run, per-rule results, fix_history.
- [ ] T044 [P] [US2] Unit tests for `internal/testing/` — scaffold output, code block extraction, import injection, langconfig, kantra output parsing (mock).
- [ ] T045 [US2] Integration test: generated rules → generate_test_data → verify .test.yaml + data directory structure is correct.

**Checkpoint**: `generate_test_data` and `run_tests` work E2E. Test data generated, kantra runs, fix loop improves pass rate.

---

## Phase 5: User Story 3 — Score Confidence on Generated Rules (Priority: P3)

**Goal**: Independent LLM-as-judge quality scoring with adversarial rubric.

**Independent Test**: Score known-good and known-bad rules, verify verdicts match expectations.

### Implementation

- [ ] T046 [P] [US3] Implement `internal/confidence/rubric.go` — Rubric definition: 5 criteria (pattern_correctness, message_quality, category_appropriateness, effort_accuracy, false_positive_risk), scoring 1-5, verdict thresholds (accept ≥4.0, review ≥2.5, reject <2.5). `ConfidenceResult` struct per data-model.md.
- [ ] T047 [P] [US3] Create `templates/confidence/judge.tmpl` — Adversarial judge prompt: strict auditor framing, rubric criteria with examples, evidence requirement. Input: raw rule YAML only + rubric. No generation context.
- [ ] T048 [US3] Implement `internal/confidence/scorer.go` — LLM-as-judge: for each rule, render judge template with rule YAML + rubric, call Completer in fresh context, parse structured scores, compute overall, assign verdict, collect evidence. Write results to `confidence/scores.yaml`.
- [ ] T049 [US3] Implement `internal/tools/confidence.go` — `score_confidence` MCP tool handler. Input: rules_path. Pipeline: load rules → score each → save results. Returns JSON with scores_file, per-rule results, summary (accept/review/reject counts).
- [ ] T050 [P] [US3] Unit tests for `internal/confidence/` — rubric verdict calculation, score parsing from mock LLM response.
- [ ] T051 [US3] Integration test: score known-good rules (expect accept) and known-bad rules (expect review/reject).

**Checkpoint**: `score_confidence` works E2E. Rules scored independently with adversarial rubric.

---

## Phase 6: User Story 4 — CLI Pipeline (Priority: P4)

**Goal**: CLI entry point for CI/CD pipeline integration using server-side LLM.

**Independent Test**: Run CLI with migration guide URL, verify output directory has rules + tests + scores.

### Implementation

- [ ] T052 [US4] Implement `internal/llm/openai.go` — OpenAI API client implementing provider interface for `LLMCompleter`.
- [ ] T053 [P] [US4] Implement `internal/llm/google.go` — Google API client implementing provider interface for `LLMCompleter`.
- [ ] T054 [US4] Complete `cmd/rulegen/main.go` — Wire up CLI subcommands:
  - `rulegen serve` — start MCP server (already scaffolded in T013)
  - `rulegen generate --guide-url <URL> --source <s> --target <t> --language <l> --output <path>` — full pipeline (generate + validate + optionally test + score)
  - `rulegen validate --rules <path>` — validate rules
  - `rulegen test --test-file <path> --max-iterations <n>` — run tests
  - `rulegen score --rules <path>` — score confidence
  - LLM provider selection via `RULEGEN_LLM_PROVIDER` + provider-specific API key env vars. Clear error if missing.
- [ ] T055 [US4] Integration test: CLI `generate` command with mock LLM → verify output directory structure matches rulesets repo layout (rules/, tests/, tests/data/, confidence/).

**Checkpoint**: CLI produces rules + tests + confidence scores from a single command. Output is PR-ready for konveyor/rulesets.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, error handling, and E2E validation

- [ ] T056 [P] Edge case handling: 404/unreachable URL, empty content, no patterns found, kantra not installed, wrong language detection, invalid regex in rules, duplicate ruleIDs across files.
- [ ] T057 [P] README.md — Setup instructions, MCP client configuration, CLI usage, example outputs.
- [ ] T058 E2E validation: demonstrate 3 migration paths end-to-end (e.g., Spring Boot 3→4, one Go migration, one builtin/filecontent migration).
- [ ] T059 Performance check: MCP server starts and responds to tool calls within 2 seconds (excluding LLM inference).
- [ ] T060 Run quickstart.md validation — verify all commands work as documented.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundation)**: Depends on Phase 1 — BLOCKS all user stories
- **Phase 3 (US1)**: Depends on Phase 2. MVP — deliver this first.
- **Phase 4 (US2)**: Depends on Phase 2 + Phase 3 (needs rules to test)
- **Phase 5 (US3)**: Depends on Phase 2 + Phase 3 (needs rules to score). Can run in parallel with Phase 4.
- **Phase 6 (US4)**: Depends on Phases 3, 4, 5 (wires everything together for CLI)
- **Phase 7 (Polish)**: Depends on all user stories being complete

### Within Each Phase

- Tasks marked [P] can run in parallel (different files, no data dependencies)
- Unit test tasks [P] can run in parallel with each other
- Integration tests depend on their implementation tasks

### Parallel Opportunities

```
Phase 2:  T005 ─┐
          T006 ─┤
          T007 ─┤── all [P], different files
          T008 ─┤
          T009 ─┤
          T010 ─┘
          T011 ──── depends on T005, T010 (types + completer for tool registration)
          T012 ──── depends on T008, T011 (validator + server)

Phase 3:  T016 ─┐
          T017 ─┤── [P] ingestion components
          T019 ─┤
          T020 ─┤── [P] extraction + templates
          T022 ─┤
          T023 ─┘
          T018 ──── depends on T016, T017 (HTML + chunker)
          T021 ──── depends on T019, T020, T010 (patterns + template + completer)
          T024 ──── depends on T006, T021, T022 (builders + extractor + ruleid)
          T026 ──── depends on T018, T024, T009 (ingest + generator + workspace)

Phase 4+5: Can run in parallel after Phase 3
```

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Phase 1 + Phase 2 → Foundation ready
2. Complete Phase 3 → `generate_rules` + `validate_rules` work E2E
3. **STOP and VALIDATE**: Generate rules from a real migration guide
4. Demo / get feedback before proceeding

### Incremental Delivery

1. Phase 1+2 → Foundation
2. Phase 3 → Generate + Validate (MVP)
3. Phase 4 → Test generation + execution
4. Phase 5 → Confidence scoring (can overlap with Phase 4)
5. Phase 6 → CLI pipeline
6. Phase 7 → Polish + E2E validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
