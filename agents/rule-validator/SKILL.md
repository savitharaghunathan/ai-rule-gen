---
name: rule-validator
description: Fix failing test data so rules pass kantra tests, with built-in verify loop
---

# Rule Validator — Fix Agent

You fix test data for failing rules using a lookup-based approach. Read the rule, identify the condition type, apply the known fix from the provider reference.

## Constraints (RIGID — follow exactly)

This is a **lookup-based fix loop**, not an investigation. You follow a fixed 4-step procedure: read rule → look up fix → apply fix → verify. Nothing else.

**FORBIDDEN — do not do any of these:**
- Download or resolve Maven artifacts (`mvn dependency:get`, `mvn dependency:resolve`, `mvn compile`)
- Inspect JAR files (`jar tf`, `unzip`, browsing `.m2/repository`)
- Verify whether FQNs, classes, or packages exist in real libraries
- Investigate whether the rule itself is correct — the rule is always authoritative
- Use `python`, `python3`, `node`, or any scripting language runtime — this is a Go project
- Run any command not listed in the Permissions table
- Add dependencies, files, or code beyond what `references/languages/<language>/fix-strategies.md` prescribes
- Modify rule YAML files — fixes always target test data

If the lookup fix doesn't resolve the failure, mark the rule as `still_failing` and move on. Do not improvise alternative approaches.

## Inputs

- `rules_dir` — Directory containing rule YAML files
- `tests_dir` — Directory containing test data
- `failing_rules` — List of failing rules, each with:
  - `rule_id` — The failing rule ID
  - `test_file` — The `.test.yaml` filename (bare name, e.g. `data-1.test.yaml`)
  - `error` — Error summary from kantra output
- `max_iterations` — Max fix-verify cycles (default: 1, max: 3)

## Returns

- `fixed_rules` — List of rule IDs that were fixed
- `still_failing` — List of rule IDs that still fail after all iterations
- `iterations_used` — Number of fix-verify cycles executed
- `fix_details` — Per-rule diagnosis and fix description:
  - `rule_id`
  - `diagnosis` — What was wrong
  - `fix` — What was changed

## Permissions

| Operation | Pattern | Purpose |
|-----------|---------|---------|
| shell | `go run ./cmd/test *` | Run kantra tests on subset |
| read | `output/rules/**` | Read rule YAML |
| read | `output/tests/**` | Read test files and kantra output |
| read | `agents/rule-validator/references/**` | Read fix strategies |
| read | `agents/rule-validator/references/languages/**` | Read language-specific fix strategies |
| edit | `output/tests/**` | Fix failing test data |
| write | `output/tests/**` | Rewrite test files when edits are insufficient |

## References

Read before starting:
- `references/fix-strategies.md` — Fix loop flow, rule integrity principle
- `references/languages/<language>/fix-strategies.md` — Condition-type fix lookup for the relevant language (java, go, nodejs, csharp, python)

## Workflow

For each iteration (up to `max_iterations`):

### Step 1. Read rule and identify condition type

For each failing rule, read the rule YAML from `rules_dir`. Extract:
- **Condition type**: `java.referenced`, `java.dependency`, `builtin.filecontent`, `builtin.xml`, `go.referenced`, etc.
- **Location** (for `*.referenced`): `ANNOTATION`, `IMPORT`, `METHOD_CALL`, `TYPE`, etc.
- **Pattern**: the regex or FQN the rule matches against

The provider language comes from the condition type prefix (e.g., `java.referenced` → `java`, `go.referenced` → `go`, `builtin.*` → use the language from the rule's labels or the majority provider in the failing set). Read only the matching `references/languages/<language>/fix-strategies.md` — not all five.

### Step 2. Look up the fix

Open `references/languages/<language>/fix-strategies.md` for the relevant provider. Find the section for the condition type from step 1. It lists the known failure mode and the fix.

### Step 3. Apply the fix

Read the test source files in the failing group's data directory. Apply the fix from step 2. Write corrected files.

### Step 4. Verify

Re-run tests on only the fixed groups:

```bash
go run ./cmd/test --rules <rules_dir> --tests <tests_dir> --files <comma-separated failing .test.yaml filenames>
```

Set timeout: 420000 (7 minutes).

Parse the JSON output:
- **All pass** → done, return results
- **Some still fail + iterations remain** → next iteration with remaining failures
- **Some still fail + no iterations remain** → report as `still_failing` and return

## Rule Integrity

See `references/fix-strategies.md` — Rule Integrity Principle. The rule is authoritative; fixes always target test data.
