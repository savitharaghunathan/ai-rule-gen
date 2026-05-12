---
name: generate-rules
description: Generate Konveyor analyzer migration rules from a migration guide (URL, file path, or pasted text)
---

# Generate Konveyor Migration Rules

Generate Konveyor analyzer migration rules from a migration guide.

**Input:** $ARGUMENTS (URL, file path, or pasted text of a migration guide)

If no argument is provided, ask the user for the migration guide source.

**IMPORTANT — Fresh run every time:** Each invocation creates a new timestamped output directory. Do NOT scan `output/` for previous runs, do NOT reuse existing directories, do NOT check for prior results. Start step 1 immediately. The only exception is when the user explicitly provides `migration_dir` with `resume_from`.

## Inputs

- `guide_source` — URL, file path, or pasted text of a migration guide
- `source` — (optional) Source technology, e.g. "spring-boot-3". Auto-detected if omitted.
- `target` — (optional) Target technology, e.g. "spring-boot-4". Auto-detected if omitted.
- `mode` — (optional) `interactive` (default) or `non_interactive`
- `checkpoint_behavior` — (optional) `ask` (default), `continue`, or `stop_after_extract`
- `resume_from` — (optional) `ingest`, `extract`, `coverage`, `scaffold`, `test`, `stamp`, `report`
- `migration_dir` — (optional) explicit output directory override. If omitted, auto-generated as `output/<source>-to-<target>-<YYYYMMDD-HHMMSS>`. Required when using `resume_from`.

## Returns

- `rules_dir` — Path to generated rule YAML files
- `tests_dir` — Path to test data
- `report` — Path to report.yaml
- `summary` — Markdown summary table with:
  - rules_count, passed, failed, pass_rate
  - coverage_report (sections processed/skipped)
  - fix_iterations used

## Execution Modes

- `interactive` mode: ask at checkpoint only when `checkpoint_behavior=ask`
- `non_interactive` mode: never prompt the user; obey `checkpoint_behavior`
  - `continue`: run full pipeline
  - `stop_after_extract`: end after extraction/coverage with untested outputs
  - `ask`: treated as `continue` (no prompt allowed in non-interactive mode)

## References

- `references/orchestrator-details.md` — status-line conventions, parallelism defaults, resume preconditions, rebuild behavior

## Permissions

| Operation | Pattern | Purpose |
|-----------|---------|---------|
| shell | `mkdir -p output` | Create output directory |
| shell | `go run ./cmd/ingest *` | Fetch migration guide as markdown |
| shell | `go run ./cmd/sections *` | Index guide sections with classification |
| shell | `go run ./cmd/merge-patterns *` | Merge partial patterns files |
| shell | `go run ./cmd/contract-validate *` | Validate sub-agent payload contracts |
| shell | `go run ./cmd/verify *` | Verify patterns against source artifacts |
| shell | `go run ./cmd/construct *` | Build rule YAML from patterns.json |
| shell | `go run ./cmd/validate *` | Validate rule YAML structure |
| shell | `go run ./cmd/scaffold *` | Create test directories and manifests |
| shell | `go run ./cmd/sanitize *` | Fix XML comments in test data |
| shell | `go run ./cmd/test *` | Run kantra tests |
| shell | `go run ./cmd/stamp *` | Mark rules with pass/fail labels |
| shell | `go run ./cmd/report *` | Generate summary report |
| shell | `go run ./cmd/coverage *` | Check guide coverage |
| shell | `go run ./cmd/log *` | Append orchestrator events to pipeline log |
| shell | `ls *` | List files and directories |
| read | `output/**` | Read manifest, test results |
| write | `output/**` | Write migration artifacts |

**Do NOT use `python`, `python3`, `node`, or any scripting language runtime.**
**Do NOT use `grep`, `sed`, `awk`, `wc`, `find`, or other shell text-processing tools to parse data that CLI commands already return.**
Every `go run ./cmd/*` command returns structured JSON. Use that JSON output directly — do not pipe it through shell commands to extract fields.
This is a Go project. Run only commands listed in this permissions table.
If a required action is not permitted, stop and report the limitation.

