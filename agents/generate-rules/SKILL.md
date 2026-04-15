---
name: generate-rules
description: Generate Konveyor analyzer migration rules from a migration guide (URL, file path, or pasted text)
---

# Generate Konveyor Migration Rules

Generate Konveyor analyzer migration rules from a migration guide.

**Input:** $ARGUMENTS (URL, file path, or pasted text of a migration guide)

If no argument is provided, ask the user for the migration guide source.

## Workspace Setup

Create an output directory for this run:

```bash
mkdir -p output
```

All generated files go under `output/`.

## Step 0: Ingest the Migration Guide

Determine the input type and ingest:

- **URL:** `go run ./cmd/ingest --input <url> --output output/guide.md`
- **File (not markdown):** `go run ./cmd/ingest --input <path> --output output/guide.md`
- **Pasted text or already markdown:** Write it directly to `output/guide.md`

Read the ingested guide content before proceeding.

## Step 1: Extract Migration Patterns (Rule Writer)

Read `agents/rule-writer/SKILL.md` and its `references/` directory for the full extraction contract.

From the migration guide content:

1. **Auto-detect** source, target, and language if not provided by the user
2. **Extract every migration pattern** — one per distinct API/annotation/config/dependency change
3. **For each pattern**, populate these fields:
   - `source_pattern` (required) — what to detect
   - `target_pattern` — the replacement (null if removed)
   - `source_fqn` — fully qualified name for `*.referenced` conditions
   - `location_type` — where it appears: TYPE, IMPORT, ANNOTATION, METHOD_CALL, CONSTRUCTOR_CALL, INHERITANCE, IMPLEMENTS_TYPE, FIELD, METHOD, CLASS, RETURN_TYPE, VARIABLE_DECLARATION, ENUM, PACKAGE
   - `alternative_fqns` — other FQNs for the same migration (creates `or` condition)
   - `rationale` (required) — why this migration is needed
   - `complexity` (required) — trivial, low, medium, high, expert
   - `category` (required) — mandatory, optional, potential
   - `concern` — grouping key (e.g., web, security, config)
   - `provider_type` — java, go, nodejs, csharp, builtin
   - `file_pattern` — for builtin.filecontent matches
   - `dependency_name` — for `*.dependency` conditions: `groupId.artifactId` (e.g., `org.springframework.boot.spring-boot-starter-undertow`)
   - `upper_bound` / `lower_bound` — version bounds for dependency conditions (at least one required when `dependency_name` is set)
   - `xpath` — XPath expression for `builtin.xml` conditions
   - `namespaces` — namespace map for XPath (e.g., `{"m": "http://maven.apache.org/POM/4.0.0"}`)
   - `xpath_filepaths` — file paths to restrict XPath matching (e.g., `["pom.xml"]`)
   - `example_before` / `example_after` — code examples
   - `documentation_url` — link to docs
   - `message` — custom migration guidance (if empty, auto-generated)
4. **Deduplicate** — same `source_fqn` or `dependency_name` only once
5. **Generate messages** — clear, actionable, 2-4 sentences with Before/After code examples
6. **Write patterns.json** to `output/patterns.json`

Key rules:
- Use specific FQNs (`javax.ejb.Stateless` not `javax.ejb.*`)
- Set `provider_type` — the CLI uses this to pick the condition type
- Set `location_type` for Java/C# — critical for matching accuracy
- For config file changes, use `provider_type: builtin` with `file_pattern`
- For removed/renamed dependencies, use `dependency_name` + `upper_bound` (produces `java.dependency`)
- For POM structure changes (parent version, plugin config, properties), use `xpath` + `namespaces` + `xpath_filepaths` (produces `builtin.xml`)

## Step 2: Construct and Validate Rules

```bash
go run ./cmd/construct --patterns output/patterns.json --output output/rules
go run ./cmd/validate --rules output/rules
```

If validation fails, fix patterns.json and re-run. Common issues:
- Missing `source_fqn` — the condition has no pattern
- Invalid `location_type` — not one of the 14 valid values
- Invalid regex in `file_pattern`

## Step 3: Scaffold Tests (Test Generator)

Read `agents/test-generator/SKILL.md` and its `references/` directory for test data generation details.

```bash
go run ./cmd/scaffold --rules output/rules --output output/tests
```

Then read `output/tests/manifest.json` to see what source files to generate.

For each group in the manifest:
1. Read the rules referenced by `rule_ids` from `output/rules/`
2. Generate the **build file** (purpose: build) — a valid project file with all dependencies
3. Generate the **source file** (purpose: source) — code that triggers every rule in the group

Source code requirements:
- Must be COMPLETE and COMPILABLE
- For EACH rule, include code that EXACTLY matches the `when` condition pattern
- Add `// Rule: <ruleID>` comment before each pattern
- All imports/dependencies must be valid and resolve

After writing files:
- **Java:** `mvn dependency:resolve -q -B` (in the data dir)
- **Go:** `go mod tidy && go mod vendor`
- **Node.js:** `npm install`

Then sanitize XML:
```bash
go run ./cmd/sanitize --dir output/tests/data
```

## Step 4: Run Kantra Tests (Rule Validator)

Read `agents/rule-validator/SKILL.md` and its `references/` directory for test interpretation details.

Check kantra is installed: `which kantra`

Find all `.test.yaml` files and run kantra:
- **Java/Node.js/C#:** `kantra test <test-files...>`
- **Go (or mixed):** `kantra test --run-local <test-files...>`

Save stdout to `output/kantra-output.txt`.

Parse results:
- Summary: look for `Rules Summary: N/M PASSED`
- Failures: look for `<ruleID>  0/N  PASSED`

## Step 5: Fix Loop (max 3 iterations)

If there are failures:
1. For each failing rule, read the rule YAML to get its pattern
2. Generate a fix hint — a single-line code snippet that matches the pattern
3. Regenerate ONLY the failing test groups with the fix hints
4. Re-resolve dependencies and sanitize XML
5. Re-run kantra tests
6. Repeat until all pass or 3 iterations exhausted

Common fixes:
- 0 incidents: test code doesn't use the API the rule matches — inject the exact pattern
- Compilation error: fix only the failing lines, keep all rule-triggering code
- Go "no views": switch to `--run-local`

## Step 6: Finalize

```bash
# Stamp pass/fail labels on rules
go run ./cmd/stamp --rules output/rules --kantra-output "$(cat output/kantra-output.txt)"

# Generate summary report
go run ./cmd/report \
  --source <source> \
  --target <target> \
  --output output/report.yaml \
  --rules-total <N> \
  --passed <P> \
  --failed <F> \
  --failed-rules <comma-separated-failing-ids>
```

## Step 7: Report to User

Present:
- Number of rules generated
- Test pass rate (passed/total)
- List of any failing rules with their patterns
- Paths to output files: `output/rules/`, `output/tests/`, `output/report.yaml`
