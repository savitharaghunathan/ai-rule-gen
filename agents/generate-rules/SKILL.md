---
name: generate-rules
description: Generate Konveyor analyzer migration rules from a migration guide (URL, file path, or pasted text)
---

# Generate Konveyor Migration Rules

Generate Konveyor analyzer migration rules from a migration guide.

**Input:** $ARGUMENTS (URL, file path, or pasted text of a migration guide)

If no argument is provided, ask the user for the migration guide source.

## Progress Updates

Print a short status line at the start and end of every major step so the user can follow along. Use this format:

```
--- Step N: <title> ---
```

At key milestones, print a one-line summary of what was produced (e.g., pattern count, rule count, test group count, pass/fail counts). The user should never wait more than a few seconds without visible output.

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

Print: `Guide ingested: output/guide.md (<N> lines)`

## Step 1: Rule Writer

Delegate to the **rule-writer** skill (`agents/rule-writer/SKILL.md`).

Provide:
- The migration guide content
- Source/target technology if the user specified them (otherwise rule-writer auto-detects)
- Output paths: `output/patterns.json` for patterns, `output/rules` for rule YAML

The rule-writer extracts patterns, writes `patterns.json`, constructs rule YAML, and validates.

Print: `Extracted <N> patterns → <M> rule files in output/rules/`

## Step 2: Test & Validate (optional)

After rule generation, ask the user:

> **N rules generated in `output/rules/`. Do you want to generate tests and run kantra validation?**
> 1. Yes — generate tests and validate with kantra
> 2. Skip — finalize with rules only (no test data, no kantra)

If the user chooses **Skip**, jump directly to **Step 5: Finalize** (stamp all rules as `untested`, skip the report or generate it with `passed: 0, failed: 0`).

If the user chooses **Yes**, continue with Steps 2a–4.

### Step 2a: Test Generator

Delegate to the **test-generator** skill (`agents/test-generator/SKILL.md`).

Provide:
- Path to the rules directory: `output/rules`
- Output directory for tests: `output/tests`

The test-generator scaffolds test structure, generates test source code, resolves dependencies, and sanitizes XML.

Print: `Test data generated: <N> groups across <M> test files`

### Step 2b: Rule Validator

Delegate to the **rule-validator** skill (`agents/rule-validator/SKILL.md`).

Provide:
- Path to the tests directory: `output/tests`
- Path to the rules directory: `output/rules`

The rule-validator runs kantra, parses results, and returns pass/fail status with fix hints for failures.

Print: `Kantra results: <P>/<T> passed`

## Step 3: Fix Loop (max 3 iterations)

If the rule-validator reports failures and iterations remain:

1. Delegate back to the **test-generator** with:
   - The failing rule IDs, their patterns, and fix hints from the rule-validator
   - Instruction to regenerate ONLY the failing test groups (preserve passing groups)

2. Delegate back to the **rule-validator** to re-test

3. Print: `Fix iteration <I>: <P>/<T> passed`

4. Repeat until all pass or 3 iterations exhausted

**Rule integrity:** If a test fails because the test data can't properly trigger the rule, fix the test data — NEVER change the rule's condition type or pattern. If the test still fails after fixing test data, mark the rule as failed.

## Step 4: Finalize

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

If tests were skipped, stamp all rules as `untested` and generate the report with test counts set to 0.

## Step 5: Report to User

Present:
- Number of rules generated
- Test pass rate (passed/total), or "tests skipped" if the user chose to skip
- List of any failing rules with their patterns
- Paths to output files: `output/rules/`, `output/tests/` (if generated), `output/report.yaml`
