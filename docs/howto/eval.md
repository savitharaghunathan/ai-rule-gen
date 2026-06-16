# How to Run Evals

Measure the quality of generated migration rules with deterministic checks and an optional LLM judge.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [kantra](https://github.com/konveyor/kantra) (for app coverage analysis)
- Java 21+ (required by kantra's bundled jdtls)

## Quick Start

```bash
# Quality-only (no app needed)
go run ./cmd/eval --rules-dir output/<migration>/rules

# With app coverage analysis
go run ./cmd/eval \
  --rules-dir output/<migration>/rules \
  --app-dir /path/to/sample-app

# Save results for regression tracking
go run ./cmd/eval \
  --rules-dir evals/<migration>/rules \
  --app-dir /path/to/sample-app \
  --migration <migration-name> \
  --save
```

## What It Measures

### Quality Scoring (per rule)

| Check | Points | What it looks for |
|-------|--------|-------------------|
| Message | 1 | Rule has a non-empty migration message |
| Links | 1 | Rule includes documentation links |
| Effort | 1 | Rule has an effort rating |
| Before/after | 1 | Message contains migration guidance (e.g., "replace", "use", "instead of") |
| Guidance depth | 0–3 | How actionable the migration message is |

### App Coverage (requires `--app-dir`)

Runs `kantra analyze` against a real application and cross-references results:

| Metric | Meaning |
|--------|---------|
| **Rules fired** | Rules that matched code in the app |
| **Effective match** | Fired rules / rules whose API is present in the app |
| **In app but unmatched** | API is in the source but kantra didn't match — potential broken rule |
| **Not in app** | App doesn't use the API — rule is correct, just not exercised |
| **Incidents** | Total code locations matched across all fired rules |

### Overlap Detection

Finds rules that fire on the same files, which may indicate redundant or conflicting conditions.

### Specificity Gaps (requires ground truth)

Identifies APIs with non-trivial changes (class renames, method removals) that are only covered by broad package-level rules instead of dedicated fine-grained rules.

### LLM Judge (via eval skill)

Compares the source migration guide against the generated rules. Produces:
- **Missed patterns** — guide actions with no corresponding rule
- **False positives** — rules that would fire incorrectly
- **Quality notes** — rules that work but could be improved

Run via your agent:
```
Read and follow agents/eval/SKILL.md
```

## Saving and Comparing Runs

```bash
# Save a snapshot
go run ./cmd/eval \
  --rules-dir evals/<migration>/rules \
  --migration <migration-name> \
  --save

# Save as baseline
go run ./cmd/eval \
  --rules-dir evals/<migration>/rules \
  --migration <migration-name> \
  --save-baseline

# Compare against baseline
go run ./cmd/eval \
  --rules-dir evals/<migration>/rules \
  --migration <migration-name> \
  --compare evals/<migration>/det_baseline.json
```

Snapshots are saved to `evals/<migration>/runs/<timestamp>.json`.

## Adding a New Migration Eval

1. Create `evals/<source>-to-<target>/`
2. Add `eval_config.yaml`:
   ```yaml
   guide_url: https://example.com/migration-guide
   app_repo: https://github.com/org/sample-app
   source: <source>
   target: <target>
   rules_dir: evals/<source>-to-<target>/rules
   ```
3. Copy or generate rules into `rules/`
4. Optionally generate ground truth for gap analysis:
   ```bash
   # From Maven artifacts (Java, most comprehensive)
   go run ./cmd/ground-truth \
     --old-artifact group:artifact:old-version \
     --new-artifact group:artifact:new-version \
     --output evals/<migration>/ground_truth.yaml

   # From migration guide (any language)
   go run ./cmd/ground-truth \
     --from-guide /path/to/guide.md \
     --output evals/<migration>/ground_truth.yaml
   ```
5. Run the eval to validate

## Example Output

```
======================================================================
EVAL REPORT
======================================================================

## Rules: 94

## Quality (avg 5.6/6)
   Messages:           94/94
   Links:              94/94
   Effort rating:      94/94
   Before/after:       88/94
   Guidance depth avg: 2.6/3

## App Coverage
   Rules fired:      46/94 (49%)
   Effective match:  46/55 (84%)
   Incidents:        68

   In app but unmatched (9 rules):
     - rule-00040 (xpath condition) → pom.xml
     ...

   Not in app (39 rules):
     - rule-00050 (SomeApi)
     ...

======================================================================
```

The report goes to stderr. Structured JSON goes to stdout for programmatic consumption.

## Comparing two rulesets

To diff a generated ruleset against another ruleset (e.g. a hand-authored baseline from [konveyor/rulesets](https://github.com/konveyor/rulesets)), use `cmd/compare`:

```bash
go run ./cmd/compare \
  --a evals/<migration>/rules \
  --b /path/to/baseline/rules \
  --name-a ai --name-b handcrafted \
  --app-dir /path/to/sample-app \
  --out comparison.md
```

Output is a coverage matrix (per-rule condition-key matching, both directions) plus a side-by-side `kantra analyze` diff on the same app. Worked examples in `evals/comparisons/`.

## See Also

- [Eval examples and directory layout](../../evals/README.md)
- [Eval skill](../../agents/eval/SKILL.md) — full eval with LLM judge
