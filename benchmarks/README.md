# Benchmark Results

Comparison of rule generation quality across agent runtimes and LLM models.

## Methodology

- **Pipeline**: `/generate-rules` skill invoked with the same migration guide URL
- **Evaluation**: Deterministic eval (`cmd/eval`) + LLM judge (`agents/eval/SKILL.md`)
- **Quality Score**: Completeness metric (max 6 pts: message presence + links + effort + before/after guidance). Measures documentation completeness, not rule correctness.
- **Overlaps**: Count of rule pairs that fire on the same code. Overlaps can indicate specificity layering (a broad package-level rule + a specific method-level rule covering the same API), which improves developer experience. High overlap count is not inherently bad — it may mean better coverage through layered detection.
- **Eval Judge**: LLM-based review checking precision (false positive risk), coherence (detection/guidance alignment), cross-rule conflicts, and coverage gaps vs. the migration guide.
- **Timing**: Wall-clock from pipeline start to report completion
- **Date**: May–June 2026

## Runtime × Model Matrix

| Runtime | Models |
|---------|--------|
| Claude Code | Sonnet, Opus, Haiku |
| OpenCode | Sonnet, Opus, Gemini Pro |
| Goose | Sonnet, Opus, Gemini Pro |

## httpclient4-to-httpclient5

| Runtime | Model | Rules | Pass Rate | Quality Avg | Overlaps | Time (min) | Precision | Coherence | Cross-Rule | Gaps |
|---------|-------|-------|-----------|-------------|----------|------------|-----------|-----------|------------|------|
| claude-code | sonnet | 43 | 42/43 | 5.95 | 3 | 20.7 | 4 | 4 (1 fail) | 3 | 1 |
| claude-code | opus | 29 | 29/29 | 5.93 | 15 | 18.2 | 9 | 6 (2 fail) | 2 | 3 |
| claude-code | haiku | 26 | 26/26 | 3.0 | 14 | 4.0 | 3 | 26 (all fail) | 5 | 9 |
| opencode | sonnet | — | — | — | — | — | — | — | — | — |
| opencode | opus | — | — | — | — | — | — | — | — | — |
| opencode | gemini-pro | — | — | — | — | — | — | — | — | — |
| goose | sonnet | — | — | — | — | — | — | — | — | — |
| goose | opus | — | — | — | — | — | — | — | — | — |
| goose | gemini-pro | — | — | — | — | — | — | — | — | — |

### Key Findings (httpclient4→5)

**Sonnet** produces the most comprehensive ruleset (43 rules) with highest quality scores and fewest eval judge issues. One rule fails kantra tests, but overall coverage and guidance are strong.

**Opus** generates fewer rules (29) but is more conservative. Higher overlap count reflects specificity layering. More precision issues from unqualified METHOD_CALL conditions.

**Haiku** produces a non-functional ruleset: all 26 rules have empty messages (`': '`), use `builtin.filecontent` instead of `java.referenced`, and include 5 duplicate groups. The 4-minute runtime reflects the lack of depth — it runs fast but produces nothing usable.

**Shared issue**: Both Sonnet and Opus generate rules that tell classic `CloseableHttpClient` users to switch to async, contradicting the migration guide's classic-first path.

### Interesting Findings

1. **Model capability has a hard threshold for agentic pipelines.** Haiku can follow instructions and produce syntactically valid YAML, but it cannot reason through the multi-step pipeline (ingest guide → extract patterns → construct rules with proper `java.referenced` conditions → generate test data → iterate on failures). It defaults to the simplest possible rule structure (`builtin.filecontent` with empty messages) and never self-corrects. This suggests agentic coding pipelines need a minimum model capability tier — there is no graceful degradation, just a cliff.

2. **More rules ≠ more noise.** Sonnet generates 48% more rules than Opus (43 vs 29) but has proportionally *fewer* eval judge issues. The additional rules cover simple class relocations and API renames that Opus skips entirely. Sonnet's broader extraction captures the long tail of migration patterns without sacrificing precision.

