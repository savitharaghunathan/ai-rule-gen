# Generate Rules Orchestrator Details

This reference contains expanded operational details that are intentionally kept
out of the core `SKILL.md` to reduce prompt size.

## Status Line Format

Use concise status lines:

```text
[step-name] message
```

Example:

```text
[ingest] Fetching guide from https://...
[ingest] Done — 3876 lines, 77 sections
[extract] Extracting patterns from 62 sections (3 parallel agents)...
[extract] Done — 52 patterns → 52 rules
[coverage] No gaps found
[test-gen] Generating test data for 52 rules...
[test-gen] Done — 12 groups, 39 files
[validate] 49/52 passed — fixing 3 failures
[fix] 2 fixed, 1 still failing
[done] 52 rules generated
```

## Parallelism Defaults

- Extraction: 2-5 parallel `rule-writer` invocations
- Test generation: 1-5 parallel `test-generator` invocations
- Balance by rule/section count per worker

## Resume Stage Preconditions

These are the canonical artifact checks for `resume_from`:

- `extract` requires `guide.md`
- `coverage` requires `patterns.json`
- `scaffold` requires `rules/`
- `test` requires `rules/` and `tests/manifest.json`
- `stamp` requires `rules/` and test execution results
- `report` requires stamped rules or pass/fail lists

If prerequisites are missing, fail fast with a structured error and do not infer
or regenerate stages implicitly unless explicitly instructed.

## Rebuild Behavior

- `force_rebuild=true` means regenerate current stage outputs even if present.
- `force_rebuild=false` means reuse existing artifacts when preconditions hold.
