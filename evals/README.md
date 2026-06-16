# Eval Examples

This directory contains migration rule evaluation examples. Each subdirectory is a self-contained eval for one migration, with rules, ground truth, and run snapshots that track quality over time.

The eval skill (`agents/eval/SKILL.md`) uses these artifacts to validate rule quality through deterministic checks (quality scoring, app coverage, overlap detection) and LLM judge review (per-rule condition/message accuracy).

For a step-by-step guide on running evals, see [docs/howto/eval.md](../docs/howto/eval.md).

## Directory Layout

```
evals/
├── <migration-name>/
│   ├── eval_config.yaml        # Migration metadata
│   ├── ground_truth.yaml       # Authoritative API change list
│   ├── rules/                  # Generated Konveyor analyzer rules
│   │   ├── ruleset.yaml
│   │   ├── core.yaml
│   │   └── ...
│   └── runs/                   # Deterministic eval snapshots
│       ├── 20260528-214724.json
│       └── ...
```

## File Formats

### eval_config.yaml

Migration metadata used by the eval skill and CI.

```yaml
guide_url: https://example.com/migration-guide
app_repo: https://github.com/org/sample-app    # optional sample app for coverage testing
source: spring-boot3
target: spring-boot4
rules_dir: evals/spring-boot3-to-spring-boot4/rules
```

### ground_truth.yaml (optional)

Authoritative list of API changes for gap analysis. When present, the eval checks which ground truth entries lack dedicated rules (specificity gaps). When absent, the eval still runs quality scoring, app coverage, and overlap detection — it just skips gap analysis.

```yaml
- old_api: org.springframework.boot.test.mock.mockito.MockBean
  new_api: org.springframework.test.context.bean.override.mockito.MockitoBean
  action_type: package_change
  severity: high
  guide_section: MockBean and SpyBean Removal
  reviewed_by: japicmp
  reviewed_date: "2026-05-28"
```

**Generation methods:**

**1. japicmp (Java, most comprehensive):** Compares old and new JARs to enumerate every API-level breaking change — removed classes, moved packages, changed method signatures. Produces the most complete ground truth but requires knowing the Maven coordinates.

```bash
# Single module
go run ./cmd/ground-truth \
  --old-artifact org.springframework.boot:spring-boot:3.5.0 \
  --new-artifact org.springframework.boot:spring-boot:4.0.0 \
  --output evals/<migration>/ground_truth.yaml

# Multiple modules — merge into one file
go run ./cmd/ground-truth \
  --old-artifact org.springframework.boot:spring-boot-autoconfigure:3.5.0 \
  --new-artifact org.springframework.boot:spring-boot-autoconfigure:4.0.0 \
  --merge evals/<migration>/ground_truth.yaml \
  --output evals/<migration>/ground_truth.yaml
```

**2. Guide extraction (any language):** Regex-extracts fully qualified names from migration guide markdown. Lower yield than japicmp (guides use short names) but works for any language.

```bash
go run ./cmd/ground-truth \
  --from-guide /path/to/guide.md \
  --guide-url https://example.com/migration-guide \
  --output evals/<migration>/ground_truth.yaml
```

**3. None:** Skip ground truth entirely. The eval still runs quality scoring, app coverage, and overlap detection — it just skips gap analysis.

### rules/

Konveyor analyzer rules in YAML, grouped by feature area (testing, web, security, etc.). Each file contains one or more rules with `ruleID`, `when` condition, `message`, `category`, `effort`, and `links`.

Validated with: `go run ./cmd/validate -rules evals/<migration>/rules/`

### runs/\<timestamp\>.json

Deterministic eval snapshot from one execution. Lightweight (~12 fields) for regression tracking:

```json
{
  "schema_version": 1,
  "timestamp": "2026-05-28T21:50:43Z",
  "migration": "spring-boot3-to-spring-boot4",
  "rule_count": 94,
  "quality_avg": 5.55,
  "effective_coverage_pct": 84.0,
  "overlap_conflict_count": 176,
  "specificity_gap_count": 0
}
```

## Running an Eval

```bash
# 1. Validate rule syntax
go run ./cmd/validate -rules evals/<migration>/rules/

# 2. Deterministic eval (quality, coverage, overlaps)
#    --ground-truth and --app-dir are optional
go run ./cmd/eval \
  --rules-dir evals/<migration>/rules/ \
  --ground-truth evals/<migration>/ground_truth.yaml \
  --app-dir /path/to/sample-app \
  --migration <migration-name> \
  --save

# 3. Minimal eval (no ground truth, no sample app)
go run ./cmd/eval \
  --rules-dir evals/<migration>/rules/ \
  --migration <migration-name> \
  --save

# 4. Full eval (deterministic + LLM judge + auto-fix)
# Use the eval skill: agents/eval/SKILL.md
```

Without ground truth, the eval reports quality scores, overlap conflicts, and (if `--app-dir` is provided) app coverage. Gap analysis and specificity checks require ground truth.

To diff a generated ruleset against another ruleset (e.g. a hand-authored baseline), use `cmd/compare` — coverage matrix plus side-by-side kantra diff. See [docs/howto/eval.md](../docs/howto/eval.md#comparing-two-rulesets) and `evals/comparisons/`.

## Adding a New Migration

1. Create `evals/<source>-to-<target>/`
2. Add `eval_config.yaml` with guide URL and source/target
3. Copy or generate rules into `rules/`
4. Run the eval skill to validate and iterate

Optionally, generate ground truth for gap analysis (step 3 above). Without it, the eval still provides quality scores, coverage, and overlap detection. See the ground truth section above for generation methods.

## Current Examples

| Migration | Rules | Ground Truth | Source |
|-----------|-------|-------------|--------|
| `httpclient4-to-httpclient5` | 42 | 124 entries | japicmp |
| `httpclient4-to-httpclient5-fromguide` | — | 172 entries | guide extraction |
| `spring-boot3-to-spring-boot4` | 94 | 999 entries | japicmp (3 modules) |
