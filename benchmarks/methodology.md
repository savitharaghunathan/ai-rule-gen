# Methodology

## Scoring Definitions

- **Pipeline**: Skill-based runs (Claude Code, OpenCode, Goose) invoke the `/generate-rules` skill with the same migration guide URL. Scribe uses a separate MCP-based pipeline with no kantra testing.
- **Evaluation**: Deterministic eval (`cmd/eval`) + LLM judge (`agents/eval/SKILL.md`)
- **Quality Score**: Completeness metric (max 6 pts: message presence + links + effort + before/after guidance). Measures documentation completeness, not rule correctness.
- **Overlaps**: Count of rule pairs that fire on the same code. Overlaps can indicate specificity layering (a broad package-level rule + a specific method-level rule covering the same API), which improves developer experience. High overlap count is not inherently bad — it may mean better coverage through layered detection.
- **Eval Judge**: LLM-based review checking precision (false positive risk), coherence (detection/guidance alignment), cross-rule conflicts, and coverage gaps vs. the migration guide.
- **Timing**: Wall-clock from pipeline start to report completion. "Rule Gen" = ingest through construct stages. "Test" = scaffold through kantra validation. Scribe has no test phase.
- **Gaps**: Migration patterns from the guide with no corresponding rule. "Adjusted Gaps" accounts for broad package-level rules (e.g., `org.apache.http*` at PACKAGE) that provide generic coverage for all sub-packages — these are not counted as gaps even though guidance is not package-specific.
- **Date**: May–June 2026

## Runtime × Model Matrix

| Runtime | Model | Model ID | Provider |
|---------|-------|----------|----------|
| Claude Code | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Claude Code | Opus | claude-opus-4-6 | Anthropic |
| Claude Code | Haiku | claude-haiku-4-5-20251001 | Anthropic |
| OpenCode | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| OpenCode | Opus | claude-opus-4-6 | Anthropic |
| OpenCode | Gemini Pro | google-vertex/gemini-3.1-pro-preview¹ | Google |
| Goose | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Goose | Opus | claude-opus-4-6 | Anthropic |
| Goose | Gemini Pro | gemini-2.5-pro | Google |
| OpenCode | DeepSeek V3.2 | google-vertex/deepseek-ai/deepseek-v3.2-maas | Google (Vertex) |
| Scribe (MCP) | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Scribe (MCP) | Opus | claude-opus-4-6 | Anthropic |

¹ OpenCode's Gemini Pro model ID (`gemini-3.1-pro-preview`) does not match a known public Gemini model. Likely the same `gemini-2.5-pro` used by Goose, accessed via a different Vertex AI endpoint. Unverified.

## How to Reproduce

Each benchmark was run manually by invoking the `/generate-rules` skill in the respective agent runtime (Claude Code, OpenCode, Goose) with the migration guide URL, then running `cmd/eval` on the output. Scribe benchmarks used Scribe's own MCP pipeline. There is no automated benchmark collection script yet.

### Migration Guide URLs

| Migration | Guide URL |
|-----------|-----------|
| httpclient4-to-httpclient5 | https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide |
| spring-boot3-to-spring-boot4 | https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide |
