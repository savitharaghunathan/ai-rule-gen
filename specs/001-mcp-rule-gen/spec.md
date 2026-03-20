# Feature Specification: Phase 1 MCP Server for AI-Powered Konveyor Rule Generation

**Feature Branch**: `001-mcp-rule-gen`
**Created**: 2026-03-19
**Status**: Draft
**Input**: User description: "Phase 1 MCP server for AI-powered Konveyor rule generation"

## User Scenarios & Testing *(mandatory)*

### User Story 1a - Interactive Rule Construction (Priority: P1)

A rule author uses Claude Code, Cursor, or Kai connected to the MCP server to generate Konveyor analyzer rules. The client LLM reads a migration guide (URL, file, or text) and identifies migration patterns. For each pattern, it calls the `construct_rule` tool with explicit parameters (ruleID, condition type, pattern, location, message, category, effort, labels, links). The tool validates the parameters and returns valid YAML. The author calls `validate_rules` to verify the output. No server-side LLM or API key is needed — the client's LLM does all the thinking.

**Why this priority**: This is the primary interactive workflow. Works in any MCP client without server configuration. Follows Scribe's proven "parametric collapse" pattern.

**Independent Test**: Connect Claude Code to the MCP server, give it a migration guide, and verify it calls `construct_rule` to produce valid rules.

**Acceptance Scenarios**:

1. **Given** a migration guide, **When** the client LLM calls `construct_rule` with correct parameters for a java.referenced condition, **Then** the tool returns valid YAML with the correct condition structure.
2. **Given** `construct_rule` is called with all 12 condition types, **Then** each returns correctly structured YAML.
3. **Given** `construct_rule` is called with invalid parameters (bad location, bad category, invalid regex), **Then** the tool returns specific validation errors.
4. **Given** the client LLM calls `get_help` with topic "condition_types", **Then** it returns documentation listing all condition types with their required/optional fields.
5. **Given** the client LLM calls `construct_ruleset` with name, description, and labels, **Then** it returns valid ruleset YAML.
6. **Given** a set of constructed rules, **When** the client calls `validate_rules`, **Then** it returns validation results with no errors.

---

### User Story 1b - Pipeline Rule Generation (Priority: P1)

A rule author or CI/CD pipeline calls `generate_rules` with a migration guide URL, source, target, and language. The tool runs an end-to-end pipeline: ingests content, uses a server-side LLM to extract migration patterns, deterministically constructs valid YAML rules, validates, and saves to disk. Requires `RULEGEN_LLM_PROVIDER` + provider API key on the server.

**Why this priority**: Enables automated/batch rule generation. Same core value as US1a but fully automated.

**Independent Test**: Configure `RULEGEN_LLM_PROVIDER=anthropic` with API key, call `generate_rules` with a real migration guide URL, verify output rules are structurally valid.

**Acceptance Scenarios**:

1. **Given** a migration guide URL, **When** a user calls `generate_rules` with source, target, and language, **Then** the server ingests, extracts patterns via LLM, and returns valid YAML rules saved to disk.
2. **Given** code snippets (before/after examples), **When** a user calls `generate_rules`, **Then** the server generates rules that detect the "before" patterns.
3. **Given** a large migration guide, **When** content exceeds LLM context limits, **Then** the server chunks the content and processes it in parts.
4. **Given** `RULEGEN_LLM_PROVIDER` is not set, **When** a user calls `generate_rules`, **Then** the tool returns a clear error: "LLM provider not configured. Set RULEGEN_LLM_PROVIDER and provider API key."
5. **Given** generated rules, **Then** rules and ruleset metadata are saved to `output/<source>-to-<target>/rules/`.

---

### User Story 2 - Generate and Run Tests for Rules (Priority: P2)

