# Feature Specification: Phase 1 MCP Server for AI-Powered Konveyor Rule Generation

**Feature Branch**: `001-mcp-rule-gen`
**Created**: 2026-03-19
**Status**: Draft
**Input**: User description: "Phase 1 MCP server for AI-powered Konveyor rule generation"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Generate Rules from Any Input (Priority: P1)

A rule author uses Claude Code (or Kai) connected to the MCP server to generate Konveyor analyzer rules. They provide input — a migration guide URL, code snippets (before/after), a changelog, or a text description of migration concerns — and call `generate_rules`. The tool internally ingests content, uses LLM (via MCP sampling) to extract migration patterns, then deterministically constructs valid YAML rules with proper conditions, messages, and metadata. The rules are validated and saved to disk.

**Why this priority**: This is the core value proposition — going from any migration knowledge source to working rules with minimal effort. The tool handles everything: ingestion, pattern extraction, and rule construction.

**Independent Test**: Point the tool at a real migration guide URL, call `generate_rules`, and verify the output rules are structurally valid and cover the key migration concerns described in the guide.

**Acceptance Scenarios**:

1. **Given** a migration guide URL, **When** a user calls `generate_rules` with the URL, source, target, and language, **Then** the server ingests the guide, extracts patterns via LLM, and returns valid YAML rules with correct provider conditions, messages with Before/After code examples, appropriate categories, source/target labels, and ruleset metadata.
2. **Given** code snippets (before/after examples), **When** a user calls `generate_rules` with those snippets, **Then** the server generates rules that detect the "before" patterns.
3. **Given** a changelog or release notes, **When** a user calls `generate_rules`, **Then** the server extracts breaking changes and generates rules for each.
4. **Given** a text description of a migration concern, **When** a user calls `generate_rules`, **Then** the server generates rules matching the described pattern.
5. **Given** a large migration guide, **When** content exceeds LLM context limits, **Then** the server chunks the content and processes it in parts without losing migration patterns.
6. **Given** a generated rule YAML, **When** a user calls `validate_rules`, **Then** the server returns a validation result with no errors.
7. **Given** a rule with missing required fields, **When** a user calls `validate_rules`, **Then** the server returns specific error messages identifying the missing fields.
8. **Given** generated rules, **Then** the rules and ruleset metadata are automatically saved to `output/<source>-to-<target>/rules/` in the correct directory structure.
9. **Given** no MCP client (CLI pipeline mode), **When** a user runs the CLI with `--guide-url`, **Then** the CLI uses its own LLM client to perform the same pipeline.

---

### User Story 2 - Generate and Run Tests for Rules (Priority: P2)

