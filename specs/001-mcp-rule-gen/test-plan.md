# Test Plan: Phase 1 MCP Server for AI-Powered Rule Generation

**Date**: 2026-03-19 | **Plan**: [plan.md](plan.md)

## Overview

All tests use `go test`. The `Completer` interface is the primary test seam — mock it to return fixture JSON and assert on deterministic outputs. No API keys or external services needed for unit or integration tests.

## Test Levels

| Level | Build Tag | What It Tests | External Deps | When to Run |
|-------|-----------|---------------|---------------|-------------|
| Unit | (none) | Single package logic | None | Every `go test ./...` |
| Integration | `integration` | Multi-package pipelines end-to-end | None | Every PR in CI |
| E2E | `e2e` | Real LLM responses + kantra | API key, kantra | Nightly / manual |

---

## Unit Tests (per package)

### `internal/rules/`

**File**: `internal/rules/types_test.go`

| Test Case | Description |
|-----------|-------------|
| TestRule_YAMLRoundtrip | Marshal Rule → YAML → unmarshal, verify all fields preserved |
| TestRuleset_YAMLRoundtrip | Marshal Ruleset → YAML → unmarshal |
| TestCondition_JavaReferenced | Verify java.referenced serializes with pattern + location |
| TestCondition_AllProviders | Roundtrip each of the 12 condition types |
| TestCondition_Combinators | and/or conditions with nested conditions |
| TestCondition_ChainingFields | from, as, ignore, not fields on conditions |

**File**: `internal/rules/builder_test.go`

| Test Case | Description |
|-----------|-------------|
| TestBuildJavaReferenced | All 14 location types + annotated variant |
| TestBuildJavaDependency | name, nameregex, upperbound, lowerbound |
| TestBuildGoReferenced | pattern field |
| TestBuildGoDependency | Shared DependencyConditionCap fields |
| TestBuildNodejsReferenced | pattern field |
| TestBuildCsharpReferenced | 4 location types (ALL, METHOD, FIELD, CLASS) |
| TestBuildBuiltinFilecontent | pattern, filePattern, filepaths |
| TestBuildBuiltinFile | pattern field |
| TestBuildBuiltinXml | xpath, namespaces, filepaths |
| TestBuildBuiltinJson | xpath, filepaths |
| TestBuildBuiltinHasTags | string array |
| TestBuildBuiltinXmlPublicID | regex, namespaces, filepaths |
| TestBuildAnd | Combines multiple conditions |
| TestBuildOr | Combines multiple conditions |

**File**: `internal/rules/serializer_test.go`

| Test Case | Description |
|-----------|-------------|
| TestReadRuleFile | Read single YAML file → []Rule |
| TestReadRuleDirectory | Read directory of YAML files → []Rule (all files) |
| TestWriteRules | Write []Rule → YAML file, verify parseable |
| TestWriteRuleset | Write Ruleset → ruleset.yaml |
| TestWriteRules_GroupByConcern | Write rules grouped by concern → multiple files |
| TestReadRuleFile_InvalidYAML | Returns clear error on malformed YAML |

**File**: `internal/rules/validator_test.go`

| Test Case | Description |
|-----------|-------------|
| TestValidate_ValidRule | Complete rule passes with no errors |
| TestValidate_MissingRuleID | Error: ruleID required |
| TestValidate_MissingWhen | Error: when condition required |
| TestValidate_MissingMessageAndTag | Error: message or tag required |
| TestValidate_HasMessageOnly | Valid (message without tag) |
| TestValidate_HasTagOnly | Valid (tag without message) |
| TestValidate_InvalidCategory | Error on category not in {mandatory, optional, potential} |
| TestValidate_EffortOutOfRange | Warning on effort outside 1-10 (per data-model.md) |
| TestValidate_InvalidRegex | Error on syntactically invalid regex pattern |
| TestValidate_BadLabelFormat | Error on labels not matching konveyor.io/source= format |
| TestValidate_DuplicateRuleIDs | Error when two rules share a ruleID |
| TestValidate_RuleIDSpecialChars | Error on newlines or semicolons in ruleID |
| TestValidate_MultipleErrors | Returns all errors, not just first |

