# Architecture

## Overview

ai-rule-gen has two entry points that share the same internal Go packages. The MCP server uses the client's LLM via MCP sampling. The CLI uses a server-side LLM with an API key.

```
┌──────────────────────┐      ┌───────────────────────┐
│  MCP Clients         │      │  CLI / Pipeline        │
│  Claude Code, Kai,   │      │  rulegen generate ...  │
│  VS Code             │      │  rulegen validate ...  │
└──────────┬───────────┘      └───────────┬────────────┘
           │ MCP (SSE)                    │ direct Go calls
           ▼                              ▼
┌────────────────────┐       ┌────────────────────────┐
│  MCP Server        │       │  CLI commands (Cobra)  │
│  (tool handlers    │       │  + server-side LLM     │
│   + MCP sampling)  │       │    (API key required)  │
└────────┬───────────┘       └───────────┬────────────┘
         │                               │
         └──────────┬────────────────────┘
                    ▼
      ┌───────────────────────────┐
      │  Shared internal packages │
      │  rules/     ingestion/    │
      │  extraction/ generation/  │
      │  testing/   confidence/   │
      └─────────────┬─────────────┘
                    │
                    ▼ shells out / imports
      ┌───────────────┐  ┌────────────────┐
      │  kantra test  │  │ rulesets repo   │
      └───────────────┘  │ (output)        │
                         └────────────────┘
```

| Entry Point | LLM | Transport |
|-------------|-----|-----------|
| MCP server | MCP sampling (client's LLM) | SSE |
| CLI | Server-side LLM (API key required) | Direct Go calls |

## Transport: SSE Only

SSE (Server-Sent Events over HTTP) is the sole transport. No stdio.

- Works for all target clients: Claude Code, Kai, VS Code, CI/CD pipelines
- Single deployment model — run locally (`localhost:port`) or containerized
- Bind to `localhost` by default for security; configurable for remote use

## Completer Interface

Both entry points share the same core logic, parameterized by an LLM interface:

```go
// Completer abstracts LLM inference
type Completer interface {
    Complete(ctx context.Context, prompt string) (string, error)
}

// MCP path: server builds prompt, client's LLM generates
type SamplingCompleter struct { sampler mcp.Sampler }

// CLI path: server calls LLM API directly
type LLMCompleter struct { client llm.Client }
```

Every internal package that needs LLM inference accepts a `Completer`. This means:
- No package knows or cares which LLM backend is in use
- Unit tests inject a `MockCompleter` that returns fixture responses
- Adding a new LLM provider only requires implementing the provider interface

## generate_rules Pipeline

The primary tool. Ingests any input, produces validated rules on disk.

```
Input (URL, code, changelog, text)
  │
  ▼
┌─────────┐     ┌──────────────────────┐     ┌──────────────┐
│ ingest  │────▶│ extract_patterns     │────▶│ construct    │
│         │     │ (LLM via Completer)  │     │ rules        │
│ URL→md  │     │                      │     │ (deterministic)
│ file    │     │ content → JSON       │     │              │
│ text    │     │ → []MigrationPattern │     │ pattern →    │
└─────────┘     └──────────────────────┘     │ Rule YAML    │
                                             └──────┬───────┘
                                                    │
                                             ┌──────▼───────┐
                                             │ validate     │
                                             │ + save       │
                                             │ to disk      │
                                             └──────────────┘
```

1. **Ingest** — fetch URL (HTML→markdown), read file, or pass through text. Chunk if content exceeds LLM context limits.
2. **Extract patterns** — LLM call per chunk using Go template prompt. Parses JSON response into `[]MigrationPattern`. Deduplicates across chunks.
3. **Construct rules** — deterministic mapping: `MigrationPattern` → `Rule`. Uses condition builders for the correct provider type. Generates ruleset metadata. Groups rules by concern into separate files.
4. **Validate + save** — structural validation, then write to `output/<source>-to-<target>/rules/`.

The LLM discovers *what* to detect. Deterministic code ensures *how* it's expressed in valid YAML.

## Test Data Generation Pipeline

Follows the proven [ARG](https://github.com/konveyor-ecosystem/analyzer-rule-generator) approach:

```
Rules YAML
  │
  ▼
┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│ extract code │────▶│ build prompt  │────▶│ call LLM     │
│ hints from   │     │ (Go templates │     │ (Completer)  │
│ patterns +   │     │  + lang-      │     │              │
│ messages     │     │  specific)    │     │              │
└──────────────┘     └───────────────┘     └──────┬───────┘
                                                  │
                     ┌───────────────┐     ┌──────▼───────┐
                     │ write files   │◀────│ post-process │
                     │ + .test.yaml  │     │ extract code │
                     │               │     │ blocks,      │
                     │               │     │ validate     │
                     │               │     │ language,    │
                     │               │     │ inject       │
                     │               │     │ imports      │
                     └───────────────┘     └──────────────┘
```

Language-specific templates (Java, Go, C#, TypeScript) produce build files, source files, and config files that trigger rule patterns.

## Test-Fix Loop

`run_tests` includes an autonomous fix loop:

```
         ┌──────────────┐
         │ kantra test  │
         └──────┬───────┘
                │
         pass? ─┤── yes → done
                │
                ▼ no
         ┌──────────────┐
         │ analyze      │
         │ failures     │
         └──────┬───────┘
                │
                ▼
         ┌──────────────┐
         │ LLM: improved│
         │ code hints   │
         └──────┬───────┘
                │
                ▼
         ┌──────────────┐
         │ regenerate   │
         │ test data    │──── loop back to kantra test
         └──────────────┘     (max_iterations, default 3)
```

Rules are the source of truth — the loop fixes **test data, not rules**. ARG achieves 70% → 95%+ pass rate in 2-3 iterations.

## Confidence Scoring

Independence is about **context, not vendor**. The judge never sees the generation prompt, chain-of-thought, or migration guide. It receives only the raw rule YAML and an adversarial rubric.

Five criteria scored 1-5: pattern correctness, message quality, category appropriateness, effort accuracy, false positive risk. Verdicts: accept (≥4.0), review (≥2.5), reject (<2.5). Evidence required for every deduction.

## Package Layout

```
internal/
├── server/        MCP server setup, tool registration, SSE transport
├── tools/         5 MCP tool handlers (thin wrappers calling internal packages)
├── llm/           Completer interface, SamplingCompleter, LLMCompleter, API providers
├── rules/         Rule/Ruleset/Condition types, builders, YAML serialization, validation
├── ingestion/     URL/file/text ingestion, HTML→markdown, content chunking
├── extraction/    MigrationPattern type, LLM-driven pattern extraction
├── generation/    Pattern→Rule deterministic construction, rule ID generation
├── testing/       Test scaffolding, ARG-style code generation, kantra runner, fix loop
├── confidence/    LLM-as-judge scorer, adversarial rubric
├── integration/   Integration tests (build tag: integration)
└── workspace/     Output directory management
```

All LLM prompts are Go templates in `templates/`, not hardcoded in source.