A rule author has generated rules and wants to verify they actually detect the intended code patterns. The `generate_test_data` tool scaffolds the `.test.yaml` file, creates the directory structure, and uses MCP sampling to produce compilable source code that triggers the rules (following ARG's template-driven pipeline). The `run_tests` tool executes `kantra test` and reports pass/fail results per rule.

**Why this priority**: Rules without tests have no confidence guarantee. Test generation proves the rules actually work against real code patterns.

**Independent Test**: Take a set of generated Java rules, run the test generation pipeline, execute `kantra test`, and verify at least 70% of rules have passing tests on the first attempt.

**Acceptance Scenarios**:

1. **Given** a set of rules, **When** a user calls `generate_test_data`, **Then** the server scaffolds the `.test.yaml` file and data directory structure, builds a language-specific prompt (using Go templates), uses MCP sampling (or server LLM) to generate compilable code, extracts code blocks from the response, validates the language, injects missing imports, and writes the test project files (build file + source file + config file).
2. **Given** generated test data, **When** a user calls `run_tests`, **Then** the server executes `kantra test` and returns structured results with pass/fail per rule, failure reasons, and debug paths.
3. **Given** test failures, **When** `run_tests` is called with `max_iterations > 0`, **Then** the server autonomously analyzes failures from kantra debug output, uses LLM to regenerate test data with improved code hints, and re-runs tests. Rules are treated as the source of truth — only test data is fixed.
4. **Given** the autonomous test-fix loop, **Then** the pass rate improves from ~70% to 95%+ within 2-3 iterations (matching ARG's real-world performance).

---

### User Story 3 - Score Confidence on Generated Rules (Priority: P3)

A rule author wants an independent quality assessment of their generated rules. The `score_confidence` tool evaluates each rule using an adversarial LLM-as-judge prompt in a fresh context (no generation history). It scores pattern correctness, message quality, category appropriateness, effort accuracy, and false positive risk, returning a verdict (accept/review/reject) with evidence.

**Why this priority**: Confidence scoring builds trust in AI-generated rules. Without it, users must manually review every rule.

**Independent Test**: Score a batch of known-good rules and known-bad rules. Good rules should receive "accept" verdicts. Bad rules (wrong patterns, vague messages, wrong categories) should receive "review" or "reject" verdicts with specific evidence.

**Acceptance Scenarios**:

1. **Given** a rules YAML file, **When** a user calls `score_confidence`, **Then** the server evaluates each rule against the rubric and returns per-rule scores (1-5 per criterion), an overall score, a verdict (accept/review/reject), and evidence citations.
2. **Given** a rule with a pattern that will match too broadly, **When** scored, **Then** the false_positive_risk score is low (1-2) with evidence citing the specific pattern issue.
3. **Given** the scoring prompt, **Then** it contains only the raw rule YAML and the rubric — no migration guide, no generation prompt, no chain-of-thought from the generation session.

---

### User Story 4 - Run E2E Pipeline from CLI (Priority: P4)

A DevOps engineer integrates rule generation into a CI/CD pipeline. They run a CLI command like `rulegen generate --guide-url <URL> --source spring-boot-3 --target spring-boot-4 --language java --output ./rulesets/`. The CLI calls the same internal packages directly (no MCP protocol), uses a configured LLM API key, and produces rules + tests + confidence scores in the rulesets repo format.

**Why this priority**: Pipeline integration enables batch rule generation and automation, but requires all other features to work first.

**Independent Test**: Run the CLI command with a real migration guide URL and verify the output directory contains valid rules, test files, test data, and confidence scores.

**Acceptance Scenarios**:

1. **Given** a configured LLM API key (env var), **When** a user runs `rulegen generate --guide-url <URL> --source <s> --target <t> --language <l> --output <path>`, **Then** the CLI produces rules, tests, and confidence scores in the output directory.
2. **Given** no LLM API key is configured, **When** a user runs the CLI, **Then** the CLI exits with a clear error message indicating the missing configuration.
3. **Given** CLI output, **When** the output directory is submitted as a PR to konveyor/rulesets, **Then** the directory structure matches the expected layout (rules/, tests/, tests/data/).

---

### Prerequisites

- **Go 1.22+**: Required to build and run the MCP server / CLI
- **kantra**: Must be installed on the machine where the MCP server runs (not the MCP client). Required for `run_tests` tool. The server shells out to `kantra test` as a subprocess.

### Edge Cases

- What happens when the migration guide URL returns a 404 or is behind authentication?
- What happens when the LLM generates code in the wrong language (e.g., Java instead of TypeScript)?
- What happens when MCP sampling is not supported by the client?
- What happens when `kantra` is not installed and `run_tests` is called?
- What happens when a rule has a regex pattern that is syntactically invalid?
- What happens when the migration guide contains no actionable migration patterns?
- What happens when two rules in the same file have duplicate ruleIDs?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose 5 MCP tools over SSE transport: generate_rules, validate_rules, generate_test_data, run_tests, score_confidence. `generate_rules` saves output to disk automatically. `run_tests` includes an autonomous test-fix loop (fixes test data, not rules). Internal functions (ingest, extract_migration_patterns, construct_rule, construct_ruleset, scaffold_test, fix_test_data) are shared packages called by tools, not exposed as separate MCP tools.
- **FR-002**: System MUST produce rule YAML parseable by analyzer-lsp's rule parser
- **FR-003**: System MUST support all Konveyor condition types: java.referenced (with all locations), java.dependency, go.referenced, go.dependency, nodejs.referenced, csharp.referenced, builtin.filecontent, builtin.file, builtin.xml, builtin.json, builtin.hasTags, builtin.xmlPublicID, and/or combinators
- **FR-004**: System MUST use MCP sampling for LLM inference when an MCP client is present — no server-side API key required for interactive use
- **FR-005**: System MUST provide a CLI entry point that calls internal packages directly with a server-side LLM for pipeline/CI use
- **FR-006**: System MUST validate rules for: valid YAML, required fields (ruleID, when, message or tag), valid category values, effort range (1-5), regex pattern validity, label format (konveyor.io/source=, konveyor.io/target=), duplicate ruleIDs
- **FR-007**: System MUST generate test data following ARG's pipeline: build prompt from templates, call LLM, extract code blocks, validate language, inject missing imports, create config files, write test project
- **FR-008**: System MUST generate `.test.yaml` files compatible with `kantra test`
- **FR-009**: System MUST score confidence using an adversarial rubric with 5 criteria (pattern correctness, message quality, category appropriateness, effort accuracy, false positive risk), producing accept/review/reject verdicts with evidence
- **FR-010**: System MUST output rules in the directory structure matching the konveyor/rulesets repo (rules/, tests/, tests/data/)
- **FR-011**: System MUST support ingestion from URLs (HTML to markdown), files, and raw text
- **FR-012**: System MUST define all LLM prompts as Go templates in a templates/ directory, not hardcoded in source

### Key Entities

- **Rule**: A single Konveyor analyzer rule with ruleID, description, category, effort, labels, message, links, when condition, customVariables, tag
- **Ruleset**: Metadata for a collection of rules — name, description, labels
- **Condition**: A provider-specific condition (java.referenced, builtin.filecontent, etc.) or a combinator (and/or)
- **MigrationPattern**: An extracted old-to-new API mapping with description, category, effort, language, provider, code examples
- **ConfidenceResult**: Per-rule scoring with criterion scores (1-5), overall score, verdict (accept/review/reject), evidence citations
- **TestsFile**: A kantra test definition with rulesPath, providers, and test cases

## Out of Scope (Phase 1) — Planned Follow-ups

- **OpenRewrite Recipe Ingestion (Phase 1.5)**: ARG supports ingesting OpenRewrite recipes (`ChangePackage`, `ChangeType`, `ChangeDependency`, composites) and converting transformation logic into Konveyor detection rules. This is a separate ingestion source that feeds into the same `MigrationPattern` intermediate type and rule generation pipeline. Phase 1 focuses on migration guide ingestion; OpenRewrite recipe support will be added as a follow-up using the same `ingest` tool (or a dedicated `ingest_openrewrite` tool) with a specialized LLM prompt mode for transformation-to-detection conversion.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A rule author can go from a migration guide URL to a validated set of rules in under 10 minutes of interactive use
- **SC-002**: Generated rules pass structural validation (validate_rules) with zero errors for at least 95% of rules
- **SC-003**: Generated test data produces passing `kantra test` results for at least 70% of rules on the first attempt
- **SC-004**: Confidence scoring correctly identifies intentionally bad rules (wrong patterns, vague messages) as "review" or "reject" at least 80% of the time
- **SC-005**: The output directory structure is directly submittable as a PR to konveyor/rulesets without manual restructuring
- **SC-006**: At least 3 migration paths (e.g., Spring Boot 3→4, PatternFly v5→v6, one Go migration) are successfully demonstrated end-to-end
- **SC-007**: The MCP server starts and responds to tool calls within 2 seconds (excluding LLM inference time)
- **SC-008**: The CLI pipeline can process a migration guide and produce output without any interactive prompts
