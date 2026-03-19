# Research: Phase 1 MCP Server for AI-Powered Rule Generation

**Date**: 2026-03-19 | **Plan**: [plan.md](plan.md)

## Decisions

### 1. MCP SDK Selection

**Decision**: `github.com/mark3labs/mcp-go`
**Rationale**: Most widely adopted Go MCP SDK (1695+ importers). Mature SSE transport support. Active maintenance. Used in production by multiple projects.
**Alternatives considered**:
- `github.com/modelcontextprotocol/go-sdk` — official SDK, newer but less battle-tested
- Custom implementation — unnecessary given mature options exist

### 2. Transport Protocol

**Decision**: SSE only (no stdio)
**Rationale**: Works for all target clients (Claude Code, Kai, VS Code, CI/CD). Single deployment model — run locally or containerized. Natural fit for Kubernetes. Bind `localhost` by default for security.
**Alternatives considered**:
- stdio — adds complexity, no client requires it, harder to containerize
- SSE + stdio — double the transport code for no demonstrated need

### 3. LLM Abstraction

**Decision**: `Completer` interface with `SamplingCompleter` (MCP path) and `LLMCompleter` (CLI path)
**Rationale**: MCP sampling eliminates API key requirement for interactive use. Server builds prompt, client's LLM generates, server post-processes. CLI path uses server-side LLM for pipeline/CI. Same interface means internal packages never know which backend is in use.
**Alternatives considered**:
- Server-side LLM only — requires API key even for interactive use, duplicates what the client already has
- Client-side only — doesn't work for CLI/pipeline where there's no MCP client

### 4. Rule Type System

**Decision**: Own YAML-serializable types. Import `analyzer-lsp/output/v1/konveyor` (Category, Link) and `kantra/pkg/testing` (TestsFile, Runner, Result). Do NOT import `engine.Rule`.
**Rationale**: `engine.Rule` contains runtime interfaces (`Conditional` with `Evaluate()`) and heavy provider dependencies. We only need types that serialize to/from YAML. Both Scribe and ARG take this same approach.
**Alternatives considered**:
- Import `engine.Rule` directly — pulls in runtime interfaces, LSP dependencies, tree-sitter parsers

### 5. Condition Schema Approach

**Decision**: Hardcoded condition structs (Approach B) mirroring upstream definitions from analyzer-lsp and external providers. Use `parser.CreateSchema()` for base rule/ruleset OpenAPI schema.
**Rationale**: Matches ARG's approach. Condition types are stable and rarely change. Avoids pulling in provider binaries, GRPC setup, and runtime initialization just to get schemas.
**Alternatives considered**:
- Import and instantiate providers (Approach A) — rejected because:
  1. C# provider is Rust (separate binary, GRPC only)
  2. Java/Go/Node.js providers are external GRPC binaries
  3. Builtin provider requires full `Init()` with logger/config
  4. Transitive dependencies (LSP clients, language servers, parsers)
  5. Manual sync on rare upstream changes is cheaper than permanent coupling

### 6. Confidence Scoring Independence

**Decision**: Context separation (fresh prompt with only rule YAML + rubric), not vendor separation
**Rationale**: The judge never sees the generation prompt, chain-of-thought, or migration guide. Works with both MCP sampling (fresh context in sampling request) and server-side LLM. Requiring a different vendor would force users to configure multiple LLM providers.
**Alternatives considered**:
- Different LLM vendor for scoring — adds configuration burden, no evidence it improves accuracy

### 7. Test Data Generation

**Decision**: ARG-style pipeline (templates + LLM + post-processing)
**Rationale**: Proven pipeline from ARG. Deterministic-only approaches cover ~30-40% of rules. The pipeline: build prompt from Go templates, call LLM via sampling/server-side, extract code blocks, validate language, inject missing imports, create config files, write test project.
**Alternatives considered**:
- Deterministic code generation — insufficient coverage
- Client LLM ad-hoc generation — no quality guarantees, no structured post-processing

### 8. OpenRewrite Support

**Decision**: Deferred to Phase 1.5
**Rationale**: Separate ingestion source feeding into the same `MigrationPattern` pipeline. Phase 1 focuses on migration guide ingestion. Easy to add later since it's just another input path. ARG's implementation (`OpenRewriteRecipeIngester`) provides a clear reference.
**Alternatives considered**:
- Include in Phase 1 — adds scope without being needed for core value demonstration

### 9. kantra Integration

**Decision**: Shell out to `kantra test` CLI initially; evaluate importing `kantra/pkg/testing` programmatically
**Rationale**: Shelling out is simpler and works immediately. Programmatic import via `Runner` interface gives structured results but may have initialization complexity. Start simple, upgrade if needed.
**Alternatives considered**:
- Programmatic only — may have hidden init requirements
- CLI only — loses structured pass/fail results per rule

### 10. Tool Surface Area

**Decision**: 5 user-facing MCP tools, internal functions for pipeline steps
**Rationale**: `generate_rules` absorbs ingest + extract + construct + save (like ARG, which saves to disk immediately after generation). `generate_test_data` absorbs scaffold_test. `run_tests` includes autonomous test-fix loop (like ARG's `--max-iterations`). Users interact with higher-level tools; the LLM decides *what* to build, deterministic code ensures *how*. Matches how users think ("generate rules from this guide" not "ingest, then extract, then generate, then save").
**Alternatives considered**:
- 11 granular tools (original design) — too many steps for the user, intermediate tools like `extract_migration_patterns` aren't useful standalone
- 6 tools with separate `save_rules` — unnecessary; ARG saves immediately, and tools need a shared workspace on disk anyway
- Single monolithic tool — too little control, can't validate/test/score independently

### 11. Test-Fix Loop

**Decision**: Autonomous test-fix loop inside `run_tests`, fixes test data (not rules), configurable `max_iterations` (default 3)
**Rationale**: Matches ARG's proven approach. Rules are the source of truth. If a test fails, the generated test code doesn't trigger the pattern correctly — fix the test code. ARG achieves 70% → 95%+ pass rate in 2-3 iterations. Loop uses LLM to generate improved code hints for failing patterns, then regenerates test data.
**Alternatives considered**:
- Client LLM orchestrates the loop — works in MCP mode but not in CLI mode. Built-in loop works for both.
- Fix rules instead of test data — rules represent user intent, should not be auto-modified

### 12. Provider Condition Coverage

**Decision**: Support 12 condition types + and/or combinators

| Provider | Conditions | Source |
|----------|-----------|--------|
| java | `referenced` (14 locations + annotated), `dependency` | analyzer-lsp external provider |
| go | `referenced`, `dependency` | analyzer-lsp LSP base + shared DependencyConditionCap |
| nodejs | `referenced` | analyzer-lsp generic external provider |
| csharp | `referenced` (4 locations: ALL, METHOD, FIELD, CLASS) | c-sharp-analyzer-provider (Rust, GRPC) |
| builtin | `filecontent`, `file`, `xml`, `json`, `hasTags`, `xmlPublicID` | analyzer-lsp internal builtin provider |

**Note**: `csharp.dependency` does not exist upstream. Chaining fields (`from`, `as`, `ignore`, `not`) supported on all conditions.
