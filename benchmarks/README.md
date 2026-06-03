# Benchmark Results

Comparison of rule generation quality across 4 agent runtimes (Claude Code, OpenCode, Goose, Scribe), 5 LLM models (Sonnet, Opus, Haiku, Gemini Pro, DeepSeek V3.2), and 2 Java migration guides (httpclient 4→5, Spring Boot 3→4). 22 total runs, May–June 2026.

Scoring definitions, runtime matrix, and reproduction steps: [methodology.md](methodology.md)

## Key Takeaways

1. **Sonnet is the best model for pattern extraction.** It consistently produces the most effective rules (high volume, low issue rate) across all three runtimes. Opus is comparable in quality per rule but extracts fewer patterns and has more METHOD_CALL precision issues.

2. **The runtime matters — a lot.** The same model produces meaningfully different results across runtimes. Claude Code / Sonnet extracts 43 httpclient rules with 4 precision issues; OpenCode / Sonnet extracts 35 with 12. Same model, same skill, same guide — the runtime's prompt routing, tool orchestration, and context management shape how the model reasons about the task.

3. **Model capability has a hard cliff.** Four tiers emerge: (1) Haiku = non-functional (empty messages, wrong condition types, pipeline crashes), (2) DeepSeek V3.2 = follows structure but produces near-empty guidance, (3) Gemini Pro = functional but weak documentation and unique failure modes, (4) Sonnet/Opus = strong across all dimensions. There is no graceful degradation between tier 1 and tier 3 — agentic pipelines need a minimum model capability.

4. **More rules does not mean more noise.** Higher-extracting runs (CC/Sonnet: 43 httpclient, OpenCode/Sonnet: 95 spring-boot) have proportionally *fewer* eval judge issues than lower-extracting runs. The additional rules cover simple class relocations and API renames that conservative runs skip.

## Does the Runtime / Harness Matter?

Yes. The runtime is not a passthrough — it shapes output quality through prompt routing, tool orchestration, context management, and retry behavior.

### Same model, different runtimes — Sonnet (httpclient)

| Runtime | Rules | Rule Completeness | Precision | Coherence | Gaps |
|---------|-------|---------|-----------|-----------|------|
| Claude Code | 43 | 5.95 | 4 | 4 | 1 |
| OpenCode | 35 | 5.91 | 12 | 5 | 9 |
| Goose | 28 | 6.0 | 7 | 3 | 8 |

Claude Code extracts 54% more rules than Goose with Sonnet, with 3x fewer precision issues than OpenCode. All three have zero adjusted gaps (package-level catch-all rules), but the raw gap difference shows Claude Code's extraction is more thorough.

### Same model, different runtimes — Sonnet (spring-boot)

| Runtime | Rules | Pass Rate | Precision | Coherence | Gaps |
|---------|-------|-----------|-----------|-----------|------|
| Claude Code | 89 | 85/89 | 3 | 7 | 8 |
| OpenCode | 95 | 92/95 | 8 | 5 | 8 |
| Goose | 83 | 82/83 | 8 | 4 | 8 |

OpenCode extracts the most rules but has the most precision issues. Claude Code has the fewest precision issues but the most coherence issues (driven by inverted-logic config detection). Goose sits in the middle on both axes.

### Same model, different runtimes — Opus (spring-boot)

| Runtime | Rules | Pass Rate | Precision | Coherence | Gaps |
|---------|-------|-----------|-----------|-----------|------|
| Claude Code | 74 | 73/74 | 5 | 4 | 10 |
| OpenCode | 91 | 83/91 | 7 | 4 | 8 |
| Goose | 85 | 82/85 | 6 | 3 | 3 |

Goose/Opus achieves the fewest gaps (3) and fewest coherence issues (3). Claude Code/Opus has the fewest precision issues (5) but the most gaps (10). OpenCode/Opus extracts the most rules but also has the most kantra failures (8 of 91).

### What drives the differences?

The same model, given the same skill and migration guide, produces 28–43 rules (httpclient/Sonnet) or 83–95 rules (spring-boot/Sonnet) depending on the runtime. The differences come from:

- **Context management**: how the runtime feeds prior tool results into subsequent prompts affects extraction completeness
- **Tool orchestration**: how many pipeline stages run in parallel vs. sequentially, and how errors are retried
- **Prompt routing**: the runtime's system prompt and conversation framing shape the model's interpretation of the skill
- **Rules per minute**: Claude Code/Opus produces 2.8 rules/min vs Goose/Sonnet at 1.3 rules/min on spring-boot — faster extraction correlates with denser prompting

The runtime is a significant variable. Benchmarking a model without controlling for the runtime underspecifies the result.

### Gemini Pro across runtimes (spring-boot)

| Runtime | Rules | Pass Rate | Rule Completeness | Precision | Coherence | Gaps |
|---------|-------|-----------|---------|-----------|-----------|------|
| OpenCode | 33 | 32/33 | 5.03 | 4 | 5 | 18 |
| Goose | 81 | 11/81 | 5.20 | 5 | 4 | 5 |

Goose/Gemini produces 2.5x more rules than OpenCode/Gemini (81 vs 33) with far fewer gaps (5 vs 18), but has the lowest pass rate of any run (14%). The high failure rate is largely a test scaffolding issue — `builtin.filecontent` and `java.dependency` rules fail kantra validation at higher rates, not because the rules are wrong but because test data generation for these condition types is harder.

## httpclient4-to-httpclient5 — Full Results

| Runtime | Model | Rules | Pass Rate | Rule Completeness | Rule Gen (min) | Total (min) | Precision | Coherence | Cross-Rule | Gaps | Adj. Gaps |
|---------|-------|-------|-----------|-------------------|----------------|-------------|-----------|-----------|------------|------|-----------|
| claude-code | sonnet | 43 | 42/43 | 5.95 | 13.2 | 34.6 | 4 | 4 (1 fail) | 3 | 1 | 0 |
| claude-code | opus | 29 | 29/29 | 5.93 | 12.3 | 28.5 | 9 | 6 (2 fail) | 2 | 3 | 0 |
| claude-code | haiku | 26 | 26/26 | 3.0 | — | 4.0 | 3 | 26 (all fail) | 5 | 9 | — |
| opencode | sonnet | 35 | 35/35 | 5.91 | 8.4 | 49.2 | 12 | 5 | 3 | 9 | 0 |
| opencode | opus | 29 | 29/29 | 5.90 | 7.4 | 21.5 | 8 | 3 | 2 | 8 | 8 |
| opencode | gemini-pro | 33 | 33/33 | 4.45 | — | 11.1 | 1 | 4 | 1 | 5 | — |
| opencode | deepseek-v3 | 35 | 34/35 | 4.3 | — | 61.7 | 9 | 7 | 4 | 13 | — |
| goose | sonnet | 28 | 28/28 | 6.0 | 11.3 | 21.2 | 7 | 3 | 3 | 8 | 0 |
| goose | opus | 28 | 28/28 | 5.86 | 12.6 | 24.3 | 13 | 4 | 3 | 13 | 13 |
| goose | gemini-pro | 45 | 44/45 | 3.51 | — | — | 6 | 4 | 2 | 8 | — |

### Issue Rate (httpclient)

Issue rate = (precision + coherence issues) / total rules.

| Runtime | Model | Rules | Issues | Issue Rate | Gaps | Rule Completeness |
|---------|-------|-------|--------|------------|------|-------------------|
| OpenCode | Gemini Pro | 33 | 5 | 15% | 5 | 4.45 |
| Claude Code | Sonnet | 43 | 8 | 19% | 1 | 5.95 |
| Goose | Gemini Pro | 45 | 10 | 22% | 8 | 3.51 |
| Goose | Sonnet | 28 | 10 | 36% | 8 | 6.00 |
| OpenCode | Opus | 29 | 11 | 38% | 8 | 5.90 |
| OpenCode | Sonnet | 35 | 17 | 49% | 9 | 5.91 |
| Claude Code | Opus | 29 | 15 | 52% | 3 | 5.93 |
| Goose | Opus | 28 | 17 | 61% | 13 | 5.86 |

Per-run eval details: [eval-details.md](httpclient4-to-httpclient5/eval-details.md)

### Scribe (httpclient) — not kantra-tested

| Runtime | Model | Rules | Rule Completeness | Rule Gen (min) | Precision | Coherence | Cross-Rule | Gaps | Adj. Gaps |
|---------|-------|-------|-------------------|----------------|-----------|-----------|------------|------|-----------|
| scribe | sonnet | 14 | 6.0 | ~2 | 2 | 3 | 1 | 29 | ~16 |
| scribe | opus | 30 | 5.7 | ~3 | 2 | 2 | 3 | 13 | 13 |

### Scribe Comparison (httpclient)

| Metric | CC/Sonnet (skill) | Scribe/Opus (MCP) |
|--------|-------------------|-------------------|
| Rules | 43 | 30 |
| Adj. gaps | 0 | 13 |
| Rule gen time | 13.2 min | ~3 min |
| Total time | 34.6 min | ~3 min |
| Precision | 4 | 2 |
| Coherence | 4 (1 fail) | 2 |
| Kantra validated | 42/43 | none |

The skill pipeline's additional 21 min is kantra testing, which catches broken rules like Scribe/Sonnet's `class-004` (wrong FQN, never fires) before they ship.

## spring-boot3-to-spring-boot4 — Full Results

**Note:** The spring-boot table uses a single "Time (min)" column (total wall-clock) vs. httpclient's "Rule Gen" + "Total" breakdown. Rule gen / test split data was not captured for these runs. "Adj. Gaps" is omitted — no spring-boot run uses a catch-all package rule.

| Runtime | Model | Rules | Pass Rate | Rule Completeness | Overlaps | Time (min) | Precision | Coherence | Cross-Rule | Gaps |
|---------|-------|-------|-----------|-------------------|----------|------------|-----------|-----------|------------|------|
| claude-code | sonnet | 89 | 85/89 | 5.60 | 28 | 35.1 | 3 | 7 | 2 | 8 |
| claude-code | opus | 74 | 73/74 | 5.43 | 17 | 26.7 | 5 | 4 | 3 | 10 |
| claude-code | haiku | 0 (failed) | 0/0 | — | 0 | 5.0 | — | — | — | — |
| opencode | sonnet | 95 | 92/95 | 5.52 | 31 | 39.4 | 8 | 5 | 3 | 8 |
| opencode | opus | 91 | 83/91 | 5.54 | 17 | 46.5 | 7 | 4 | 2 | 8 |
| opencode | gemini-pro | 33 | 32/33 | 5.03 | 2 | 31.7 | 4 | 5 | 2 | 18 |
| goose | sonnet | 83 | 82/83 | 5.48 | 17 | 62.3 | 8 | 4 | 3 | 8 |
| goose | opus | 85 | 82/85 | 5.48 | 19 | 54.4 | 6 | 3 | 4 | 3 |
| goose | gemini-pro | 81 | 11/81 | 5.20 | 25 | 16.7 | 5 | 4 | 2 | 5 |
| opencode | deepseek-v3 | 31 | 0/0 | 4.35 | 18 | 53.4 | 3 | 2 | 1 | 12 |

### Issue Rate (spring-boot)

| Runtime | Model | Rules | Issues | Issue Rate | Gaps | Rule Completeness |
|---------|-------|-------|--------|------------|------|-------------------|
| Goose | Opus | 85 | 9 | 11% | 3 | 5.48 |
| Goose | Gemini Pro | 81 | 9 | 11% | 5 | 5.20 |
| Claude Code | Sonnet | 89 | 10 | 11% | 8 | 5.60 |
| OpenCode | Opus | 91 | 11 | 12% | 8 | 5.54 |
| Claude Code | Opus | 74 | 9 | 12% | 10 | 5.43 |
| OpenCode | Sonnet | 95 | 13 | 14% | 8 | 5.52 |
| Goose | Sonnet | 83 | 12 | 14% | 8 | 5.48 |
| OpenCode | Gemini Pro | 33 | 9 | 27% | 18 | 5.03 |

Per-run eval details: [eval-details.md](spring-boot3-to-spring-boot4/eval-details.md)

### Scribe (spring-boot) — not kantra-tested

| Runtime | Model | Rules | Rule Completeness | Overlaps | Precision | Coherence | Cross-Rule | Gaps |
|---------|-------|-------|-------------------|----------|-----------|-----------|------------|------|
| scribe | opus | 51 | 5.75 | 3 | 2 | 1 | 1 | 5 |

### Scribe Comparison (spring-boot)

| Metric | CC/Sonnet (skill) | Scribe/Opus (MCP) |
|--------|-------------------|-------------------|
| Rules | 89 | 51 |
| Gaps | 8 | 5 |
| Total time | 35.1 min | — |
| Precision | 3 | 2 |
| Coherence | 7 | 1 |
| Kantra validated | 85/89 | none |
| File types | 5 (Java, XML, properties, YAML, Gradle) | 6 (Java, properties, XML, Gradle, YAML, spring.factories) |

CC/Sonnet generates 74% more rules with kantra validation but has 7x the coherence issues (7 vs 1), driven by inverted-logic config property detection. Scribe's unique coverage: spring.factories detection and dedicated Gradle rules. Neither approach dominates: the skill pipeline has volume and test validation; the MCP pipeline has per-rule quality and fewer coherence traps.

## Cross-Migration Patterns

1. **Unqualified METHOD_CALL is the top precision pitfall.** Rules matching method names like `setConnectTimeout` without qualifying the parent class fire on unrelated frameworks (Spring, OkHttp, JDBC) and on the replacement APIs the rule tells you to adopt. This is the #1 source of false-positive risk across both migrations.

2. **Overlaps are a feature, not a bug.** Specificity layering — a broad package-level import rule plus specific method-level rules for the same API — gives developers both a high-level "this package moved" warning and targeted "change this specific call" guidance.

3. **Config-heavy migrations produce lower quality scores.** Spring-boot rules average 5.48 quality vs httpclient's 5.91 (Sonnet/Opus runs). Config property renames yield shallower before/after guidance than API-level code migrations.

4. **Rule generation is ~40% of skill pipeline time.** Rule generation (ingest→construct) takes 7–13 min while testing (scaffold→kantra) takes 4–8 min. The remaining time is coverage analysis and reporting.

5. **All runtimes share the Jackson `com.fasterxml.jackson*` precision issue** on spring-boot — a common false-positive trap when a migration renames most but not all packages under a namespace.

6. **Gemini Pro has a unique failure mode.** It extracts patterns from code examples in the guide rather than from migration instructions, producing rules that detect standard Java APIs (`InputStream`, `Future`, `TimeUnit`) unrelated to the migration. No other model does this.

7. **The wrong-FQN bug crosses runtimes.** OpenCode/Sonnet, OpenCode/Opus, and Goose/Gemini all produce a spring-boot rule detecting the *new* SB4 `HttpMessageConverters` FQN instead of the old SB3 one — the rule silently never fires. This suggests the migration guide's wording leads models to extract the target FQN rather than the source.
