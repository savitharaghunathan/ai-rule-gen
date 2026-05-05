---
name: generate-rules
description: Generate Konveyor analyzer migration rules from a migration guide (URL, file path, or pasted text)
---

# Generate Konveyor Migration Rules

Generate Konveyor analyzer migration rules from a migration guide.

**Input:** $ARGUMENTS (URL, file path, or pasted text of a migration guide)

If no argument is provided, ask the user for the migration guide source.

## Inputs

- `guide_source` — URL, file path, or pasted text of a migration guide
- `source` — (optional) Source technology, e.g. "spring-boot-3". Auto-detected if omitted.
- `target` — (optional) Target technology, e.g. "spring-boot-4". Auto-detected if omitted.

## Returns

- `rules_dir` — Path to generated rule YAML files
- `tests_dir` — Path to test data
- `report` — Path to report.yaml
- `summary` — Markdown summary table with:
  - rules_count, passed, failed, pass_rate
  - coverage_report (sections processed/skipped)
  - fix_iterations used

## Permissions

| Operation | Pattern | Purpose |
|-----------|---------|---------|
| shell | `mkdir -p output` | Create output directory |
| shell | `go run ./cmd/ingest *` | Fetch migration guide as markdown |
| shell | `go run ./cmd/sections *` | Index guide sections with classification |
| shell | `go run ./cmd/merge-patterns *` | Merge partial patterns files |
| shell | `go run ./cmd/construct *` | Build rule YAML from patterns.json |
| shell | `go run ./cmd/validate *` | Validate rule YAML structure |
| shell | `go run ./cmd/scaffold *` | Create test directories and manifests |
| shell | `go run ./cmd/sanitize *` | Fix XML comments in test data |
| shell | `go run ./cmd/test *` | Run kantra tests |
| shell | `go run ./cmd/stamp *` | Mark rules with pass/fail labels |
| shell | `go run ./cmd/report *` | Generate summary report |
| shell | `go run ./cmd/coverage *` | Check guide coverage |
| shell | `wc -l *` | Count guide lines |
| shell | `grep *` | Count sections, search output |
| read | `output/**` | Read manifest, test results |
| write | `output/guide.md` | Write pasted/markdown guide to file |

## UX Principles

The user should see a **flowing stream of short status lines** — never silence. Every step prints one line when it starts and one line when it finishes. No walls of text, no tables mid-pipeline, no unnecessary questions.

**Checkpoint after extraction.** After extraction completes, ask the user: "Continue with test generation and validation?" This is the only question in the pipeline. If they say no or say to skip testing, finalize with untested rules.

**Don't ask beyond the checkpoint.** Once testing starts, run the full pipeline (test → validate → fix → finalize) without further questions. If something fails after 3 fix attempts, report it and move on — don't block.

**Only fix what's broken.** When the fix loop runs, only re-generate and re-validate the failing rules, not the entire suite.

## Output Format

Use this format for every status line:

```
[step-name] message
```

Example full run:

```
[ingest] Fetching guide from https://...
[ingest] Done — 3876 lines, 77 sections

[extract] Extracting patterns from 62 sections (3 parallel agents)...
[extract] Done — 52 patterns → 52 rules in output/rules/

[coverage] No gaps found

[test-gen] Generating test data for 52 rules...
[test-gen] Done — 12 groups, 39 files

[validate] Running tests on 12 groups...
[validate] 49/52 passed — fixing 3 failures
[validate] Fix 1/3 — 51/52 passed
[validate] Fix 2/3 — 52/52 passed

[done] 52 rules generated, 52/52 passed — output/rules/
```

That's the entire user-visible output. Everything else happens silently in sub-agents.

## Sub-agent Protocol

This orchestrator delegates heavy LLM work to sub-agents using **invoke blocks**. Each block names the skill, passes inputs, and states what it expects back.

The runtime translates each invoke block into a sub-agent call:
1. "Read and follow `agents/<skill-name>/SKILL.md`."
2. Inputs from the invoke block, with actual values substituted.

If the runtime supports parallel sub-agents, invoke blocks marked `Parallel: yes` should be dispatched concurrently. If the runtime does not support parallel dispatch or sub-agents, run all invoke blocks sequentially in the current agent context — read and follow each sub-skill's SKILL.md inline.

**Parallel extraction:** The guide is split into chunks by section, and multiple rule-writer agents process chunks concurrently. The orchestrator merges the partial patterns files and runs construct/validate once. Each agent reads only its assigned sections from the guide.

**Do NOT read sub-agent references.** The orchestrator must NOT read files under `agents/<skill>/references/` — sub-agents read their own references. The orchestrator only needs to know the invoke contract (inputs/returns). Reading references wastes context and risks the orchestrator overriding sub-agent decisions with its own interpretation of reference material.