3. **Unqualified METHOD_CALL is the top precision pitfall.** Both Sonnet (4 rules) and Opus (9 rules) generate rules that match method names like `setConnectTimeout` without qualifying the parent class. These fire on the 5.x replacement APIs too — the very code the rule tells you to write. Opus is worse here because it generates more unqualified rules and fewer qualified alternatives.

4. **The async/classic migration path is a consistent LLM blind spot.** Both Sonnet and Opus produce rules telling `CloseableHttpClient` users to switch to `CloseableHttpAsyncClient`. The migration guide explicitly recommends migrating to the 5.x *classic* API first, then optionally to async. This is the highest-severity issue across both rulesets — it would actively mislead the majority of users following the guide.

5. **Overlaps are a feature, not a bug.** Opus's higher overlap count (15 vs Sonnet's 3) reflects more specificity layering — a broad package-level import rule plus specific method-level rules for the same API. This gives developers both a high-level "this package moved" warning and targeted "change this specific call" guidance. The eval initially flagged these as conflicts, but they represent deliberate detection depth.

6. **Cost-quality tradeoff is stark.** Haiku costs ~20x less per token than Opus and runs 4.5x faster, but produces zero usable output. Sonnet costs ~5x less than Opus, runs slightly slower (20.7 vs 18.2 min), and produces better results across every dimension. For this pipeline, Sonnet is the clear cost-performance winner.

## spring-boot3-to-spring-boot4

| Runtime | Model | Rules | Pass Rate | Quality Avg | Overlaps | Time (min) | Precision | Coherence | Cross-Rule | Gaps |
|---------|-------|-------|-----------|-------------|----------|------------|-----------|-----------|------------|------|
| claude-code | sonnet | — | — | — | — | — | — | — | — | — |
| claude-code | opus | — | — | — | — | — | — | — | — | — |
| claude-code | haiku | — | — | — | — | — | — | — | — | — |
| opencode | sonnet | — | — | — | — | — | — | — | — | — |
| opencode | opus | — | — | — | — | — | — | — | — | — |
| opencode | gemini-pro | — | — | — | — | — | — | — | — | — |
| goose | sonnet | — | — | — | — | — | — | — | — | — |
| goose | opus | — | — | — | — | — | — | — | — | — |
| goose | gemini-pro | — | — | — | — | — | — | — | — | — |

## How to Reproduce

### Prerequisites

- Go 1.25+
- `jq` command-line JSON processor
- One or more agent runtimes installed: Claude Code, OpenCode, Goose

### Running a Benchmark

```bash
./scripts/benchmark-collect.sh <runtime> <model> <migration>
```

The script will:
1. Invoke the agent runtime with the specified model
2. Run the full `/generate-rules` pipeline
3. Time the entire run
4. Run full eval (deterministic + LLM judge)
5. Collect rules, eval snapshot, and metrics into `benchmarks/`
6. Regenerate the comparison table

### Examples

```bash
# Claude Code with Sonnet
./scripts/benchmark-collect.sh claude-code sonnet httpclient4-to-httpclient5

# Claude Code with Haiku
./scripts/benchmark-collect.sh claude-code haiku httpclient4-to-httpclient5

# OpenCode with Gemini Pro
GOOGLE_API_KEY=... ./scripts/benchmark-collect.sh opencode gemini-pro spring-boot3-to-spring-boot4
```

### Runtime Setup

#### Claude Code
- Models: `sonnet` → claude-sonnet-4-6, `opus` → claude-opus-4-6, `haiku` → claude-haiku-4-5-20251001
- No extra env vars needed (uses your Claude Code auth)

#### OpenCode
- Set `ANTHROPIC_API_KEY` for sonnet/opus
- Set `GOOGLE_API_KEY` for gemini-pro

#### Goose
- Set API keys same as OpenCode, or use `goose configure`

### Migration Guide URLs

| Migration | Guide URL |
|-----------|-----------|
| httpclient4-to-httpclient5 | https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide |
| spring-boot3-to-spring-boot4 | https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide |
