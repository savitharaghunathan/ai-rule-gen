# MCP Tool Contracts

**Date**: 2026-03-19 | **Plan**: [../plan.md](../plan.md)

4 deterministic tools exposed over SSE transport. No server-side LLM needed — the client's LLM (Claude Code, Cursor, Kai) does the thinking: reads migration guides, extracts patterns, and calls these tools to construct valid YAML.

Pipeline capabilities (generate_rules, generate_test_data, run_tests, score_confidence) are **CLI-only** — they call internal packages directly with a server-side LLM configured via `RULEGEN_LLM_PROVIDER`. See [CLI Pipeline](#cli-pipeline) below.

---

## MCP Tools (Deterministic)

These tools are purely deterministic — no LLM on the server. The interactive MCP workflow is:

1. User gives client LLM a migration guide or description
2. Client LLM reads it, identifies migration patterns
3. Client LLM calls `get_help` to learn about condition types and locations
4. Client LLM calls `construct_rule` for each pattern with the right parameters
5. Client LLM calls `construct_ruleset` to create ruleset metadata
6. Client LLM calls `validate_rules` to verify the output

---

### 1. construct_rule

Construct a single Konveyor analyzer rule from explicit parameters. Returns valid YAML string. The tool description is detailed so the client LLM knows exactly what parameters to pass (Scribe-style "parametric collapse").

**Input**:
```json
{
  "ruleID": "spring-boot-4-migration-00010",
  "condition_type": "java.referenced",
  "pattern": "org.springframework.web.bind.annotation.RequestMapping",
  "location": "ANNOTATION",
  "message": "## Before\n\n```java\n@RequestMapping(...)\n```\n\n## After\n\n```java\n@GetMapping(...)\n```\n\n## Additional info\n\n- Use specific mapping annotations instead",
  "category": "mandatory",
  "effort": 3,
  "description": "Replace deprecated @RequestMapping",
  "labels": [
    "konveyor.io/source=spring-boot-3",
    "konveyor.io/target=spring-boot-4"
  ],
  "links": [
    {"title": "Spring Boot 4 Migration Guide", "url": "https://spring.io/blog/migration"}
  ]
}
```

**Supported `condition_type` values**:
- `java.referenced` — requires `pattern`, `location` (ANNOTATION, IMPORT, CLASS, METHOD_CALL, CONSTRUCTOR_CALL, FIELD, METHOD, INHERITANCE, IMPLEMENTS_TYPE, ENUM, RETURN_TYPE, VARIABLE_DECLARATION, TYPE, PACKAGE). Optional: `annotated` (pattern + elements).
- `java.dependency` — requires `name` or `nameRegex`. Optional: `lowerbound`, `upperbound`.
- `go.referenced` — requires `pattern`.
- `go.dependency` — requires `name` or `nameRegex`. Optional: `lowerbound`, `upperbound`.
- `nodejs.referenced` — requires `pattern`.
- `csharp.referenced` — requires `pattern`. Optional: `location` (ALL, METHOD, FIELD, CLASS).
- `builtin.filecontent` — requires `pattern`. Optional: `filePattern`, `filepaths`.
- `builtin.file` — requires `pattern`.
- `builtin.xml` — requires `xpath`. Optional: `namespaces`, `filepaths`.
- `builtin.json` — requires `xpath`. Optional: `filepaths`.
- `builtin.hasTags` — requires `tags` (string array).
- `builtin.xmlPublicID` — requires `regex`. Optional: `namespaces`, `filepaths`.

**Output**:
```json
{
  "yaml": "- ruleID: spring-boot-4-migration-00010\n  description: ...\n  ...",
  "valid": true,
  "errors": []
}
```

**Errors**: Missing required fields for condition type, invalid location, invalid category, invalid regex pattern.

---

### 2. construct_ruleset

Construct a ruleset metadata YAML file.

**Input**:
```json
{
  "name": "spring-boot-4-migration",
  "description": "Rules for migrating Spring Boot 3.x to 4.0",
  "labels": [
    "konveyor.io/source=spring-boot-3",
    "konveyor.io/target=spring-boot-4"
  ]
}
```

**Output**:
```json
{
  "yaml": "name: spring-boot-4-migration\ndescription: ...\nlabels:\n  - ...",
  "valid": true
}
```

---

### 3. validate_rules

Validate rule YAML for structural correctness. Deterministic.

**Input**:
```json
{
  "rules_path": "./output/spring-boot-3-to-spring-boot-4/rules/web.yaml"
}
```

**Output**:
```json
{
  "valid": true,
  "errors": [],
  "warnings": ["Rule spring-boot-00020: effort=8 is unusually high"],
  "rule_count": 12
}
```

**Checks**: Valid YAML, required fields (ruleID, when, message or tag), valid category enum, effort range, regex pattern syntax, label format, duplicate ruleIDs.

---

### 4. get_help

Return documentation about Konveyor rule format, condition types, valid locations, label conventions, and examples. Helps the client LLM understand rule structure.

**Input**:
```json
{
  "topic": "condition_types"
}
```

**Valid topics**: `condition_types`, `locations`, `labels`, `categories`, `rule_format`, `ruleset_format`, `examples`, `all`.

**Output**:
```json
{
  "topic": "condition_types",
  "content": "## Supported Condition Types\n\n### java.referenced\n..."
}
```

---

## CLI Pipeline

Pipeline capabilities are accessed via the `rulegen` CLI, not via MCP. They call internal packages directly with a server-side LLM.

**LLM Configuration** (env vars):
| Variable | Description |
|----------|-------------|
| `RULEGEN_LLM_PROVIDER` | Provider name: `anthropic`, `openai`, `gemini`, `ollama` |
| `ANTHROPIC_API_KEY` | API key for Anthropic (Claude) |
| `OPENAI_API_KEY` | API key for OpenAI (GPT) |
| `GEMINI_API_KEY` | API key for Google Gemini |
| `OLLAMA_HOST` | Ollama server URL (default: `http://localhost:11434`) |
| `OLLAMA_MODEL` | Ollama model name (default: `llama3`) |

### CLI Commands

```bash
# Full pipeline: generate + validate + test + score
rulegen generate \
  --guide-url https://spring.io/blog/migration-guide \
  --source spring-boot-3 \
  --target spring-boot-4 \
  --language java \
  --output ./output/

# Individual operations
rulegen validate --rules ./output/spring-boot-3-to-spring-boot-4/rules/
rulegen test --test-file ./output/.../tests/web.test.yaml --max-iterations 3
rulegen score --rules ./output/spring-boot-3-to-spring-boot-4/rules/
```

### generate (CLI)

Generate Konveyor analyzer rules from any input. Internally ingests content, extracts migration patterns via server-side LLM, deterministically constructs valid YAML rules + ruleset metadata, and saves to disk.

**Internal pipeline**: ingest → extract_migration_patterns (server-side LLM) → construct_rule (deterministic) → construct_ruleset → validate → save to disk.

### test (CLI)

Generate test data and run `kantra test` with autonomous test-fix loop. Fixes test data, not rules.

**Test-fix loop** (requires LLM provider):
1. Run `kantra test`
2. If all pass → done
3. Analyze kantra debug output for each failure
4. Server-side LLM generates improved code hints for failing patterns
5. Regenerate test data with patched hints
6. Re-run, repeat until passing or max iterations

### score (CLI)

LLM-as-judge scoring with adversarial rubric in fresh context.

---

## Output Directory Structure

All pipeline output is written to a shared workspace under `output/<source>-to-<target>/`:

```
output/spring-boot-3-to-spring-boot-4/
├── rules/                              # generate output
│   ├── ruleset.yaml
│   ├── web.yaml
│   └── security.yaml
├── tests/                              # test output
│   ├── web.test.yaml
│   ├── security.test.yaml
│   └── data/
│       ├── web/
│       │   ├── pom.xml
│       │   └── src/main/java/com/example/App.java
│       └── security/
│           ├── pom.xml
│           └── src/main/java/com/example/App.java
└── confidence/                         # score output
    └── scores.yaml
```