**Do NOT micro-manage sub-agent work.** Pass the inputs specified in the invoke block. Do not pre-digest rule YAML, pre-read the guide, or compose line-by-line instructions. The sub-agent's SKILL.md tells it what to read and how to work.

## Pipeline

**Error handling:** If any CLI command (`ingest`, `construct`, `validate`, `scaffold`, `sanitize`, `test`, `stamp`, `report`) exits non-zero, print `[step] FAILED — <error>` and stop the pipeline. Do not proceed to the next step.

**Partial failures:** If a step produces partial results before failing (e.g., `construct` succeeds for 48 patterns but fails on 2), preserve the partial output and report what succeeded. The user can fix the failing patterns and re-run. Do not discard valid results because of a partial failure.

### 1. Ingest

```
[ingest] Fetching guide from <source>...
```

```bash
mkdir -p output
```

- **URL:** `go run ./cmd/ingest --input <url> --output output/guide.md`
- **File (not markdown):** `go run ./cmd/ingest --input <path> --output output/guide.md`
- **Pasted text or already markdown:** Write directly to `output/guide.md`

Count lines (`GUIDE_LINES`) and section headings. Print:

```
[ingest] Done — <GUIDE_LINES> lines, <N> sections
```

### 2. Extract (parallel)

Extraction uses parallel agents to process the guide faster. The orchestrator splits the guide into chunks and spawns multiple rule-writer agents.

**2a. Index sections and auto-detect metadata:**

```bash
go run ./cmd/sections --guide output/guide.md
```

This returns JSON with all sections classified as `content` or `header-only`. Filter to content sections only.

Read the first ~50 lines of the guide to auto-detect source, target, and language. Use lowercase, hyphenated names (e.g., `spring-boot-3`). If the user provided source/target, use those instead.

```
[extract] Extracting patterns from <content_count> sections (<N> parallel agents)...
```

**2b. Split into chunks and dispatch:**

Split the content sections into **N balanced chunks** (minimum 2, maximum 5 agents). Balance by section count. Assign each chunk a number (1, 2, ..., N).

**Invoke:** `rule-writer` (one per chunk, in parallel)
**Purpose:** Extract migration patterns from assigned sections.
**Inputs per invocation:**
  - guide: output/guide.md
  - source: {detected source}
  - target: {detected target}
  - rules_dir: output/rules
  - sections: {chunk subset — list of `{heading, start_line, end_line}` objects}
  - output_file: output/patterns-{chunk_number}.json
**Parallel:** yes
**Expect:**
  - patterns_count, output_file

**2c. Merge and construct:**

After all agents complete, merge the partial patterns files and build rules:

```bash
go run ./cmd/merge-patterns --output patterns.json output/patterns-1.json output/patterns-2.json ...
go run ./cmd/construct --patterns patterns.json --output output/rules
go run ./cmd/validate --rules output/rules
```

If validation fails, fix the patterns.json (remove invalid entries) and re-run construct.

```
[extract] Done — <patterns_count> patterns → <rules_count> rules in output/rules/
```

Save `source`, `target`, `patterns_count`, `rules_count` for the final summary.

### 2d. Coverage Check

Run the coverage tool to find sections with named artifacts that weren't extracted:

```bash
go run ./cmd/coverage --guide output/guide.md --patterns patterns.json --language <language>
```

If `gap_count > 0` in the JSON output, print:

```
[coverage] <gap_count> sections with uncovered artifacts: <heading_1>, <heading_2>, ...
```

Convert gaps to sections and send back to the rule-writer in chunk mode for targeted re-extraction. Each gap has `heading` and `line` — use the sections index to get the `start_line` and `end_line` for that heading. Mention the uncovered artifacts in the prompt so the sub-agent focuses on them.

**Invoke:** `rule-writer` (chunk mode)
**Purpose:** Extract patterns from specific sections that the coverage check flagged.
**Inputs:**
  - guide: output/guide.md
  - source: {detected source}
  - target: {detected target}
  - rules_dir: output/rules
  - sections: {gap sections converted to `[{heading, start_line, end_line}]` format}
  - output_file: output/patterns-gaps.json
**Parallel:** no
**Expect:**
  - patterns_count, output_file

The sub-agent writes only the new patterns to `output/patterns-gaps.json`. It does NOT read or inspect existing patterns — the orchestrator handles deduplication. After the sub-agent returns, merge and rebuild:

```bash
go run ./cmd/merge-patterns --output patterns.json patterns.json output/patterns-gaps.json
go run ./cmd/construct --patterns patterns.json --output output/rules
go run ./cmd/validate --rules output/rules
```

Re-run the coverage check. If gaps remain, accept them — one re-extraction pass is enough.

```
[coverage] Done — <final_gap_count> remaining gaps (accepted)
```

If `gap_count == 0`:

```
[coverage] No gaps found
```

### Checkpoint

After printing the extract summary, ask the user:

```
Continue with test generation and validation? (y/n)
```