A rule author has generated rules and wants to verify they actually detect the intended code patterns. The `generate_test_data` tool scaffolds the `.test.yaml` file, creates the directory structure, and uses server-side LLM to produce compilable source code that triggers the rules (following ARG's template-driven pipeline). The `run_tests` tool executes `kantra test` and reports pass/fail results per rule.

**Why this priority**: Rules without tests have no confidence guarantee. Test generation proves the rules actually work against real code patterns.

**Independent Test**: Take a set of generated Java rules, run the test generation pipeline, execute `kantra test`, and verify at least 70% of rules have passing tests on the first attempt.

**Acceptance Scenarios**:

1. **Given** a set of rules, **When** a user calls `generate_test_data`, **Then** the server scaffolds the `.test.yaml` file and data directory structure, uses server-side LLM to generate compilable code, extracts code blocks, validates language, injects imports, and writes test project files.
2. **Given** generated test data, **When** a user calls `run_tests`, **Then** the server executes `kantra test` and returns structured results with pass/fail per rule.
3. **Given** test failures, **When** `run_tests` is called with `max_iterations > 0`, **Then** the server autonomously analyzes failures, uses LLM to regenerate test data with improved hints, and re-runs tests. Only test data is fixed, not rules.
4. **Given** the autonomous test-fix loop, **Then** the pass rate improves from ~70% to 95%+ within 2-3 iterations.

---

### User Story 3 - Score Confidence on Generated Rules (Priority: P3)

A rule author wants an independent quality assessment of their generated rules. The `score_confidence` tool evaluates each rule using an adversarial LLM-as-judge prompt in a fresh context (no generation history). It scores pattern correctness, message quality, category appropriateness, effort accuracy, and false positive risk, returning a verdict (accept/review/reject) with evidence.

**Why this priority**: Confidence scoring builds trust in AI-generated rules. Without it, users must manually review every rule.

**Independent Test**: Score a batch of known-good rules and known-bad rules. Good rules should receive "accept" verdicts. Bad rules should receive "review" or "reject" verdicts with specific evidence.

**Acceptance Scenarios**:

1. **Given** a rules YAML file, **When** a user calls `score_confidence`, **Then** the server evaluates each rule against the rubric and returns per-rule scores, verdict, and evidence.
2. **Given** a rule with a pattern that will match too broadly, **When** scored, **Then** the false_positive_risk score is low (1-2) with evidence citing the issue.
3. **Given** the scoring prompt, **Then** it contains only the raw rule YAML and the rubric — no migration guide or generation context.

---

### User Story 4 - Run E2E Pipeline from CLI (Priority: P4)

A DevOps engineer integrates rule generation into a CI/CD pipeline. They run `rulegen generate --guide-url <URL> --source <s> --target <t> --language <l> --output <path>`. The CLI calls internal packages directly (no MCP protocol), uses a configured LLM provider, and produces rules + tests + confidence scores.

**Why this priority**: Pipeline integration enables batch rule generation and automation, but requires all other features to work first.

**Independent Test**: Run the CLI command with a real migration guide URL and verify the output directory contains valid rules, test files, and confidence scores.

**Acceptance Scenarios**:

1. **Given** `RULEGEN_LLM_PROVIDER` and API key configured, **When** a user runs `rulegen generate`, **Then** the CLI produces rules, tests, and confidence scores in the output directory.
2. **Given** no LLM provider configured, **When** a user runs the CLI, **Then** it exits with a clear error message.
3. **Given** CLI output, **When** the output directory is submitted as a PR to konveyor/rulesets, **Then** the directory structure matches the expected layout.

---

### Prerequisites

- **Go 1.22+**: Required to build and run the MCP server / CLI
- **kantra**: Must be installed on the machine where the MCP server runs. Required for `run_tests` tool.

### Edge Cases

- What happens when the migration guide URL returns a 404 or is behind authentication?
- What happens when the LLM generates code in the wrong language?
- What happens when `RULEGEN_LLM_PROVIDER` is not set and user runs a CLI pipeline command?
- What happens when `kantra` is not installed and `run_tests` is called?
- What happens when a rule has a regex pattern that is syntactically invalid?
- What happens when the migration guide contains no actionable migration patterns?
- What happens when two rules in the same file have duplicate ruleIDs?
- What happens when `construct_rule` is called with an unsupported condition type?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose 4 deterministic MCP tools over SSE transport: `construct_rule`, `construct_ruleset`, `validate_rules`, `get_help`. These require no server-side LLM. Pipeline capabilities (`generate_rules`, `generate_test_data`, `run_tests`, `score_confidence`) are CLI-only and require `RULEGEN_LLM_PROVIDER` + provider API key.
- **FR-002**: System MUST produce rule YAML parseable by analyzer-lsp's rule parser
- **FR-003**: System MUST support all Konveyor condition types: java.referenced (with all locations), java.dependency, go.referenced, go.dependency, nodejs.referenced, csharp.referenced, builtin.filecontent, builtin.file, builtin.xml, builtin.json, builtin.hasTags, builtin.xmlPublicID, and/or combinators
- **FR-004**: MCP tools MUST require no server-side LLM or API key. CLI pipeline commands MUST support multiple providers: Anthropic, OpenAI, Google Gemini, and Ollama (local models). Provider configured via `RULEGEN_LLM_PROVIDER` env var.
- **FR-005**: System MUST provide a CLI entry point that calls internal packages directly with a server-side LLM for pipeline/CI use. Pipeline capabilities (generate, test, score) are exposed only via CLI, not as MCP tools.
- **FR-006**: System MUST validate rules for: valid YAML, required fields (ruleID, when, message or tag), valid category values, effort range (1-10), regex pattern validity, label format (konveyor.io/source=, konveyor.io/target=), duplicate ruleIDs
- **FR-007**: System MUST generate test data following ARG's pipeline: build prompt from templates, call LLM, extract code blocks, validate language, inject missing imports, create config files, write test project
- **FR-008**: System MUST generate `.test.yaml` files compatible with `kantra test`
- **FR-009**: System MUST score confidence using an adversarial rubric with 5 criteria, producing accept/review/reject verdicts with evidence
- **FR-010**: System MUST output rules in the directory structure matching the konveyor/rulesets repo
- **FR-011**: System MUST support ingestion from URLs (HTML to markdown), files, and raw text
- **FR-012**: System MUST define all LLM prompts as Go templates in a templates/ directory

### Key Entities

- **Rule**: A single Konveyor analyzer rule with ruleID, description, category, effort, labels, message, links, when condition, customVariables, tag
- **Ruleset**: Metadata for a collection of rules — name, description, labels
- **Condition**: A provider-specific condition (java.referenced, builtin.filecontent, etc.) or a combinator (and/or)
- **MigrationPattern**: An extracted old-to-new API mapping with description, category, effort, language, provider, code examples
- **ConfidenceResult**: Per-rule scoring with criterion scores (1-5), overall score, verdict (accept/review/reject), evidence citations
- **TestsFile**: A kantra test definition with rulesPath, providers, and test cases
- **construct_rule tool**: Deterministic rule constructor — takes explicit parameters, returns valid YAML (Scribe-style parametric collapse)

## Out of Scope (Phase 1) — Planned Follow-ups

- **OpenRewrite Recipe Ingestion (Phase 1.5)**: ARG supports ingesting OpenRewrite recipes and converting transformation logic into Konveyor detection rules. Phase 1 focuses on migration guide ingestion; OpenRewrite support will be added as a follow-up.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A rule author can go from a migration guide URL to a validated set of rules in under 10 minutes of interactive use
- **SC-002**: Generated rules pass structural validation (validate_rules) with zero errors for at least 95% of rules
- **SC-003**: Generated test data produces passing `kantra test` results for at least 70% of rules on the first attempt
- **SC-004**: Confidence scoring correctly identifies intentionally bad rules as "review" or "reject" at least 80% of the time
- **SC-005**: The output directory structure is directly submittable as a PR to konveyor/rulesets
- **SC-006**: At least 3 migration paths are successfully demonstrated end-to-end
- **SC-007**: The MCP server starts and responds to tool calls within 2 seconds (excluding LLM inference time)
- **SC-008**: The CLI pipeline can process a migration guide and produce output without any interactive prompts
- **SC-009**: `construct_rule` works in any MCP client (Claude Code, Cursor, Kai) without server-side LLM configuration
