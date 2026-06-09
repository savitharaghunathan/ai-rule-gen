# Eval Framework Improvements Design

**Date:** 2026-05-27
**Status:** Approved (rev 5 — addressed review rounds 1-4)
**Branch:** eval-framework

## Background

The eval framework has two layers: a deterministic eval (`cmd/eval`, `internal/eval/`) that scores rule quality and measures app coverage, and an LLM judge (`agents/eval/SKILL.md`) that reviews each rule for condition accuracy, message correctness, and gap analysis. Both are functional but the framework has strategic gaps identified by two independent code reviews:

1. No feedback loop between eval output and rule generation
2. No regression tracking across runs
3. Gap analysis depends on uncalibrated LLM extraction
4. Quality score is too coarse (4/4 average on real rules)
5. No inter-rule consistency checks
6. Only Java has a language reference for the judge
7. No CI integration

## Key architectural constraint: deterministic vs judge boundary

The deterministic eval (`cmd/eval`) and LLM judge (`eval` skill) are separate pipelines with different properties:

| Property | Deterministic | Judge |
|----------|--------------|-------|
| Reproducible | Yes | No |
| CI-runnable | Yes | No (requires API keys, expensive) |
| Metrics | rule_count, coverage, quality_avg, guidance_depth, overlap | precision_issues, coherence_issues, gaps |
| Artifacts | `det_baseline.json`, `runs/<ts>.json` | `judge_findings.json` |
| Gating | CI gates on deterministic metrics only | Judge findings are advisory, human-reviewed |

All artifacts, CLI flags, and CI gates respect this boundary. `--compare` operates on deterministic baselines only. Judge findings are never mixed into deterministic artifacts.

## Design

### Phase 1: Close the feedback loop

#### 1a. Structured eval output (findings.json)

The eval produces a machine-readable `findings.json` alongside its markdown report.

**Canonical schema:**

```json
{
  "schema_version": 1,
  "language": "java",
  "timestamp": "2026-05-27T15:30:00Z",
  "precision_issues": [
    {
      "rule_id": "string, required — ruleID from the YAML",
      "pattern": "string, required — the condition pattern",
      "issue": "string, required — one-line description",
      "severity": "warn | fail",
      "fix_type": "add_scoping | qualify_fqn | narrow_pattern",
      "fix": {
        "scope_package": "string, optional — package pattern for as/from scoping",
        "qualified_pattern": "string, optional — full FQN to replace current pattern"
      }
    }
  ],
  "coherence_issues": [
    {
      "rule_id": "string, required",
      "issue": "string, required — one-line description",
      "severity": "warn | fail",
      "fix_type": "narrow_condition | broaden_message | split_rule",
      "condition_scope": "broad | narrow | wrong",
      "message_scope": "broad | narrow | wrong"
    }
  ],
  "cross_rule_issues": [
    {
      "rule_ids": ["string array, required — the conflicting rule IDs"],
      "issue": "string, required",
      "severity": "warn | fail",
      "fix_type": "deduplicate | reorder | reconcile_messages"
    }
  ],
  "gaps": [
    {
      "source_fqn": "string, required — old API identifier",
      "location": "string, optional — condition location type",
      "action_type": "string, required — class_rename | method_rename | method_removal | package_change | config_change | signature_change | behavioral_change",
      "target_api": "string, required — replacement API",
      "message_sketch": "string, required — draft message for the rule",
      "severity": "high | medium | low",
      "guide_section": "string, required — section title from guide"
    }
  ]
}
```

**Required fields:** Every field marked "required" above must be present. The auto-fix orchestrator parses `fix_type` to dispatch the correct fix strategy. `severity` uses a two-level enum (`warn`/`fail` for issues, `high`/`medium`/`low` for gaps) — no numeric scores.

**`action_type` enum:** `class_rename`, `method_rename`, `method_removal`, `package_change`, `config_change`, `signature_change`, `behavioral_change`. Extensible — unknown values are treated as `behavioral_change` but logged as warnings: `"unknown action_type '<value>' in gap for <source_fqn>, treating as behavioral_change"`. This makes novel types visible during development rather than silently swallowed. The warning appears in the eval markdown report under a "Schema warnings" section (only if warnings exist).