If the user declines, skip to Step 6 (Summary) with untested rules. Otherwise continue.

### 3. Test Generation

**3a. Scaffold (orchestrator runs this directly, not an agent):**

```bash
go run ./cmd/scaffold --rules output/rules --output output/tests
```

This creates all directories, `.test.yaml` files, and `manifest.json`. No LLM needed.

**3b. Read manifest and split into batches:**

Read `output/tests/manifest.json`. It contains a `groups` array — each group has `name`, `data_dir`, `files` (paths to generate), and `rule_ids`.

Split the groups into **batches of ~5 groups each** (minimum 1 batch, maximum 5 batches), balanced by rule count.

**3c. Spawn test-generator agents:**

```
[test-gen] Generating test data for <rules_count> rules (<B> parallel agents)...
```

**Invoke:** `test-generator` (one per batch, up to 5 in parallel)
**Purpose:** Generate compilable test source code that triggers the assigned rules.
**Inputs per invocation:**
  - rules_dir: output/rules
  - tests_dir: output/tests
  - groups: {batch subset from manifest.json — include name, data_dir, rule_ids, files}
**Parallel:** yes
**Expect:**
  - groups_completed, files_written

**3d. Collect results and sanitize.** After all agents complete:

Sanitize XML:

```bash
go run ./cmd/sanitize --dir output/tests/tests/data
```

```
[test-gen] Done — <total_groups> groups, <total_files> files
```

### 4. Validate (orchestrator-driven loop)

The orchestrator runs tests and uses a sub-agent for LLM-driven repairs on failures.

**4a. Run tests:**

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --timeout 5m
```

The CLI runs each test file sequentially (avoids Docker contention) and automatically retries timed-out files once (`--retry-timeouts`, on by default). To run a subset, use `--files` with bare filenames (e.g., `--files data-1.test.yaml,data-2.test.yaml`), resolved relative to `--tests` dir.

Print the result:

```
[validate] <total_passed>/<total_rules> passed
```

Or if failures:

```
[validate] <total_passed>/<total_rules> passed — <F> failures: <rule_id_1>, <rule_id_2>, ...
```

If all passed, skip to step 5.

**4b. Fix (only if failures):**

```
[fix] Fixing <F> failures: <rule_id_1>, <rule_id_2> ...
```

**Invoke:** `rule-validator`
**Purpose:** Fix test data for failing rules so they pass kantra validation.
**Inputs:**
  - rules_dir: output/rules
  - tests_dir: output/tests/tests
  - failing_rules: {list with rule_id, test_file, error for each failure}
  - max_iterations: 1
**Parallel:** no
**Expect:**
  - fixed_rules, still_failing, iterations_used, fix_details

The validator agent owns the fix-verify loop — it fixes files, re-runs tests via `cmd/test`, and iterates up to `max_iterations`. Default is 1 iteration; pass up to 3 if the user requests more attempts.

**The orchestrator must NOT run its own fix attempts.** Do not manually edit test files, create stub classes, toggle analysisParams, or retry. All fix work goes through the rule-validator sub-agent. When the validator returns `still_failing`, accept that result and move on.

Print the result:

```
[fix] <fixed_count> fixed, <still_failing_count> still failing
```

If still_failing is non-empty, move on — don't block the pipeline.

### 5. Stamp + Report

Collate pass/fail results from the test run and the fix loop. No need to re-run the full test suite — stamp directly from results:

```bash
go run ./cmd/stamp --rules output/rules --passed <comma-separated passed rule IDs> --failed <comma-separated failed rule IDs>
go run ./cmd/report --source <source> --target <target> --output output/report.yaml --rules-total <N> --passed <P> --failed <F> --failed-rules <comma-separated>
```

### 6. Summary

Print a formatted summary table using GitHub-flavored markdown:

```markdown
## Summary

| | |
|---|---|
| **Input** | <guide title or URL as a markdown link> |
| **Migration** | <source> → <target> (<language>) |
| **Guide** | <GUIDE_LINES> lines, <N> sections → <M> produced patterns, <K> skipped |
| **Rules** | <rules_count> generated, **<P>/<N> passed (<percent>%)** |
| **Fix iterations** | <iterations used, 0 if none> |
| **Output** | `output/rules/` (rules), `output/tests/` (tests), `output/report.yaml` (report) |

### Rule Categories

Use the `groups` array from the `construct` JSON output — it lists each file and its rule count. Do not iterate rule files manually.

| Group | Rules | Status |
|---|---|---|
| <group_name> (<brief description of what rules cover>) | <rule_count> | <passed>/<total> passed |
| ... | ... | ... |
```

If there are failures, add a row to the top-level table:

```
| **Failed** | <rule_id_1>, <rule_id_2>, ... |
```

If coverage was low (< 30% of sections produced patterns), add a row:

```
| **Warning** | Low extraction coverage (<M>/<K> sections) |
```