## UX Principles

- Emit short status lines continuously (start + finish for each step).
- Only ask at checkpoint when `mode=interactive` and `checkpoint_behavior=ask`.
- Do not block after testing starts; continue even if some rules remain failing.
- Fix only failing rules, not the full suite.

## Pipeline Transcript

The pipeline log (`<migration_dir>/pipeline.log`) captures the full session. **All logging goes through CLI commands** — no manual `echo >>` or shell variable assignments.

### CLI command logging (automatic)

Every `go run ./cmd/*` invocation auto-logs its JSON output when you pass `--log`. Pass `--log <migration_dir>/pipeline.log --agent orchestrator --model <your_model_id>` to every CLI invocation. Substitute actual paths and values directly — do not use shell variables. Sub-agents pass their own `--agent` and `--model` values.

### Orchestrator event logging (via cmd/log)

Log agent-level events using `cmd/log` instead of shell echo commands:

```bash
go run ./cmd/log --log <migration_dir>/pipeline.log --agent orchestrator --model <your_model_id> --message "<message>"
```

Events to log:
- Pipeline start: guide source, detected source/target/language
- Every `[step]` status line (start and done)
- Sub-agent dispatches: skill name, chunk number, agent name, model
- Sub-agent returns: patterns_count, files_written, errors, suspected_kantra_limitations
- Errors, retries, and partial failures
- Final summary table

### Sub-agent CLI invocations

When a sub-agent invokes a CLI command (e.g., rule-validator running `cmd/test`), the sub-agent passes its own identity:

```bash
go run ./cmd/test --rules ... --tests ... --log <migration_dir>/pipeline.log --agent rule-validator --model <agent_model>
```

This ensures every log entry is traceable to which agent ran it and which LLM powered the decision.

## Timing

Track wall-clock elapsed time for the pipeline and each major stage. Record the current time (HH:MM:SS) at the start of the pipeline and at the start/end of each stage.

**Pipeline timer:** Record start time before step 1. Print total elapsed in the final summary.

**Stage timers:** For each stage, include elapsed time in the "Done" status line using the format `(<elapsed>)` where elapsed is `Xm Ys` (e.g. `2m 15s`). Omit hours unless the stage takes ≥ 60 minutes.

Format:
```
[step] Done — <metrics> (<elapsed>)
```

Stages to time:
- **ingest** — from guide fetch start to guide written
- **extract** — from section indexing start to rules constructed + validated (includes parallel rule-writer agents, merge, construct, validate)
- **coverage** — from coverage check start to re-extraction complete (or "No gaps found")
- **test-gen** — from scaffold start to all test-generator agents complete + sanitize
- **validate** — from first `cmd/test` run to fix loop complete (includes all fix iterations)
- **stamp+report** — from stamp start to report written

## Sub-agent Protocol

This orchestrator delegates heavy LLM work to sub-agents using **invoke blocks**. Each block names the skill, passes inputs, and states what it expects back.

The runtime translates each invoke block into a sub-agent call:
1. "Read and follow `agents/<skill-name>/SKILL.md`."
2. Inputs from the invoke block, with actual values substituted.

**Contract validation is mandatory.** For every invoke:
- Validate invoke input JSON against `agents/<skill>/contract.json` with `--mode inputs`
- Validate sub-agent return JSON against `agents/<skill>/contract.json` with `--mode returns`
- If validation fails, stop that step and print `[step] FAILED — contract validation error`

Example:

```bash
go run ./cmd/contract-validate <log_flags> --contract agents/rule-writer/contract.json --mode inputs --payload-file <migration_dir>/contracts/rule-writer-input-1.json
go run ./cmd/contract-validate <log_flags> --contract agents/rule-writer/contract.json --mode returns --payload-file <migration_dir>/contracts/rule-writer-return-1.json
```