All structured artifacts include `schema_version: 1` for forward compatibility.

**Implementation:** Add an output format instruction to SKILL.md Step 6 requesting JSON output written to a run-scoped path (`/tmp/eval-<timestamp>/findings.json`). The path is created at the start of the eval run and passed between steps. The eval writes both the markdown report (to stdout/conversation) and the JSON (to file).

#### 1b. Auto-fix loop

After producing findings.json, the eval can orchestrate fixes. This loop is optional — triggered by explicit user instruction. The eval can still produce a report-only output.

**Patch-first workflow:**
1. Mirror the full current ruleset into a working directory (`/tmp/eval-<timestamp>/proposed-rules/`) — full copy, not just modified files
2. Apply fixes to the mirrored copy (modify existing files, add new files for gaps)
3. Run `cmd/validate` on the full mirrored ruleset
4. Run deterministic eval on the full mirrored ruleset — this is a valid comparison because it's a complete rule directory
5. Present the diff (mirrored vs original) to the human for review before applying

**Fix steps:**
1. Mirror: `cp -r <rules_dir> /tmp/eval-<timestamp>/proposed-rules/`
2. Fix precision issues: invoke rule-writer with scoping instructions, writing to the mirrored copy
3. Generate gap rules: feed gap entries as patterns to rule-writer, writing to the mirrored copy
4. Validate: `go run ./cmd/validate --rules-dir <proposed-rules>`
5. Re-run deterministic eval on proposed-rules. Compare against pre-fix state (not baseline — see below).
6. If issues remain AND no metric regressed vs pre-fix, run one more fix iteration (max 2 total)

**Per-iteration verification:**
After each fix iteration, re-run the judge (Steps 4/4.5) on modified rules to produce an updated findings.json. Compare against prior iteration findings to compute deltas. This is the only place the judge runs inside the loop — the deterministic eval gates on hard metrics, the judge gates on semantic regression.

**Regression comparator:** The auto-fix loop compares each iteration against the **pre-fix state** (the eval result from before fixes started), not the baseline. This prevents local degradation when the current branch is above baseline. The baseline is for CI gating; the pre-fix snapshot is for fix-loop guardrails.

**Stop conditions — do NOT continue if:**
- Any deterministic metric (coverage, quality_avg) drops below pre-fix state
- `cmd/validate` fails
- Judge coherence issues regress: compare by fingerprint (`rule_id + fix_type`). Regression = any new `fail`-severity issue not in prior iteration, OR total `fail` count increases
- Warn budget exceeded: total `warn`-severity issue count across all categories grows by more than 50% vs pre-fix state (minimum threshold: 3 new warns). This prevents fixes from trading `fail` issues for an unbounded number of `warn` issues. Example: pre-fix has 4 warns → budget allows up to 6; pre-fix has 0 warns → budget allows up to 3.

**Apply step:** When the human approves:
1. Create a backup: `mv <rules_dir> <rules_dir>.bak`
2. Move proposed rules into place: `mv proposed-rules <rules_dir>`
3. Remove backup: `rm -rf <rules_dir>.bak`

If step 2 fails (cross-filesystem), fall back to `cp -r proposed-rules <rules_dir>.new && mv <rules_dir> <rules_dir>.bak && mv <rules_dir>.new <rules_dir>`. The `.bak` directory is kept until the human confirms; rollback is `rm -rf <rules_dir> && mv <rules_dir>.bak <rules_dir>`.

**Scope restriction:** The auto-fix loop only modifies files under `rules/` in the output directory. It never touches `SKILL.md`, test files, or Go code.

**Implementation:** Add a "Step 7: Auto-fix (optional)" to SKILL.md that reads findings.json, invokes rule-writer for fixes, validates, and re-runs `cmd/eval`.

#### 1c. Regression tracking

Two storage layers, strictly deterministic:

**Baseline** (`evals/<migration>/det_baseline.json`): Checked into git. Contains the deterministic metrics snapshot that CI gates against. Updated manually when a new rule set is accepted. Includes `schema_version: 1`.

**Run history** (`evals/<migration>/runs/<timestamp>.json`): Local development history. Each `cmd/eval` run optionally writes its result here.

**Judge findings** are stored separately: `evals/<migration>/judge_findings/<timestamp>.json`. These are advisory — never used for CI gating or `--compare`.

