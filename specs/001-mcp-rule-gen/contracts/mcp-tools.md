# MCP Tool Contracts

**Date**: 2026-03-19 | **Plan**: [../plan.md](../plan.md)

5 user-facing tools exposed over SSE transport. Input/output as JSON via MCP protocol. All tools save output to disk automatically under `output/<source>-to-<target>/`.

---

## 1. generate_rules

Generate Konveyor analyzer rules from any input. Internally ingests content, extracts migration patterns via LLM, deterministically constructs valid YAML rules + ruleset metadata, and saves to disk.

**Input** (provide one or more):
```json
{
  "guide_url": "https://spring.io/blog/migration-guide",
  "code_snippets": "// Before\n@RequestMapping(...)\n// After\n@GetMapping(...)",
  "changelog": "## Breaking Changes\n- Removed @RequestMapping...",
  "text": "Detect usage of javax.ejb.Stateless annotation",
  "source": "spring-boot-3",
  "target": "spring-boot-4",
  "language": "java",
  "output_dir": "./output"
}
```

**Output**:
```json
{
  "output_path": "./output/spring-boot-3-to-spring-boot-4",
  "rules_dir": "./output/spring-boot-3-to-spring-boot-4/rules",
  "files_written": [
    "rules/ruleset.yaml",
    "rules/web.yaml",
    "rules/security.yaml"
  ],
  "rule_count": 12,
  "concerns": ["web", "security", "data"],
  "input_type": "guide_url",
  "patterns_extracted": 15
}
```

**Internal pipeline**: ingest → extract_migration_patterns (LLM) → construct_rule (deterministic) → construct_ruleset → save to disk.

**Errors**: 404/unreachable URL, empty content, no patterns found, LLM timeout.

---

## 2. validate_rules

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

## 3. generate_test_data

Generate compilable test source code and test scaffolding using ARG-style pipeline. Saves to `output/<source>-to-<target>/tests/`.

**Input**:
```json
{
  "rules_path": "./output/spring-boot-3-to-spring-boot-4/rules/web.yaml",
  "language": "java"
}
```

**Output**:
```json
{
  "test_yaml_path": "./output/spring-boot-3-to-spring-boot-4/tests/web.test.yaml",
  "files_written": [
    "tests/web.test.yaml",
    "tests/data/web/pom.xml",
    "tests/data/web/src/main/java/com/example/App.java"
  ],
  "language_detected": "java",
  "post_processing": {
    "imports_injected": 3,
    "config_files_created": 1
  }
}
```

**Internal pipeline**: scaffold_test → build prompt (Go templates) → LLM generates code → extract code blocks → validate language → inject imports → create config files → write to disk.

---

## 4. run_tests

Execute `kantra test` with autonomous test-fix loop. Fixes test data, not rules.

**Input**:
```json
{
  "test_file": "./output/spring-boot-3-to-spring-boot-4/tests/web.test.yaml",
  "max_iterations": 3
}
```

**Output**:
```json
{
  "passed": 11,
  "failed": 1,
  "total": 12,
  "iterations_run": 2,
  "results": [
    {
      "ruleID": "spring-boot-4-migration-00010",
      "status": "pass",
      "incidents": 1
    },
    {
      "ruleID": "spring-boot-4-migration-00020",
      "status": "fail",
      "reason": "expected at least 1 incident, got 0",
      "debug_path": "./tests/data/web"
    }
  ],
  "fix_history": [
    {
      "iteration": 1,
      "passed": 10,
      "failed": 2,
      "fixed": ["spring-boot-4-migration-00030"]
    },
    {
      "iteration": 2,
      "passed": 11,
      "failed": 1,
      "fixed": ["spring-boot-4-migration-00040"]
    }
  ]
}
```

**Test-fix loop** (when `max_iterations > 0`):
1. Run `kantra test`
2. If all pass → done
3. Analyze kantra debug output for each failure
4. LLM generates improved code hints for failing patterns
5. Regenerate test data with patched hints
6. Re-run, repeat until passing or max iterations

**Errors**: `kantra` not installed, test file not found, rules file not found.

---

## 5. score_confidence

LLM-as-judge scoring with adversarial rubric in fresh context. Saves to `output/<source>-to-<target>/confidence/`.

**Input**:
```json
{
  "rules_path": "./output/spring-boot-3-to-spring-boot-4/rules/web.yaml"
}
```

**Output**:
```json
{
  "scores_file": "./output/spring-boot-3-to-spring-boot-4/confidence/scores.yaml",
  "results": [
    {
      "ruleID": "spring-boot-4-migration-00010",
      "scores": {
        "pattern_correctness": 5,
        "message_quality": 4,
        "category_appropriateness": 5,
        "effort_accuracy": 4,
        "false_positive_risk": 5
      },
      "overall": 4.6,
      "verdict": "accept",
      "evidence": [
        "message_quality: -1 because Before example missing import context"
      ]
    }
  ],
  "summary": {
    "accept": 10,
    "review": 1,
    "reject": 1
  }
}
```

**LLM**: Fresh context — only rule YAML + adversarial rubric. Prompt from `templates/confidence/judge.tmpl`.

---

## Output Directory Structure

All tools write to a shared workspace under `output/<source>-to-<target>/`:

```
output/spring-boot-3-to-spring-boot-4/
├── rules/                              # generate_rules output
│   ├── ruleset.yaml
│   ├── web.yaml
│   └── security.yaml
├── tests/                              # generate_test_data output
│   ├── web.test.yaml
│   ├── security.test.yaml
│   └── data/
│       ├── web/
│       │   ├── pom.xml
│       │   └── src/main/java/com/example/App.java
│       └── security/
│           ├── pom.xml
│           └── src/main/java/com/example/App.java
└── confidence/                         # score_confidence output
    └── scores.yaml
```

---

## CLI Entry Point

Same internal logic, no MCP protocol.

```bash
# Full pipeline
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

Requires: `RULEGEN_LLM_PROVIDER` + provider-specific API key env var.