If the runtime supports parallel sub-agents, invoke blocks marked `Parallel: yes` should be dispatched concurrently. If the runtime does not support parallel dispatch or sub-agents, run all invoke blocks sequentially in the current agent context — read and follow each sub-skill's SKILL.md inline.

**Parallel extraction:** The guide is split into chunks by section, and multiple rule-writer agents process chunks concurrently. The orchestrator merges the partial patterns files and runs construct/validate once. Each agent reads only its assigned sections from the guide.

**Do NOT read sub-agent references.** The orchestrator must NOT read files under `agents/<skill>/references/` — sub-agents read their own references. The orchestrator only needs to know the invoke contract (inputs/returns). Reading references wastes context and risks the orchestrator overriding sub-agent decisions with its own interpretation of reference material.

**Do NOT micro-manage sub-agent work.** Pass the inputs specified in the invoke block. Do not pre-digest rule YAML, pre-read the guide, or compose line-by-line instructions. The sub-agent's SKILL.md tells it what to read and how to work.

## Pipeline

**Error handling:** If any CLI command (`ingest`, `construct`, `validate`, `scaffold`, `sanitize`, `test`, `stamp`, `report`) exits non-zero, print `[step] FAILED — <error>` and stop the pipeline. Do not proceed to the next step.

**Partial failures:** If a step produces partial results before failing (e.g., `construct` succeeds for 48 patterns but fails on 2), preserve the partial output and report what succeeded. The user can fix the failing patterns and re-run. Do not discard valid results because of a partial failure.

**Resume behavior:** When `resume_from` is set, the user must also provide `migration_dir` pointing to the specific timestamped directory to resume. Skip prior stages and validate prerequisites first. Fail fast when required artifacts are missing.

Stage prerequisites (paths relative to `<migration_dir>`):
- `extract` requires `guide.md`
- `coverage` requires `patterns.json`
- `scaffold` requires `rules/`
- `test` requires `rules/` and `tests/manifest.json`
- `stamp` requires stamped pass/fail test results and `rules/`
- `report` requires source/target metadata plus pass/fail totals

### 1. Ingest

Generate a timestamp once at pipeline start (`YYYYMMDD-HHMMSS` in local time).

Determine the migration directory path:
- If the user provided `migration_dir`, use that (no timestamp)
- If the user provided `source` and `target`, use `output/<source>-to-<target>-<YYYYMMDD-HHMMSS>`
- If auto-detecting: ingest to `output/guide-temp.md` first, read the first ~50 lines to detect source/target/language (lowercase, hyphenated names like `spring-boot-3`), then use `output/<source>-to-<target>-<YYYYMMDD-HHMMSS>`

```bash
mkdir -p output/<source>-to-<target>-<YYYYMMDD-HHMMSS>
```

Every `go run` command should be a standalone invocation with fully-expanded arguments — substitute actual paths and flag values directly into each command. Do not use shell variable assignments (`VAR=...`) or compound `&&` chains, since these don't match the permissions table.

The log flags are: `--log <migration_dir>/pipeline.log --agent orchestrator --model <your_model_id>`. Append these literally to every CLI invocation.

```
[ingest] Fetching guide from <source>...
```

- **URL:** `go run ./cmd/ingest --log <migration_dir>/pipeline.log --agent orchestrator --model <your_model_id> --input <url> --output <migration_dir>/guide.md`
- **File (not markdown):** `go run ./cmd/ingest --log <migration_dir>/pipeline.log --agent orchestrator --model <your_model_id> --input <path> --output <migration_dir>/guide.md`
- **Pasted text or already markdown:** Write directly to `<migration_dir>/guide.md`

If auto-detecting, write `output/guide-temp.md` to `<migration_dir>/guide.md` using the Write tool after the migration directory path is determined.

Count lines (`GUIDE_LINES`) and section headings. Print:

```
[ingest] Done — <GUIDE_LINES> lines, <N> sections → <migration_dir>/ (<elapsed>)
```

### 2. Extract (parallel)

Extraction uses parallel agents to process the guide faster. The orchestrator splits the guide into chunks and spawns multiple rule-writer agents.

**2a. Index sections and auto-detect metadata:**