**CLI changes:**
- `cmd/eval --save` writes result to `evals/<migration>/runs/<timestamp>.json`
- `cmd/eval --compare <path>` diffs current run against a previous result and reports deltas
- `cmd/eval --migration <name>` resolves the eval case directory (alternative to inferring from `--rules-dir`)

**Deterministic metrics (CI-gatable):**
| Metric | Type | Regression signal |
|--------|------|-------------------|
| rule_count | int | Drop = rules lost |
| effective_coverage_pct | int | Drop >5% = coverage regression |
| quality_avg | float | Drop = quality regression |
| guidance_depth_avg | float | Drop = message quality regression |
| overlap_conflict_count | int | Report only — not a CI gate until semantics are provider-aware. Value is `null` when `app_dir` is not provided (incident overlap requires app data). Normalized as conflicts-per-fired-rule for cross-run comparability. |

**Judge metrics (advisory only, not in `--compare`):**
| Metric | Type | Signal |
|--------|------|--------|
| precision_issue_count | int | More broad patterns |
| coherence_issue_count | int | More mismatched rules |
| gap_count | int | More missing coverage |

**Schema compatibility:** `--compare` requires matching `schema_version` between current output and baseline. On mismatch, fail with `incompatible baseline version (got N, need M) — regenerate baseline with current eval`. No auto-conversion — baselines are small and cheap to regenerate.

**Comparison output (deterministic only):**
```
REGRESSION CHECK: httpclient4-to-httpclient5
  effective_coverage: 95% -> 93%  REGRESSED (-2%)
  quality_avg:        4.0 -> 4.0  OK
  guidance_depth_avg: 2.8 -> 2.9  OK
  overlap_conflicts:  0 -> 0      OK
  Verdict: REVIEW (coverage dropped)
```

#### 1d. Baseline governance

**Update policy:** A baseline update requires:
1. A passing deterministic eval run on the new ruleset
2. A manual judge run confirming no new `fail`-severity findings (judge report attached to the PR)
3. PR review by at least one team member — the diff of `det_baseline.json` makes metric changes visible

**Who updates:** Any team member, via a PR that includes the updated `det_baseline.json` and the judge report that validates the semantic quality. The PR description must state which metrics changed and why.

**Preventing semantic regression in PR flow:** The deterministic eval runs in CI but the judge does not (expensive, non-deterministic). To prevent semantic quality from regressing silently between judge runs:
- Baseline update PRs must include a judge report (enforced by PR template checklist, not automation)
- For migrations with a `ground_truth.yaml`, deterministic gap coverage (rules vs ground truth entries) is included in `det_baseline.json` and CI-gated. This gives a deterministic proxy for semantic coverage.
- Teams are encouraged to run the judge manually before merging large rule changes. The eval README documents this as a best practice, not a hard gate.

#### 1e. Per-migration CI thresholds

The global 5% coverage threshold is a default, not a mandate. Each migration can override thresholds in its `eval_config.yaml`:

```yaml
ci_thresholds:
  effective_coverage_drop_pct: 5   # default
  quality_avg_drop: 0.0            # default: any drop fails
```

If `ci_thresholds` is absent, the global defaults apply. This lets small migrations (10 rules) tolerate a larger absolute swing than large ones (200 rules) by adjusting the percentage, and lets mature migrations tighten their gates.

### Phase 2: Eval depth

#### 2a. Ground-truth migration map

For each eval case, create `evals/<migration>/ground_truth.yaml` — a hand-curated list of every actionable migration pattern from the guide.

**Bootstrap process:**
1. Run the eval to extract its migration map as a starting point
2. Export as `ground_truth.yaml`
3. Human independently reviews against the original guide: add missed patterns, remove false ones, verify action types
4. Each entry includes provenance — a source quote from the guide that justifies the entry

The LLM-generated map is a convenience, not the authority. The human review against the original guide is what breaks the circularity. Entries without a verifiable guide quote should be removed.

**Schema:**
```yaml
schema_version: 1
guide_url: https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide
guide_version: "2026-05-27"  # date or version of the guide when ground truth was created
entries:
  - old_api: org.apache.http.client.methods.HttpGet
    new_api: ClassicRequestBuilder.get()
    action_type: class_rename
    severity: medium
    guide_section: "Migration to classic APIs"
    source_quote: "HttpGet ... can be replaced with ClassicRequestBuilder"
    reviewed_by: human
    reviewed_date: "2026-05-27"
```

**Guide version pinning:** The `guide_version` field records when the ground truth was last validated against the upstream guide. If the guide URL content changes (new library version, updated recommendations), the ground truth must be re-reviewed. The eval logs a warning if `guide_version` is older than 90 days: `"ground truth last reviewed <date> — consider re-validating against current guide"`. This is advisory, not blocking.

**Usage in eval:** Step 5 (gap analysis) compares rules against ground truth instead of its own extraction. Report includes extraction metrics:
- Extraction recall: X/Y ground truth entries found by LLM
- Extraction precision: X/Y LLM entries verified in ground truth

This measures the judge's own reliability and anchors gap analysis to verified data.

#### 2b. Graduated quality scoring

Replace the binary `hasGuidance` with a multi-tier `GuidanceDepth`:

| Tier | Score | Criteria |
|------|-------|----------|
| None | 0 | No message or empty/whitespace |
| Exists | 1 | Message exists but no actionable content |
| Actionable | 2 | Contains "replace", "instead of", "renamed to", backtick code |
| Specific | 3 | Names a specific replacement API (FQN or code-fenced identifier different from the condition pattern) |

The existing 4-dimension score changes: `message` (0-1), `links` (0-1), `effort` (0-1), `guidanceDepth` (0-3). MaxScore becomes 6.

**Detection logic for tier 3:** Extract all backtick-quoted strings from the message. Extract the condition pattern's last FQN segment (e.g., `getStatusLine` from `org.apache.http.HttpResponse.getStatusLine`). If any backtick string is a valid identifier AND does not match the condition pattern's last segment, the message names a specific replacement. Example: condition pattern `getStatusLine`, message contains `` `getCode()` `` → tier 3.

**Known limitations:** This heuristic is approximate. It misses valid guidance without backticks and can false-positive on incidental identifiers. It is an advisory signal for quality tracking, not a gate. The improvement over the current binary check ("contains the word 'before'") is significant even if imperfect.

**Implementation:** Extend `quality.go` `ScoreRule`:
- Add `GuidanceDepth` field (0-3) to `RuleDetail`
- Extract backtick-quoted strings with a regex
- Compare against condition pattern to determine tier
- Update `QualitySummary` to include `GuidanceDepthAvg`
- Existing `HasGuidance` bool maps to `GuidanceDepth >= 2`

#### 2c. Overlap detection

New module `internal/eval/overlap.go`:

**Pattern overlap** (always available, approximate):
- For each rule pair, check if one condition pattern is a substring/prefix of the other
- Flag pairs where rule A fires on a superset of rule B's matches
- Check for rules with identical `location` + overlapping patterns

**Limitation:** Substring/prefix matching is syntactic, not semantic. It will miss overlap between wildcard patterns and produce false positives for unrelated FQNs that share a suffix. This is a v1 heuristic — provider-aware and location-aware overlap semantics can be added later. Results are advisory, not used for gating.

**Incident overlap** (when `app_dir` provided):
- Using kantra violation data, find rules that fire on the same file
- For same-file pairs, check message action types: complementary (different actions) or conflicting (same API, different advice)

**Output:** `overlaps` section in EvalResult JSON:
```json
{
  "overlaps": [
    {
      "rule_a": "00020",
      "rule_b": "00130",
      "type": "complementary",
      "shared_files": ["WeatherService.java"],
      "reason": "package-level and method-level rules for same migration"
    }
  ]
}
```

#### 2d. Cross-rule coherence prompt (SKILL.md Step 4.5)

After per-rule review (Step 4), add a new step:

> **Step 4.5: Cross-rule coherence check**
>
> Read all rule messages together. Check:
> 1. Do any two rules give contradictory replacement advice?
> 2. If a developer sees all incidents in one file, does the combined guidance make a coherent migration path?
> 3. Are there implicit ordering dependencies (e.g., "rename package imports before changing method calls")?
>
> Report cross-rule issues in the findings under a separate "cross-rule" category.

### Phase 3: Production readiness

