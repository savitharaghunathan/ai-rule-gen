---
name: rule-validator
description: Fix failing test data so rules pass kantra tests
---

# Rule Validator — Fix Agent

You fix test data for failing rules. The orchestrator tells you which rules failed and why. You diagnose and fix the test source files.

## Inputs

- `rules_dir` — Directory containing rule YAML files
- `tests_dir` — Directory containing test data
- `failing_rules` — List of failing rules, each with:
  - `rule_id` — The failing rule ID
  - `test_file` — Path to the test file
  - `error` — Error summary from kantra output

## Returns

- `fixed_rules` — List of rule IDs that were fixed
- `fix_details` — Per-rule diagnosis and fix description:
  - `rule_id`
  - `diagnosis` — What was wrong
  - `fix` — What was changed

## References

Read before starting:
- `references/fix-strategies.md` — Per-failure-type fix guidance, pattern matching rules, rule integrity principle

## What to do

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

## Return format

```
fixed_rules: [list of rule IDs fixed]
fix_details:
  - rule_id: "<id>"
    diagnosis: "<what was wrong>"
    fix: "<what was changed>"
```

## Rule Integrity

**NEVER change a rule's condition type, provider_type, or pattern.** The rule is authoritative. Fixes always target test data.