```bash
go run ./cmd/sections <log_flags> --guide <migration_dir>/guide.md
```

This returns JSON with all sections classified as `content` or `header-only`. Filter to content sections only.

```
[extract] Extracting patterns from <content_count> sections (<N> parallel agents)...
```

**2b. Split into chunks and dispatch:**

Split the content sections into **N balanced chunks** (minimum 1, maximum 5 agents). Balance by section count. Assign each chunk a number (1, 2, ..., N).

**Invoke:** `rule-writer` (one per chunk, in parallel)
**Purpose:** Extract migration patterns from assigned sections.
**Inputs per invocation:**
  - guide: <migration_dir>/guide.md
  - source: {detected source}
  - target: {detected target}
  - rules_dir: <migration_dir>/rules
  - sections: {chunk subset — list of `{heading, start_line, end_line}` objects}
  - output_file: <migration_dir>/patterns-{chunk_number}.json
**Parallel:** yes
**Expect:**
  - patterns_count, output_file

**2c. Merge, verify, and construct:**

After all agents complete, merge the partial patterns files, verify source artifacts, and build rules:

```bash
go run ./cmd/merge-patterns <log_flags> --output <migration_dir>/patterns.json <migration_dir>/patterns-1.json <migration_dir>/patterns-2.json ...
go run ./cmd/verify <log_flags> --patterns <migration_dir>/patterns.json --output <migration_dir>/verify-results.json --cache <migration_dir>/verify-cache
go run ./cmd/construct <log_flags> --patterns <migration_dir>/patterns.json --output <migration_dir>/rules
go run ./cmd/validate <log_flags> --rules <migration_dir>/rules
```

**Verification step:** `cmd/verify` downloads source JARs from Maven Central and checks that each pattern's `source_fqn` exists in the declared `source_artifact`. This catches hallucinated class names before rules reach testing.

The verify stdout JSON contains `verified`, `not_found`, `skipped`, and `offline` counts. Print:

```
[verify] <verified> verified, <not_found> not found, <skipped> skipped
```

If `not_found > 0`, read `<migration_dir>/verify-results.json` and note which `pattern_index` values have `status: "not_found"`. After construct runs, use the `pattern_rule_map` from construct's stdout JSON to map each `pattern_index` → `rule_id`. Carry `verified_rule_ids` and `not_found_rule_ids` forward to step 5 (stamp).

If validation fails, fix `<migration_dir>/patterns.json` (remove invalid entries) and re-run construct.

```
[extract] Done — <patterns_count> patterns → <rules_count> rules in <migration_dir>/rules/ (<elapsed>)
```

Save `source`, `target`, `patterns_count`, `rules_count`, and verify counts for the final summary.

### 2d. Coverage Check

Run the coverage tool to find sections with named artifacts that weren't extracted:

```bash
go run ./cmd/coverage <log_flags> --guide <migration_dir>/guide.md --patterns <migration_dir>/patterns.json --language <language>
```

If `gap_count > 0` in the JSON output, print:

```
[coverage] <gap_count> sections with uncovered artifacts: <heading_1>, <heading_2>, ...
```

Convert gaps to sections and send back to the rule-writer in chunk mode for targeted re-extraction. Each gap has `heading` and `line` — use the sections index to get the `start_line` and `end_line` for that heading. Mention the uncovered artifacts in the prompt so the sub-agent focuses on them.

**Invoke:** `rule-writer` (chunk mode)
**Purpose:** Extract patterns from specific sections that the coverage check flagged.
**Inputs:**
  - guide: <migration_dir>/guide.md
  - source: {detected source}
  - target: {detected target}
  - rules_dir: <migration_dir>/rules
  - sections: {gap sections converted to `[{heading, start_line, end_line}]` format}
  - output_file: <migration_dir>/patterns-gaps.json
**Parallel:** no
**Expect:**
  - patterns_count, output_file

The sub-agent writes only the new patterns to `<migration_dir>/patterns-gaps.json`. It does NOT read or inspect existing patterns — the orchestrator handles deduplication. After the sub-agent returns, merge, verify, and rebuild:

```bash
go run ./cmd/merge-patterns <log_flags> --output <migration_dir>/patterns.json <migration_dir>/patterns.json <migration_dir>/patterns-gaps.json
go run ./cmd/verify <log_flags> --patterns <migration_dir>/patterns.json --output <migration_dir>/verify-results.json --cache <migration_dir>/verify-cache
go run ./cmd/construct <log_flags> --patterns <migration_dir>/patterns.json --output <migration_dir>/rules
go run ./cmd/validate <log_flags> --rules <migration_dir>/rules
```

Re-run the coverage check. If gaps remain, accept them — one re-extraction pass is enough.

```
[coverage] Done — <final_gap_count> remaining gaps (accepted) (<elapsed>)
```

If `gap_count == 0`:

```
[coverage] No gaps found (<elapsed>)
```

### Checkpoint

After extraction/coverage:

- If `mode=non_interactive`, do **not** prompt:
  - `checkpoint_behavior=continue` or `ask`: continue automatically
  - `checkpoint_behavior=stop_after_extract`: skip testing and go to Step 6
- If `mode=interactive`:
  - `checkpoint_behavior=continue`: continue automatically
  - `checkpoint_behavior=stop_after_extract`: skip testing and go to Step 6
  - `checkpoint_behavior=ask`: ask:
    - `Continue with test generation and validation? (y/n)`
    - if no, skip to Step 6 with untested rules

### 3. Test Generation

**3a. Scaffold (orchestrator runs this directly, not an agent):**

```bash
go run ./cmd/scaffold <log_flags> --rules <migration_dir>/rules --output <migration_dir>/tests
```

This creates all directories, `.test.yaml` files, and `manifest.json`. No LLM needed.

**3b. Read manifest and split into batches:**

Read `<migration_dir>/tests/manifest.json`. It contains a `groups` array — each group has `name`, `data_dir`, `files` (paths to generate), and `rule_ids`.

Split the groups into **batches of ~5 groups each** (minimum 1 batch, maximum 5 batches), balanced by rule count.

**3c. Spawn test-generator agents:**

```
[test-gen] Generating test data for <rules_count> rules (<B> parallel agents)...
```

**Invoke:** `test-generator` (one per batch, up to 5 in parallel)
**Purpose:** Generate compilable test source code that triggers the assigned rules.
**Inputs per invocation:**
  - rules_dir: <migration_dir>/rules
  - tests_dir: <migration_dir>/tests
  - groups: {batch subset from manifest.json — include name, data_dir, rule_ids, files}
**Parallel:** yes
**Expect:**
  - groups_completed, files_written

**3d. Collect results and sanitize.** After all agents complete:

Collect `suspected_kantra_limitations` from all rule-writer and test-generator agent returns (merge, deduplicate by `rule_id`). If non-empty, print:

```
[test-gen] <N> suspected kantra limitations (Maven Central: no plain-semver version): <rule_id_1>, ...
```

Carry this merged list forward as `pre_classified_kantra_limitations` — it will be passed to the rule-validator to skip the fix loop for these rules.

Sanitize XML:

```bash
go run ./cmd/sanitize <log_flags> --dir <migration_dir>/tests/data
```

```
[test-gen] Done — <total_groups> groups, <total_files> files (<elapsed>)
```

### 4. Validate (orchestrator-driven loop)

The orchestrator runs tests and uses a sub-agent for LLM-driven repairs on failures.

**4a. Run tests:**

```bash
go run ./cmd/test <log_flags> --rules <migration_dir>/rules --tests <migration_dir>/tests --timeout 5m
```

The CLI runs each test file sequentially (avoids Docker contention) and automatically retries timed-out files once (`--retry-timeouts`, on by default). To run a subset, use `--files` with bare filenames (e.g., `--files data-1.test.yaml,data-2.test.yaml`), resolved relative to `--tests` dir.

Print the result:

```
[validate] <total_passed>/<total_rules> passed (<elapsed>)
```

