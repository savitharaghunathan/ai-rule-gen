---
name: generate-rules
description: Generate Konveyor analyzer migration rules from a migration guide (URL, file path, or pasted text)
---

# Generate Konveyor Migration Rules

Generate Konveyor analyzer migration rules from a migration guide.

**Input:** $ARGUMENTS (URL, file path, or pasted text of a migration guide)

If no argument is provided, ask the user for the migration guide source.

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

[extract] Extracting migration patterns...
[extract] Done — 52 patterns → 52 rules in output/rules/

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

If the runtime supports parallel sub-agents, invoke blocks marked `Parallel: yes` should be dispatched concurrently.

**Single agent for extraction:** One agent processing the full guide is faster than batching across multiple agents (avoids duplicated reference reads and merge overhead).

## Pipeline

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

### 2. Extract

```
[extract] Extracting migration patterns...
```

**Invoke:** `rule-writer`
**Purpose:** Extract migration patterns from the guide and produce validated rules.
**Inputs:**
  - guide: output/guide.md
  - source: {source if user specified, otherwise "auto-detect"}
  - target: {target if user specified, otherwise "auto-detect"}
  - rules_dir: output/rules
**Parallel:** no
**Expect:**
  - source, target, patterns_count, rules_count, rules_dir, coverage_report

Run syntactic validation and print the result:

```bash
go run ./cmd/validate --rules output/rules
```

Print:

```
[extract] Done — <patterns_count> patterns → <rules_count> rules in output/rules/
[extract] Validation: <result from cmd/validate>
```

Save `source`, `target`, `patterns_count`, `rules_count`, and the coverage report for the final summary.

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

Split the groups into **3 roughly equal batches** by rule count.

**3c. Spawn test-generator agents:**

```
[test-gen] Generating test data for <rules_count> rules (3 parallel agents)...
```

**Invoke:** `test-generator` (one per batch, 3 in parallel)
**Purpose:** Generate compilable test source code that triggers the assigned rules.
**Inputs per invocation:**
  - rules_dir: output/rules
  - tests_dir: output/tests
  - groups: {batch subset from manifest.json — include name, data_dir, rule_ids, files}
**Parallel:** yes
**Expect:**
  - groups_completed, files_written

**3d. Collect results and sanitize.** After all 3 agents complete:

```bash
go run ./cmd/sanitize --dir output/tests/tests/data
```

```
[test-gen] Done — <total_groups> groups, <total_files> files
```

### 4. Validate (orchestrator-driven loop)

The orchestrator runs tests directly in **batched sequential runs** to avoid OOM (one giant kantra run) and Docker contention (parallel kantra runs). The fix loop uses a sub-agent for LLM-driven repairs.

**4a. Batch and run tests:**

Split the groups from `manifest.json` into **3 roughly equal batches** by rule count (same batching as test-gen). Run each batch sequentially using `--files`:

```
[validate] Running batch 1/3: <group_names> (<N> rules)...
```

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <batch1.test.yaml>,<batch2.test.yaml>,...
```

```
[validate] Running batch 2/3: <group_names> (<N> rules)...
```

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <batch3.test.yaml>,<batch4.test.yaml>,...
```

```
[validate] Running batch 3/3: <group_names> (<N> rules)...
```

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <batch5.test.yaml>,<batch6.test.yaml>,...
```

**Why batched sequential, not parallel agents:** `kantra test` runs Docker containers. Multiple kantra instances running simultaneously cause Docker contention and hangs. Sequential batches keep each kantra run small (avoids OOM) while avoiding contention.

**`--files` takes bare filenames** (e.g., `data-1.test.yaml`), resolved relative to `--tests` dir. The runner scopes results to only the rules referenced by those test files.

Collate results across all batches. Print the combined result:

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

Print the result:

```
[fix] <fixed_count> fixed, <still_failing_count> still failing
```

If still_failing is non-empty, move on — don't block the pipeline.

### 5. Stamp + Report

Collate pass/fail results from all batch runs and the fix loop. No need to re-run the full test suite — stamp directly from results:

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