### `internal/workspace/`

**File**: `internal/workspace/workspace_test.go`

| Test Case | Description |
|-----------|-------------|
| TestNewWorkspace | Creates `output/<source>-to-<target>/` with subdirs |
| TestWorkspace_RulesDir | Returns correct rules/ path |
| TestWorkspace_TestsDir | Returns correct tests/ path |
| TestWorkspace_ConfidenceDir | Returns correct confidence/ path |
| TestWorkspace_CustomOutputDir | Respects custom output_dir |

Uses `t.TempDir()` for all filesystem operations.

### `internal/llm/`

**File**: `internal/llm/completer_test.go`

| Test Case | Description |
|-----------|-------------|
| TestMockCompleter | Verify mock returns configured responses in order |
| TestMockCompleter_RecordsCalls | Verify prompts are recorded for assertion |
| TestMockCompleter_ExhaustedResponses | Error when all responses consumed |
| TestLLMCompleter_NoProvider | Error when no provider configured |
| TestSamplingCompleter_NotSupported | Returns clear error when MCP client does not support sampling (spec edge case) |

### `internal/server/`

**File**: `internal/server/server_test.go`

| Test Case | Description |
|-----------|-------------|
| TestNewServer | Server creates with all 5 tools registered |
| TestServer_ToolNames | Registered tools match: generate_rules, validate_rules, generate_test_data, run_tests, score_confidence |
| TestServer_SSETransport | Server starts on configured host:port, responds to SSE connections |
| TestServer_StartupTime | Server starts within 2 seconds (SC-007) |

### `internal/tools/`

**File**: `internal/tools/validate_test.go`

| Test Case | Description |
|-----------|-------------|
| TestValidateTool_ValidInput | Tool handler parses input JSON, calls validator, returns valid=true |
| TestValidateTool_InvalidInput | Tool handler returns errors from validator |
| TestValidateTool_MissingPath | Tool handler returns error for missing rules_path |

**File**: `internal/tools/generate_test.go`

| Test Case | Description |
|-----------|-------------|
| TestGenerateTool_ParsesInput | Tool handler parses all input fields (guide_url, code_snippets, changelog, text, source, target, language) |
| TestGenerateTool_RequiresSourceTarget | Error when source or target missing |
| TestGenerateTool_OutputJSON | Tool handler returns expected JSON structure (output_path, files_written, rule_count) |

**File**: `internal/tools/test_generate_test.go`

| Test Case | Description |
|-----------|-------------|
| TestTestGenerateTool_ParsesInput | Tool handler parses rules_path + language |
| TestTestGenerateTool_OutputJSON | Returns test_yaml_path, files_written, post_processing |

**File**: `internal/tools/test_run_test.go`

| Test Case | Description |
|-----------|-------------|
| TestRunTestsTool_DefaultMaxIterations | Defaults to max_iterations=3 when not specified |
| TestRunTestsTool_OutputJSON | Returns passed, failed, total, iterations_run, results |

**File**: `internal/tools/confidence_test.go`

| Test Case | Description |
|-----------|-------------|
| TestConfidenceTool_ParsesInput | Tool handler parses rules_path |
| TestConfidenceTool_OutputJSON | Returns scores_file, results, summary |

### `internal/ingestion/`

**File**: `internal/ingestion/html_test.go`

| Test Case | Description |
|-----------|-------------|
| TestHTMLToMarkdown | Convert sample HTML → markdown, verify headings/links/code preserved |
| TestHTMLToMarkdown_EmptyBody | Returns empty string, no error |
| TestHTMLToMarkdown_CodeBlocks | Preserves fenced code blocks |

**File**: `internal/ingestion/chunker_test.go`

