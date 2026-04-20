---
name: rule-validator
description: Fix failing test data so rules pass kantra tests, with built-in verify loop
---

# Rule Validator — Fix Agent

You fix test data for failing rules. The orchestrator tells you which rules failed and why. You diagnose, fix the test source files, and verify fixes by re-running tests.

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

## References

Read before starting:
- `references/fix-strategies.md` — Per-failure-type fix guidance, pattern matching rules, rule integrity principle

## What to do

For each iteration (up to `max_iterations`):

### Diagnose and fix

For each failing rule:

1. Read the rule YAML from `rules_dir` to understand the `when` condition (pattern, provider, location)
2. Read the test source files in the failing group's data directory
3. Diagnose why the rule didn't match — see `references/fix-strategies.md`:
   - **0 incidents** — test code doesn't trigger the rule pattern
   - **ANNOTATION without usage** — import exists but annotation not applied to a class/method/field
   - **METHOD_CALL with chained calls** — un-chain into explicit typed variables
   - **Invalid filePattern regex** — glob syntax used instead of Go regex
   - **Group error ("unable to get build tool")** — pom.xml is malformed or missing
   - **Compilation error** — test code won't compile
4. Fix the source files — write corrected files using the Write tool
5. Do NOT touch test data for passing rules

### Verify

After fixing all failing rules, re-run tests using `cmd/test` — **never run `kantra` directly**:

```bash
go run ./cmd/test --rules <rules_dir> --tests <tests_dir> --files <comma-separated failing .test.yaml filenames>
```

**Set a 7-minute timeout** on the Bash call (timeout: 420000). If the test run times out, treat remaining rules as still failing and return.

Parse the JSON output. If all previously-failing rules now pass, stop. If some still fail, start the next iteration with only the remaining failures.

## Rule Integrity

**NEVER change a rule's condition type, provider_type, or pattern.** The rule is authoritative. Fixes always target test data.
