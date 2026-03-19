<!--
Sync Impact Report
- Version change: 0.0.0 → 1.0.0
- Added principles: I. MCP-First, II. Sampling Over Server LLM, III. Ecosystem Alignment, IV. Template-Driven Generation, V. Test-First, VI. Simplicity
- Added sections: Integration Constraints, Development Workflow
- Templates requiring updates: ✅ constitution.md updated
- Follow-up TODOs: None
-->

# AI Rule Generator Constitution

## Core Principles

### I. MCP-First

All capabilities MUST be exposed as MCP tools over SSE transport. The MCP server is the primary interface for interactive clients (Claude Code, Kai, VS Code). A CLI entry point provides direct access to the same internal packages for pipeline/CI use. Both entry points share identical internal logic — no feature may exist in one path without being available in the other.

### II. Sampling Over Server LLM

When an MCP client is present, LLM inference MUST use MCP sampling — the server builds the prompt, the client's LLM generates, the server post-processes. The server MUST NOT require its own LLM API key for interactive use. Server-side LLM is only for the CLI/pipeline path where no MCP client exists. The same `Completer` interface MUST abstract both paths so internal packages never know which backend is in use.

### III. Ecosystem Alignment

The tool MUST produce output compatible with the Konveyor ecosystem:
- Rule YAML MUST be parseable by `analyzer-lsp`'s rule parser
- Test YAML MUST be valid for `kantra test`
- Output directory structure MUST match `konveyor/rulesets` repo layout
- Importable types from `analyzer-lsp/output/v1/konveyor` (Category, Link) and `kantra/pkg/testing` (TestsFile, Runner, Result) MUST be used where applicable
- Runtime types from `analyzer-lsp/engine` (Conditional interface) MUST NOT be imported — use own YAML-serializable types instead

### IV. Template-Driven Generation

All LLM prompts MUST be defined as Go templates (`text/template`) in the `templates/` directory, not hardcoded in Go source. This applies to pattern extraction, rule generation, test data generation, and confidence scoring. Templates MUST be language-specific where applicable (Java, Go, TypeScript). This ensures prompt quality is reviewable, testable, and improvable independently of code changes.

### V. Test-First

Every internal package MUST have unit tests before feature work is considered complete. Integration tests MUST validate end-to-end flows (guide URL to generated rules, rules to test data, test data to kantra test results). Generated rules MUST be validated both structurally (deterministic) and via confidence scoring (LLM-as-judge with adversarial rubric). Test data generation MUST follow the ARG-style pipeline: templates, LLM generation, post-processing (code block extraction, language validation, import injection).

### VI. Simplicity

Start with the minimum viable set of tools. Do not add abstractions, indirections, or configurability that is not needed for Phase 1. Prefer 11 focused tools over 27 granular ones. Prefer direct function calls over framework magic. Prefer `map[string]interface{}` conditions over a sealed type hierarchy when the output is just YAML. Add complexity only when a concrete use case demands it.

## Integration Constraints

- **Language**: Go — aligns with analyzer-lsp and kantra
- **MCP SDK**: `github.com/mark3labs/mcp-go` — SSE transport, widely adopted
- **Transport**: SSE only — no stdio. Bind to `localhost` by default
- **LLM Providers**: Anthropic (default), OpenAI, Google — configured via environment variables for CLI path only
- **Confidence Scoring**: MUST use adversarial framing, rubric-based scoring, evidence requirement. Independence is context separation (fresh prompt with only rule YAML), not vendor separation
- **Supported Condition Types**: java.referenced, java.dependency, go.referenced, go.dependency, nodejs.referenced, builtin.filecontent, builtin.file, builtin.hasTags, and/or combinators

## Development Workflow

- All changes MUST be reviewed against ecosystem compatibility (does the output still parse in analyzer-lsp? does kantra test still pass?)
- Templates MUST be tested with real migration guides before merging
- Generated rules MUST be validated against at least one real codebase before claiming a migration path is "supported"
- PRs MUST include before/after examples of generated YAML for any change to generation logic
- Confidence scoring rubric changes MUST be justified with examples of rules that were incorrectly scored under the old rubric

## Governance

This constitution governs all development on the ai-rule-gen project. Amendments require:
1. Documentation of the proposed change and rationale
2. Verification that the change does not break ecosystem compatibility
3. Update of all dependent templates and artifacts

Complexity MUST be justified. If a simpler approach achieves the same outcome, the simpler approach MUST be chosen. The PLAN.md in the project root is the authoritative source for architectural decisions and tool design.

**Version**: 1.0.0 | **Ratified**: 2026-03-19 | **Last Amended**: 2026-03-19