| Test Case | Description |
|-----------|-------------|
| TestChunk_SmallContent | Content under limit returns single chunk |
| TestChunk_LargeContent | Splits at section boundaries (## headings) |
| TestChunk_PreservesSections | No section split mid-paragraph |
| TestChunk_Overlap | Adjacent chunks share context at boundaries |

**File**: `internal/ingestion/ingest_test.go`

| Test Case | Description |
|-----------|-------------|
| TestIngest_URL | Fetches from `httptest.NewServer`, converts HTML→markdown |
| TestIngest_URL_404 | Returns error with clear message |
| TestIngest_File | Reads file from disk |
| TestIngest_File_NotFound | Returns error |
| TestIngest_RawText | Passes through text unchanged |
| TestIngest_EmptyContent | Returns error |

### `internal/extraction/`

**File**: `internal/extraction/patterns_test.go`

| Test Case | Description |
|-----------|-------------|
| TestMigrationPattern_Fields | All fields serialize/deserialize correctly |

**File**: `internal/extraction/extractor_test.go`

| Test Case | Description |
|-----------|-------------|
| TestExtract_ParsesJSON | Mock Completer returns JSON → parses into []MigrationPattern |
| TestExtract_Deduplication | Duplicate patterns (same source_fqn) are merged |
| TestExtract_ChunkedContent | MockCompleter configured with N responses (one per chunk). Extractor calls Complete once per chunk, parses each response, merges all patterns. Verify patterns from all chunks are present and deduplicated. |
| TestExtract_NoPatterns | Returns error when LLM finds nothing actionable |
| TestExtract_MalformedJSON | Returns error on unparseable LLM response |

### `internal/generation/`

**File**: `internal/generation/ruleid_test.go`

| Test Case | Description |
|-----------|-------------|
| TestRuleID_Sequential | IDs increment by 10: 00010, 00020, 00030 |
| TestRuleID_Prefix | Prefix derived from source-target |
| TestRuleID_Reset | New generator starts at 00010 |

**File**: `internal/generation/generator_test.go`

| Test Case | Description |
|-----------|-------------|
| TestGenerate_JavaReferenced | MigrationPattern with provider_type=java → java.referenced condition |
| TestGenerate_BuiltinFilecontent | provider_type=builtin + file_pattern → builtin.filecontent condition |
| TestGenerate_GoDependency | provider_type=go + dependency → go.dependency condition |
| TestGenerate_NodejsReferenced | provider_type=nodejs → nodejs.referenced condition |
| TestGenerate_CsharpReferenced | provider_type=csharp → csharp.referenced condition |
| TestGenerate_ComboCondition | alternative_fqns → or combinator |
| TestGenerate_ComplexityToEffort | Maps MigrationPattern.complexity to Rule.effort. Mapping TBD during implementation — test will define the canonical mapping (e.g., trivial→1, low→3, medium→5, high→7, expert→9). Must produce values within 1-10 range. |
| TestGenerate_GroupByConcern | Rules grouped by concern field into separate files |
| TestGenerate_RulesetMetadata | Generates ruleset.yaml with correct name, labels |
| TestGenerate_Labels | source/target → konveyor.io/source= and konveyor.io/target= labels |
| TestGenerate_Message | Mock Completer provides message → included in rule |

### `internal/testing/`

**File**: `internal/testing/langconfig_test.go`

| Test Case | Description |
|-----------|-------------|
| TestLangConfig_Java | Correct pom.xml path, source path, package structure |
| TestLangConfig_Go | Correct go.mod, main.go paths |
| TestLangConfig_TypeScript | Correct package.json, src/ paths |
| TestLangConfig_CSharp | Correct .csproj, Program.cs paths |

**File**: `internal/testing/scaffold_test.go`

| Test Case | Description |
|-----------|-------------|
| TestScaffold_TestYAML | Generates .test.yaml with rulesPath, providers, test cases |
| TestScaffold_DataDirectory | Creates data dir with correct structure per language |
| TestScaffold_MultipleRules | One test case per rule, each with atLeast:1 |

**File**: `internal/testing/testgen_test.go`

| Test Case | Description |
|-----------|-------------|
| TestExtractCodeBlocks | Parses fenced blocks from LLM response by type (xml→build, java→source) |
| TestInjectImports_Java | Adds missing import statements for IMPORT location rules |
| TestValidateLanguage | Detects Java/TypeScript mismatch |
| TestCreateConfigFiles | Creates placeholder config files for builtin.filecontent rules |

**File**: `internal/testing/runner_test.go`

| Test Case | Description |
|-----------|-------------|
| TestParseKantraOutput_AllPass | Parse output with all rules passing |
| TestParseKantraOutput_Failures | Parse output with failures, extract ruleID + reason |
| TestRunner_KantraNotInstalled | Returns clear error when kantra not on PATH |

**File**: `internal/testing/fixer_test.go`

| Test Case | Description |
|-----------|-------------|
| TestAnalyzeFailure | Parse kantra debug output → identify failing pattern |
| TestGenerateHints | Mock Completer returns improved code hints for failing pattern |

### `internal/confidence/`

**File**: `internal/confidence/rubric_test.go`

| Test Case | Description |
|-----------|-------------|
| TestVerdict_Accept | Overall ≥ 4.0 → accept |
| TestVerdict_Review | Overall ≥ 2.5 and < 4.0 → review |
| TestVerdict_Reject | Overall < 2.5 → reject |
| TestOverallScore | Average of 5 criterion scores |

**File**: `internal/confidence/scorer_test.go`

| Test Case | Description |
|-----------|-------------|
| TestScore_ParsesLLMResponse | Mock Completer returns structured scores → parsed correctly |
| TestScore_FreshContext | Prompt contains only rule YAML + rubric, no generation context |
| TestScore_Evidence | Evidence citations extracted from LLM response |
| TestScore_WritesResults | Results written to confidence/scores.yaml |

### Template Rendering

**File**: `internal/extraction/extractor_test.go` (template tests alongside the package that uses them)

| Test Case | Description |
|-----------|-------------|
| TestExtractionTemplate_Renders | `extract_patterns.tmpl` renders with content, source, target, language — output is non-empty, contains expected placeholders |
| TestExtractionTemplate_MissingFields | Template with empty source/target still renders without error |

**File**: `internal/generation/generator_test.go`

| Test Case | Description |
|-----------|-------------|
| TestMessageTemplate_Renders | `generate_message.tmpl` renders with MigrationPattern fields — output contains Before/After sections |
| TestRulesTemplate_Renders | `generate_rules.tmpl` renders with pattern context |

**File**: `internal/testing/testgen_test.go`

| Test Case | Description |
|-----------|-------------|
| TestTestingTemplate_Java | `main.tmpl` + `java.tmpl` renders with rules and langconfig — output contains pom.xml instructions |
| TestTestingTemplate_Go | `main.tmpl` + `go.tmpl` renders — output contains go.mod instructions |
| TestTestingTemplate_CSharp | `main.tmpl` + `csharp.tmpl` renders — output contains .csproj instructions |
| TestTestingTemplate_TypeScript | `main.tmpl` + `typescript.tmpl` renders — output contains package.json instructions |

**File**: `internal/confidence/scorer_test.go`

| Test Case | Description |
|-----------|-------------|
| TestJudgeTemplate_Renders | `judge.tmpl` renders with rule YAML + rubric — output contains adversarial framing and all 5 criteria |
| TestJudgeTemplate_NoGenerationContext | Template output does NOT contain migration guide content or extraction prompts |

---

## Integration Tests

**Directory**: `internal/integration/` (build tag: `integration`)

| File | Test Case | Description |
|------|-----------|-------------|
| `generate_test.go` | TestGenerateRules_E2E | Sample migration text → mock Completer (configured with extraction response + message responses) → `generate_rules` pipeline → verify: valid YAML output, correct directory structure, rules have conditions + messages + labels |
| `generate_test.go` | TestGenerateRules_MultipleInputTypes | Test with guide_url (via httptest), code_snippets, changelog, text — all produce valid rules |
| `generate_test.go` | TestGenerateRules_LargeContent | Content exceeding chunk limit → mock Completer with N responses → all patterns captured and deduplicated |
| `validate_test.go` | TestValidateRules_GeneratedOutput | Run validator on output from generate pipeline → zero errors |
| `test_pipeline_test.go` | TestTestDataGeneration_E2E | Generated rules → mock Completer → `generate_test_data` → verify .test.yaml structure + data directory with build file + source file |
| `test_pipeline_test.go` | TestTestFixLoop | Mock Completer configured with: (1) initial test code that will "fail", (2) improved hints response, (3) fixed test code. Mock kantra runner returns failure on first run, pass on second. Verify fix_history records the iteration. |
| `confidence_test.go` | TestConfidenceScoring_E2E | Generated rules → mock Completer → `score_confidence` → verify per-rule scores, verdicts, evidence, scores.yaml written |
| `cli_test.go` | TestCLI_Generate | CLI `generate` command with mock LLM → verify output matches rulesets repo layout (rules/, tests/, tests/data/, confidence/) |
| `cli_test.go` | TestCLI_NoAPIKey | CLI without API key → clear error message |

---

## E2E Tests

**Directory**: `test/e2e/` (build tag: `e2e`)

Require: `RULEGEN_LLM_PROVIDER` + API key env vars. Optionally kantra on PATH.

| File | Test Case | Description |
|------|-----------|-------------|
| `generate_e2e_test.go` | TestGenerate_RealGuide_Java | Real Spring Boot migration guide URL → generate_rules → validate output rules are structurally valid |
| `generate_e2e_test.go` | TestGenerate_RealGuide_Go | Real Go migration guide → rules with go.referenced conditions |
| `generate_e2e_test.go` | TestGenerate_RealGuide_Builtin | Real guide with config file changes → rules with builtin.filecontent |
| `test_e2e_test.go` | TestTestData_Java | Generated Java rules → generate_test_data → kantra test → ≥70% pass rate |
| `confidence_e2e_test.go` | TestConfidence_GoodRules | Score known-good rules → expect accept verdicts |
| `confidence_e2e_test.go` | TestConfidence_BadRules | Score intentionally bad rules → expect review/reject verdicts |
| `pipeline_e2e_test.go` | TestFullPipeline | URL → generate → validate → test → score → verify complete output directory |

---

## Test Fixtures

```
testdata/
├── rules/
│   ├── valid/
│   │   ├── java-referenced.yaml       # Rules using java.referenced with various locations
│   │   ├── java-dependency.yaml        # Rules using java.dependency
│   │   ├── go-rules.yaml              # Rules using go.referenced + go.dependency
│   │   ├── nodejs-rules.yaml          # Rules using nodejs.referenced
│   │   ├── builtin-rules.yaml         # Rules using builtin.filecontent, builtin.file, builtin.xml
│   │   ├── csharp-rules.yaml          # Rules using csharp.referenced
│   │   ├── combo-rules.yaml           # Rules using and/or combinators
│   │   └── complete-ruleset/          # Full ruleset directory (ruleset.yaml + rule files)
│   │       ├── ruleset.yaml
│   │       └── web.yaml
│   └── invalid/
│       ├── missing-ruleid.yaml        # Rule without ruleID
│       ├── missing-when.yaml          # Rule without when condition
│       ├── missing-message-tag.yaml   # Rule without message or tag
│       ├── bad-regex.yaml             # Rule with invalid regex in pattern
│       ├── bad-category.yaml          # Rule with category "critical" (invalid)
│       ├── bad-labels.yaml            # Rule with malformed labels
│       └── duplicate-ids.yaml         # Two rules with same ruleID
├── ingestion/
│   ├── sample-guide.html             # HTML migration guide for ingestion tests
│   ├── sample-guide-expected.md       # Expected markdown output
│   └── large-guide.md                # Long content for chunking tests
├── extraction/
│   ├── mock-patterns-java.json        # Mock LLM response: Java migration patterns
│   ├── mock-patterns-go.json          # Mock LLM response: Go migration patterns
│   ├── mock-patterns-nodejs.json      # Mock LLM response: Node.js migration patterns
│   └── mock-patterns-builtin.json     # Mock LLM response: builtin patterns
├── generation/
│   └── expected-rules-java.yaml       # Expected rule output from known Java patterns
├── testing/
│   ├── mock-kantra-pass.txt           # Mock kantra output: all pass
│   ├── mock-kantra-fail.txt           # Mock kantra output: some failures
│   └── mock-llm-testcode-java.txt     # Mock LLM response with fenced Java code blocks
└── confidence/
    ├── mock-scores-good.json          # Mock LLM scoring response for good rules
    └── mock-scores-bad.json           # Mock LLM scoring response for bad rules
```

---

## Mock Completer

The primary test seam. Used across all packages that call LLM:

```go
// MockCompleter returns preconfigured responses for testing.
// Supports multi-call scenarios (e.g., chunked extraction, test-fix loop)
// by returning responses in order.
type MockCompleter struct {
    Responses []string  // Return responses in order, one per Complete() call
    Calls     []string  // Record prompts received (for assertions)
    index     int
}

func (m *MockCompleter) Complete(ctx context.Context, prompt string) (string, error) {
    m.Calls = append(m.Calls, prompt)
    if m.index >= len(m.Responses) {
        return "", fmt.Errorf("MockCompleter: exhausted all %d configured responses", len(m.Responses))
    }
    resp := m.Responses[m.index]
    m.index++
    return resp, nil
}
```

**Multi-call usage example** (chunked extraction with 3 chunks):
```go
mock := &MockCompleter{
    Responses: []string{
        loadFixture("extraction/mock-patterns-java.json"),   // chunk 1 response
        loadFixture("extraction/mock-patterns-go.json"),     // chunk 2 response
        loadFixture("extraction/mock-patterns-builtin.json"),// chunk 3 response
    },
}
// Extractor calls mock.Complete() 3 times, once per chunk
patterns, err := extractor.Extract(ctx, chunks, mock)
assert.Len(t, mock.Calls, 3) // verify 3 calls made
```

---

## CI Integration

```yaml
# GitHub Actions (or equivalent)
test:
  steps:
    - go test ./internal/...                              # Unit tests
    - go test -tags=integration ./internal/integration/... # Integration tests
    - golangci-lint run ./...                              # Lint

test-e2e:  # Nightly or manual trigger
  env:
    RULEGEN_LLM_PROVIDER: anthropic
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  steps:
    - go test -tags=e2e -timeout=10m ./test/e2e/...
```

---

## Coverage Goals

| Package | Target | Rationale |
|---------|--------|-----------|
| `rules/` | ≥90% | Core types and validation — must be solid |
| `llm/` | ≥70% | API clients only tested in E2E; interface + error paths unit-testable |
| `server/` | ≥70% | SSE transport harder to unit test; integration tests cover E2E |
| `tools/` | ≥75% | Handlers are thin wrappers; logic tested via packages they call |
| `ingestion/` | ≥80% | URL fetching has external deps, mock where possible |
| `extraction/` | ≥80% | LLM response parsing — mock Completer |
| `generation/` | ≥90% | Deterministic mapping — fully testable |
| `testing/` | ≥75% | kantra runner harder to test without kantra |
| `confidence/` | ≥80% | Score parsing — mock Completer |
| `workspace/` | ≥90% | Pure filesystem — use t.TempDir() |
| **Overall** | **≥80%** | |
