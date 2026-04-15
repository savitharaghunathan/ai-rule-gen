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

## Step 1: Rule Writer

Delegate to the **rule-writer** skill (`agents/rule-writer/SKILL.md`).

Provide:
- The migration guide content
- Source/target technology if the user specified them (otherwise rule-writer auto-detects)
- Output paths: `output/patterns.json` for patterns, `output/rules` for rule YAML

The rule-writer extracts patterns, writes `patterns.json`, constructs rule YAML, and validates.

## Step 2: Test Generator

Delegate to the **test-generator** skill (`agents/test-generator/SKILL.md`).

Provide:
- Path to the rules directory: `output/rules`
- Output directory for tests: `output/tests`

The test-generator scaffolds test structure, generates test source code, resolves dependencies, and sanitizes XML.

## Step 3: Rule Validator

Delegate to the **rule-validator** skill (`agents/rule-validator/SKILL.md`).

Provide:
- Path to the tests directory: `output/tests`
- Path to the rules directory: `output/rules`

The rule-validator runs kantra, parses results, and returns pass/fail status with fix hints for failures.

## Step 4: Fix Loop (max 3 iterations)

If the rule-validator reports failures and iterations remain:

1. Delegate back to the **test-generator** with:
   - The failing rule IDs, their patterns, and fix hints from the rule-validator
   - Instruction to regenerate ONLY the failing test groups (preserve passing groups)

2. Delegate back to the **rule-validator** to re-test

3. Repeat until all pass or 3 iterations exhausted

## Step 5: Finalize

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

## Step 6: Report to User

Present:
- Number of rules generated
- Test pass rate (passed/total)
- List of any failing rules with their patterns
- Paths to output files: `output/rules/`, `output/tests/`, `output/report.yaml`
