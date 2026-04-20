---
name: test-generator
description: Generate compilable test source code that triggers Konveyor analyzer rule patterns
---

# Test Generator

You generate test application source code that triggers Konveyor analyzer rules. The test code must be compilable and must contain code that EXACTLY matches each rule's `when` condition pattern.

## Inputs

- `rules_dir` — Directory containing rule YAML files
- `tests_dir` — Directory containing scaffolded test structure
- `groups` — (optional) List of groups to generate. If provided, skip scaffold and manifest steps. If omitted, run scaffold first and read manifest.json to get the full group list. Each group has:
  - `name` — Group name
  - `data_dir` — Path to group's data directory
  - `rule_ids` — Rule IDs in this group
  - `files` — Files to generate (path + purpose)

## Returns

- `groups_completed` — Number of groups processed
- `files_written` — Number of files written

## References

Read this before starting:
- `references/test-data-guide.md` — How the analyzer matches each condition type, project structure per language, output format, manifest.json structure

## Workflow

### 1. Scaffold (skip if `groups` provided)

If `groups` was provided in the inputs, skip to step 3 — scaffold and manifest were already handled.

Otherwise, run the CLI to create test structure and manifest:

```bash
go run ./cmd/scaffold --rules <rules-dir> --output <output-dir>
```

This creates:
- `.test.yaml` files (kantra test definitions)
- Data directories for each test group
- `manifest.json` describing what source files to generate

### 2. Read manifest.json

The manifest tells you exactly what files to generate:

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

### 3. Generate source code for each group

For each group in the manifest:

1. Read the rules referenced by `rule_ids` from the rules directory
2. Look at each rule's `when` condition to understand what pattern must be matched
3. Generate the **build file** (purpose: `build`) — a valid project file with all required dependencies
4. Generate the **source file** (purpose: `source`) — code that triggers every rule in the group

**Source code requirements:**
- The project must be COMPLETE and COMPILABLE
- For EACH rule, include code that EXACTLY matches the pattern in the `when` condition
- Add a comment before each pattern: `// Rule: <ruleID>`
- Keep code minimal — one example per rule, just enough to trigger the pattern
- All imports/dependencies must be valid and resolve

**How the analyzer matches each condition type:**
- `java.referenced` with location `ANNOTATION`: USE the annotation on a class, method, or field (e.g., `@MockBean private Object svc;`). **An import alone is NOT enough** — the annotation must appear as `@Name` on an actual element.
- `java.referenced` with location `IMPORT`: use the import statement
- `java.referenced` with location `TYPE`: declare or use the type
- `java.referenced` with location `METHOD_CALL`: call the method on an explicitly typed variable. **Do NOT chain calls** — `Foo.get().bar()` fails in source-only mode because JDTLS can't resolve intermediate return types. Instead: `Foo f = Foo.get(); f.bar();`
- `go.referenced`: import and use the package/symbol (e.g., `import "golang.org/x/crypto/md4"`)
- `nodejs.referenced`: import and use the symbol
- `builtin.filecontent`: include text matching the regex pattern in the appropriate file

### 4. Resolve dependencies (only when needed)

Most rules use source-only analysis and do NOT need dependency resolution. Only resolve when required:

- **Go:** Always run `go mod tidy` then `go mod vendor` (gopls in kantra container can't download modules)
- **Java `java.referenced` (source-only):** Do NOT run `mvn dependency:resolve`. Source-only analysis resolves IMPORT, ANNOTATION, TYPE patterns without downloaded JARs.
- **Java `java.dependency`:** DO run `mvn dependency:resolve -q -B`. These rules use Maven dependency resolution, not JDTLS.
- **Node.js:** `npm install` only if needed for type resolution
- **C#:** `dotnet restore` only if needed for type resolution

### 5. Sanitize XML

Run the sanitizer to fix any illegal XML comments:

```bash
go run ./cmd/sanitize --dir <tests-data-dir>
```

### 6. Return

Return the path to the tests directory to the orchestrator.

## Fix Iterations

On fix iterations, the orchestrator provides:
- Failing rule IDs
- Their patterns (from the rule YAML)
- Failure context and fix guidance from the rule-validator

When fixing:
- Regenerate ONLY the failing test groups — do not touch passing groups
- Use the fix guidance to understand what the test code needs
- The most common failure is: the test code doesn't actually use the API that the rule pattern matches
- If a specific code hint is provided (a single-line snippet), inject that exact line into the source file

### Compilation fix approach

If the test code has compilation errors:
1. Run the language-specific compiler to check
2. Fix ONLY the lines mentioned in the errors
3. Keep ALL rule-triggering code — every import and usage must remain
4. Do NOT change library versions in the build file — fix the code to match the installed version
5. For Go: run `go doc <package>` to get actual function signatures
6. Re-resolve dependencies after fixing
