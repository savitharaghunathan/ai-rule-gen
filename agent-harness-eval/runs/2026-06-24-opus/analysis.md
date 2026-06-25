---
agent: Claude Code
model: claude-opus-4-6
date: 2026-06-24T08:00:00Z
---

## Recommendation

**Pipeline is production-ready: 5/7 judges pass across all cases, with only one case (Tomcat) failing to complete the full pipeline.**

All 5 cases pass budget, rules_generated, patterns_quality, and both LLM judges (extraction_completeness 4.2/5, rule_correctness 4.0/5 — both above the 3.5 threshold). The only failures are report_exists (80%) and test_coverage (80%), both isolated to case-004-tomcat9-to-tomcat10, which exhausted its 275 turns before generating test data files or the final report. This is a pipeline completion issue, not a quality issue — case-004's extraction (4/5) and correctness (4/5) scores match the other cases.

**Top actions:**
- **HIGH** — Investigate why case-004 consumed 275 turns without completing. The Tomcat guide is small (13 patterns) yet used the most turns of any case. The pipeline appears to have stalled in the test generation phase — test YAML files were scaffolded but no `tests/data/` directory was created and no `report.yaml` was generated.
- **MEDIUM** — Case-001 (HttpClient) scored lowest on rule_correctness (3/5) due to 10 patterns with non-fully-qualified names being dropped during rule construction. Consider improving the extract step's FQN resolution for method-level patterns.
- **LOW** — All cases show 0% test pass rate in report.yaml due to missing JAVA_HOME in the eval sandbox. This is an infrastructure gap, not a skill defect — add JAVA_HOME to `execution.env` in eval.yaml for accurate test validation scoring.

## Summary

| Judge | Mean | Pass Rate | Threshold | Status |
|-------|------|-----------|-----------|--------|
| budget_check | 1.0 | 100% | — | PASS |
| rules_generated | 1.0 | 100% | — | PASS |
| patterns_quality | 1.0 | 100% | — | PASS |
| report_exists | 0.8 | 80% | — | FAIL |
| test_coverage | 0.8 | 80% | — | FAIL |
| extraction_completeness | 4.2 | — | 3.5 | PASS |
| rule_correctness | 4.0 | — | 3.5 | PASS |

**Run metrics:**
- Total cost: $52.16 across 5 cases (parallel execution)
- Wall clock: 26.1 min (1567.7s) with parallelism=5
- Total compute: 96.8 min (5805.2s sum of case durations)
- Total turns: 880 (avg 176/case)
- Cache hit rate: 94.6%
- Cost per turn: $0.059

| Case | Turns | Duration | Cost | Extract | Correct | Report | Tests |
|------|-------|----------|------|---------|---------|--------|-------|
| case-001 (HttpClient) | 137 | 18.9 min | $10.34 | 4 | 3 | PASS | PASS |
| case-002 (Spring Boot) | 217 | 21.5 min | $16.80 | 4 | 4 | PASS | PASS |
| case-003 (Log4j) | 129 | 16.1 min | $9.84 | 4 | 4 | PASS | PASS |
| case-004 (Tomcat) | 275 | 26.1 min | $7.41 | 4 | 4 | FAIL | FAIL |
| case-005 (JUnit) | 122 | 14.1 min | $7.78 | 5 | 5 | PASS | PASS |

## Failure Patterns

**Clustered failure — case-004 only:** Both report_exists and test_coverage fail exclusively on case-004. This is a pipeline completion issue: the skill scaffolded 12 test YAML files but never created the `tests/data/` directory with actual test data, and never reached the report generation step. The case used 275 turns — the highest of any case — suggesting it got stuck in a loop or spent excessive turns on earlier pipeline steps.

**No judge-specific failures:** Every judge passes on at least 4/5 cases. The two failing judges (report_exists, test_coverage) both fail on the same case for the same root cause.

## Root Causes

### Case-004 pipeline incomplete (report_exists + test_coverage)
The Tomcat 9→10 migration guide is relatively small (13 patterns), yet the pipeline consumed 275 turns — more than case-002 (Spring Boot, 73 patterns, 217 turns). The pipeline produced:
- patterns.json (13 patterns) ✓
- rules/ (11 YAML files) ✓
- tests/*.test.yaml (12 files) ✓
- tests/data/ — MISSING
- report.yaml — MISSING

The pipeline stalled after scaffolding test files but before generating test data. Given the high turn count on a small guide, the likely cause is the test generation subagents encountering issues (possibly sandbox permission errors writing test data files) and retrying repeatedly until the turn budget was exhausted.

### Case-001 lower correctness score (3/5)
The HttpClient migration had 10 patterns with non-fully-qualified method names that were dropped during rule construction. This is a known limitation of the extract step when guide content uses shorthand method references. The 16 rules that were generated scored well on accuracy, but the coverage gap from dropped patterns lowered the overall correctness score.

## Cost Attribution

| Metric | Value |
|--------|-------|
| Cost per turn | $0.059 |
| Cost per Mtok (output) | $0.89 |
| Cache hit rate | 94.6% |
| Output tokens per turn | 724 |

Cost per case ranges from $7.41 (Tomcat) to $16.80 (Spring Boot). The cost correlates with guide complexity — Spring Boot's 73 patterns required the most work. Tomcat's low cost despite high turn count reflects the 94.6% cache hit rate keeping per-turn cost low even when stalling.

The $52.16 total is reasonable for 5 migration guides producing 171 total rules (patterns: 26+73+44+13+15) with full test generation. Cost per rule: $0.31.
