# Generate Rules Orchestrator Details

This reference contains expanded operational details that are intentionally kept
out of the core `SKILL.md` to reduce prompt size.

## Contents

- Status Line Format
- Parallelism Defaults
- Resume Stage Preconditions
- Rebuild Behavior
- Pipeline Transcript
- Timing
- Sub-agent Protocol

## Status Line Format

Use concise status lines:

```text
[step-name] message
```

Example:

```text
[ingest] Fetching guide from https://...
[ingest] Done — 3876 lines, 77 sections (0m 12s)
[extract] Extracting patterns from 62 sections (3 parallel agents)...
[extract] Done — 52 patterns → 52 rules (4m 35s)
[coverage] No gaps found (1m 20s)
[test-gen] Generating test data for 52 rules...
[test-gen] Done — 12 groups, 39 files (3m 45s)
[validate] 49/52 passed — fixing 3 failures
[fix] 2 fixed, 1 still failing (6m 10s)
[done] 52 rules generated (16m 02s total)
```

## Parallelism Defaults

- Extraction: 1-5 parallel `rule-writer` invocations
- Test generation: 1-5 parallel `test-generator` invocations
- Balance by rule/section count per worker

## Resume Stage Preconditions

These are the canonical artifact checks for `resume_from`:

- `extract` requires `guide.md`
- `coverage` requires `patterns.json`
- `scaffold` requires `rules/`
- `test` requires `rules/` and `tests/manifest.json`
- `report` requires source/target metadata plus pass/fail totals and rule ID lists

If prerequisites are missing, fail fast with a structured error and do not infer
or regenerate stages implicitly unless explicitly instructed.

## Rebuild Behavior

- `force_rebuild=true` means regenerate current stage outputs even if present.
- `force_rebuild=false` means reuse existing artifacts when preconditions hold.

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
- **report** — from report generation start to report written

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