#### 3a. Language references

Create `agents/eval/references/languages/go.md`:
- Module path patterns (`github.com/old/pkg.Function`)
- No location types — Go provider uses pattern matching only
- Function vs method vs type distinction
- Calibration examples: pass, precision issue, gap

Create `agents/eval/references/languages/nodejs.md`:
- Module specifier patterns (`require('express')`, `import ... from 'express'`)
- No location types — Node.js provider uses pattern matching only
- CommonJS vs ESM detection differences
- Named vs default export patterns
- Calibration examples: pass, precision issue, gap

#### 3b. CI workflow

GitHub Actions workflow: `.github/workflows/eval.yml`

**Triggers:** PRs touching `agents/eval/`, `internal/eval/`, `agents/rule-writer/`, or `cmd/eval/`.

**Steps:**
1. Install Go, install kantra
2. Clone sample app from `eval_config.yaml` `app_repo` at pinned `app_commit` SHA
3. Run `go run ./cmd/eval --rules-dir ... --app-dir ... --compare evals/<migration>/det_baseline.json`
4. Hard fail if (deterministic metrics only, thresholds from `eval_config.yaml` `ci_thresholds` or global defaults):
   - Effective coverage drops beyond threshold (default: >5%)
   - Quality average drops beyond threshold (default: any drop)
   Soft warn (reported in PR comment, does not block merge):
   - `guidance_depth_avg` drops (heuristic — noisy until tier-3 detection stabilizes)
   - `overlap_conflict_count` increases
5. Post summary as PR comment

The LLM judge stays manual — expensive, non-deterministic, and requires API keys. Only the deterministic eval runs in CI. Judge-derived metrics (precision_issues, coherence_issues, gap_count) are never CI gates.

**eval_config.yaml changes:**
```yaml
guide_url: https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide
app_repo: https://github.com/savitharaghunathan/httpclient-migration
app_commit: abc1234  # pinned SHA for reproducibility
source: httpclient4
target: httpclient5
```

**Backward compatibility:** If `app_commit` is missing from eval_config.yaml, CI clones the default branch and emits a warning: `"app_commit not pinned — results may vary across runs"`. This avoids breaking existing configs. Migration: add `app_commit` to each eval_config.yaml and re-run to establish the pinned baseline.

**Multi-migration CI:** The workflow iterates all `evals/*/eval_config.yaml` cases that have a `det_baseline.json`. Cases without a baseline are skipped with a warning. Each case runs independently; the workflow fails if ANY case fails its gates. This scales automatically as new eval cases are added — just create the eval_config.yaml and baseline.

## Implementation Order

| Order | Item | Depends on | Effort | Files |
|-------|------|-----------|--------|-------|
| 1 | Graduated quality scoring (2b) | — | Low | `quality.go`, `types.go`, `quality_test.go` |
| 2 | Structured findings.json (1a) | — | Low | `agents/eval/SKILL.md` |
| 3 | Regression tracking (1c) | — | Medium | `cmd/eval/main.go`, new `internal/eval/compare.go` |
| 4 | Per-migration CI thresholds (1e) | 1c | Low | `eval_config.yaml`, `internal/eval/compare.go` |
| 5 | Cross-rule coherence prompt (2d) | — | Low | `agents/eval/SKILL.md` |
| 6 | Ground-truth migration map (2a) | 1a | Medium | `evals/.../ground_truth.yaml`, `agents/eval/SKILL.md` |
| 7 | Overlap detection (2c) | — | Medium | new `internal/eval/overlap.go`, `eval.go`, `types.go` |
| 8 | Go + Node.js references (3a) | — | Low | new `references/languages/go.md`, `nodejs.md` |
| 9 | Auto-fix loop (1b) | 1a, 2b | Medium | `agents/eval/SKILL.md` |
| 10 | CI workflow (3b) | 1c, 1d, 1e | Medium | `.github/workflows/eval.yml` |

Items 1-5 are quick wins that can be done in one session. Baseline governance (1d) is a process/documentation item addressed inline in the CI and eval README — no separate implementation step. Item 9 (auto-fix loop) comes late because it needs structured output and quality scoring to be solid first. Item 10 (CI) needs regression tracking, baseline governance, and per-migration thresholds in place first.
