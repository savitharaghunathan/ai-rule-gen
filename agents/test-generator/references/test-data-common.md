# Test Data Generation — Common Reference

This covers language-agnostic rules for generating test data. Read the language-specific guide at `references/languages/<language>/test-data-guide.md` for project structure, dependency resolution, and version rules.

## Goal

For each rule, generate a COMPLETE, COMPILABLE project where the source code contains code that EXACTLY matches the rule's `when` condition pattern. The analyzer will run against this code and must find at least 1 incident per rule.

## Requirements

1. Create a COMPLETE, COMPILABLE project for the detected language
2. For EACH rule, include code that EXACTLY matches the pattern in the `when` condition
3. Add a comment before each pattern: `// Rule: <ruleID>` (or `# Rule: <ruleID>` for Python)
4. Keep code minimal — one example per rule, just enough to trigger the pattern
5. All imports/dependencies must be valid and resolve

## Source Code Must Use the OLD (Source) API

The test data simulates code that has NOT been migrated yet. Every import, annotation, type reference, and dependency version must use the **source** (pre-migration) API — never the target.

The `pattern` field in the rule YAML tells you the exact FQN to use in the test code. Copy it verbatim.

## Output Format

Generate EXACTLY TWO fenced code blocks per test group:

**FIRST block:** Build file contents (pom.xml, go.mod, package.json, Project.csproj, or requirements.txt)

**SECOND block:** Main source file contents (Application.java, main.go, App.tsx, Program.cs, or main.py)

Do NOT include any other text or code blocks.

## Reading the manifest.json

`go run ./cmd/scaffold` outputs a `manifest.json` that tells you exactly what files to generate:

```json
{
  "language": "java",
  "groups": [
    {
      "name": "web",
      "data_dir": "tests/data/web",
      "test_file": "tests/web.test.yaml",
      "rule_count": 3,
      "providers": ["java"],
      "files": [
        {"path": "tests/data/web/pom.xml", "file_type": "xml", "purpose": "build"},
        {"path": "tests/data/web/src/main/java/com/example/Application.java", "file_type": "java", "purpose": "source"}
      ],
      "rule_ids": ["rule-00010", "rule-00020", "rule-00030"]
    }
  ]
}
```

For each group:
1. Read the rules referenced by `rule_ids` to see what patterns must be matched
2. Generate the build file at the `path` with `purpose: "build"`
3. Generate the source file at the `path` with `purpose: "source"`
4. The source code must trigger ALL rules in that group

## XML Sanitization

After generating all test files, run `go run ./cmd/sanitize --dir <tests-dir>` to clean XML files. LLMs frequently generate comments like `<!-- --add-opens flag -->` which contains `--` inside a comment — this is illegal XML and breaks Maven's POM parser. The sanitizer replaces `--` sequences inside XML comments with spaces.

## Merging Small Test Groups

When many small test groups use the same provider (e.g., multiple groups with 1-3 rules each), consider merging them into fewer groups. Each group runs a separate language server session in the kantra container, and too many sessions cause OOM failures.

The scaffold command groups by concern automatically, but manual merging may be needed when:
- Multiple concerns share the same provider and have few rules each
- Kantra tests fail with container memory errors

To merge: edit the `.test.yaml` file to add the extra rule entries, and add the corresponding test code to the shared source file.

## Fix Iterations

When a rule fails kantra tests, you'll receive:
- The failing rule IDs
- Their patterns (from the rule YAML)
- Failure context from the validator

On fix iterations:
- Regenerate ONLY the failing test groups (preserve passing groups)
- Use the fix guidance to understand what the test code is missing
- The most common failure is: the test code doesn't actually use the API that the rule pattern matches

**Rule integrity:** The rule is authoritative — never change a rule to make a test pass. Fix the test data instead.