Or if failures:

```
[validate] <total_passed>/<total_rules> passed — <F> failures: <rule_id_1>, <rule_id_2>, ...
```

If all passed (elapsed shown above), skip to step 5.

**4b. Fix (only if failures):**

```
[fix] Fixing <F> failures: <rule_id_1>, <rule_id_2> ...
```

**Invoke:** `rule-validator`
**Purpose:** Fix test data for failing rules so they pass kantra validation.
**Inputs:**
  - rules_dir: <migration_dir>/rules
  - tests_dir: <migration_dir>/tests
  - failing_rules: {list with rule_id, test_file, error for each failure}
  - pre_classified_kantra_limitations: {merged list from rule-writer and test-generator suspected_kantra_limitations — may be empty}
  - max_iterations: 1
**Parallel:** no
**Expect:**
  - fixed_rules, still_failing, kantra_limitation_rules, iterations_used, fix_details
  - results_by_rule (with `result_type` + `recommended_action`)

The validator agent owns the fix-verify loop — it fixes files, re-runs tests via `cmd/test`, and iterates up to `max_iterations`. Default is 1 iteration; pass up to 3 if the user requests more attempts.

**The orchestrator must NOT run its own fix attempts.** Do not manually edit test files, create stub classes, toggle analysisParams, or retry. All fix work goes through the rule-validator sub-agent. When the validator returns `still_failing`, accept that result and move on.

Print the result:

```
[fix] <fixed_count> fixed, <still_failing_count> still failing, <kantra_limitation_count> kantra limitations (<validate elapsed>)
```

The `<validate elapsed>` is the total time since the validate stage started (step 4a), including test runs and all fix iterations.

If `kantra_limitation_rules` is non-empty, print:

```
[fix] kantra limitations (engine cannot test — rule is correct): <rule_id_1>, ...
```

Group unresolved failures by `result_type` in `results_by_rule` and include grouped counts in the final summary (e.g., `still_failing_timeout: 2`, `still_failing_kantra_limitation: 1`).

If still_failing is non-empty, move on — don't block the pipeline.

### 5. Stamp + Report

Collate pass/fail results from the test run and the fix loop. No need to re-run the full test suite — stamp directly from results. Include verification labels from step 2c:

```bash
go run ./cmd/stamp <log_flags> --rules <migration_dir>/rules --passed <comma-separated passed rule IDs> --failed <comma-separated failed rule IDs> --kantra-limitation <comma-separated kantra limitation rule IDs> --verified <comma-separated verified rule IDs> --not-found <comma-separated not-found rule IDs>
go run ./cmd/report <log_flags> --source <source> --target <target> --output <migration_dir>/report.yaml --rules-total <N> --passed <P> --failed <F> --kantra-limitation <K> --failed-rules <comma-separated>
```

The `--verified` and `--not-found` flags come from the verification mapping built in step 2c (pattern_index → rule_id via construct's pattern_rule_map). These stamp `konveyor.io/source-verified=true/false` labels alongside the test-result labels.

Rules in `kantra_limitation_rules` are stamped with `test-result=kantra-limitation`. They are not counted as passed or failed. The pass rate in the report is computed as `passed / (total - kantra_limitation)` to remain honest — kantra limitations are not failures, but they are also not confirmed passes.

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
| **Verification** | <verified>/<total> source-verified, <not_found> not found, <skipped> skipped (omit row if all skipped) |
| **Kantra limitations** | <K> rules correct but not auto-testable — engine cannot compare non-semver versions (omit row if K=0) |
| **Fix iterations** | <iterations used, 0 if none> |
| **Timing** | total: <total elapsed> — ingest: <elapsed>, extract: <elapsed>, coverage: <elapsed>, test-gen: <elapsed>, validate: <elapsed>, stamp+report: <elapsed> |
| **Output** | `<migration_dir>/patterns.json` (patterns), `<migration_dir>/rules/` (rules), `<migration_dir>/tests/` (tests), `<migration_dir>/report.yaml` (report) |

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
