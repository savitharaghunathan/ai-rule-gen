# Architecture

## Overview

ai-rule-gen has two entry points sharing identical internal packages:

- **MCP server** (`rulegen serve`): Exposes 4 deterministic tools over SSE. Client LLM (Claude Code, Cursor, Kai) reads migration guides and calls tools to construct valid rule YAML. No server-side LLM needed.
- **CLI** (`rulegen generate/test/score`): Runs E2E pipelines with a server-side LLM. For CI/automation.

### Interactive Workflow (MCP Client — no server LLM)

```
You: "Generate rules from https://example.com/spring-boot-4-migration"
 │
 ▼
Client LLM (Claude/Cursor/Kai):
 │  1. Fetches the URL, reads the migration guide
 │  2. Understands the content — identifies breaking changes, API removals
 │  3. For each migration pattern it finds, calls construct_rule:
 │
 ├── construct_rule(ruleID, condition_type, pattern, location, message, ...)
 │     → returns valid YAML ✓
 ├── construct_rule(...)  → valid YAML ✓
 ├── construct_rule(...)  → valid YAML ✓
 ├── construct_ruleset(name, description, labels)  → ruleset YAML ✓
 └── validate_rules(rules_path)  → {valid: true, rule_count: 3}
 │
 ▼
┌──────────────────────────────────────────────────────────────────┐
│  MCP Server (deterministic tools — no LLM, just builds YAML)    │
│  construct_rule · construct_ruleset · validate_rules · get_help  │
└──────────────────────────────────────────────────────────────────┘
```

### Pipeline Workflow (CLI — server-side LLM)

```
$ export RULEGEN_LLM_PROVIDER=anthropic
$ export ANTHROPIC_API_KEY=sk-ant-...
$ rulegen generate --guide-url <URL> --source s --target t --language java
 │
 ▼
┌────────────────────────────────────────────────────────────────┐
│  CLI does everything automatically:                            │
│                                                                │
│  1. INGEST:   Fetch URL → HTML→markdown → chunk if large       │
│  2. EXTRACT:  Send chunks + prompt to LLM → []MigrationPattern │
│  3. CONSTRUCT: Pattern → Rule (deterministic, same as          │
│               construct_rule tool)                              │
│  4. VALIDATE: Same checks as validate_rules tool               │
│  5. SAVE:    Write to output/<source>-to-<target>/rules/       │
│                                                                │
│  Providers: Anthropic, OpenAI, Gemini, Ollama (local)          │
└──────────────────────┬─────────────────────────────────────────┘
                       │
                       ▼
         output/spring-boot-3-to-spring-boot-4/rules/
         ├── ruleset.yaml
         ├── web.yaml
         └── security.yaml
```

### Same output, different "brain"

| | Interactive (MCP) | Pipeline (CLI) |
|---|---|---|
| **Who reads the guide?** | Client LLM (Claude/Cursor) | Server-side LLM |
| **Who picks the patterns?** | Client LLM | Server-side LLM |
| **Who builds the YAML?** | Server (`construct_rule`) | CLI (same internal code) |
| **Who validates?** | Server (`validate_rules`) | CLI (same internal code) |
| **API key needed?** | No | Yes |
| **Human in the loop?** | Yes | No |
| **Best for** | Crafting rules, learning, small batches | CI/CD, bulk generation, automation |

### System Architecture

```
┌──────────────────────────┐      ┌───────────────────────┐
│  MCP Clients             │      │  CLI                   │
│  Claude Code, Cursor,    │      │  rulegen generate ...  │
│  Kai                     │      │  rulegen test ...      │
└──────────┬───────────────┘      │  rulegen score ...     │
           │ MCP (SSE)            └───────────┬────────────┘
           ▼                                  │ direct Go calls
┌────────────────────────┐                    │
│  MCP Server            │                    │
│  (4 deterministic      │                    │
│  tools, no LLM)        │                    │
│                        │                    │
│  construct_rule        │                    │
│  construct_ruleset     │                    │
│  validate_rules        │                    │
│  get_help              │                    │
└──────────┬─────────────┘                    │
           │                                  │
           ▼                                  ▼
┌────────────────────────────────────────────────────────────┐
│  Shared internal packages                                  │
│  rules/     ingestion/     extraction/     generation/     │
│  testing/   confidence/    workspace/      llm/            │
└────────────────────────────────┬───────────────────────────┘
                                 │
                                 ▼ shells out
                   ┌───────────────┐  ┌────────────────┐
                   │  kantra test  │  │ rulesets repo   │
                   └───────────────┘  │ (output)        │
                                      └────────────────┘
```

## MCP Tools (4 deterministic)

| Tool | Description |
|------|-------------|
| `construct_rule` | Takes all rule parameters (ruleID, condition type, pattern, location, message, category, effort, labels, links), returns valid YAML. Rich description for client LLM. |
| `construct_ruleset` | Takes name, description, labels, returns ruleset YAML. |
| `validate_rules` | Validates rule YAML: structure, required fields, regex syntax, label format, duplicate ruleIDs. |
| `get_help` | Returns documentation on rule format, condition types, locations, examples. |

## CLI Commands (pipeline, require LLM)

| Command | Description |
|---------|-------------|
| `rulegen generate` | E2E pipeline: ingest → extract patterns (LLM) → construct rules → validate → save to disk. |
| `rulegen test` | ARG-style test generation + `kantra test` + autonomous fix loop. |
| `rulegen score` | LLM-as-judge scoring with adversarial rubric in fresh context. |

## Transport: SSE

SSE is the primary transport via the official Go MCP SDK. No stdio.

- Works for all target clients: Claude Code, Kai, Cursor
- Single deployment model — run locally (`localhost:port`) or containerized
- Bind to `localhost` by default for security; configurable for remote use

## LLM Provider Configuration

CLI pipeline commands require a server-side LLM. Configured via environment variables:

| Variable | Description |
|----------|-------------|
| `RULEGEN_LLM_PROVIDER` | `anthropic`, `openai`, `gemini`, `ollama` |
| `ANTHROPIC_API_KEY` | Anthropic (Claude) API key |
| `OPENAI_API_KEY` | OpenAI (GPT) API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `OLLAMA_HOST` | Ollama server URL (default: `http://localhost:11434`) |
| `OLLAMA_MODEL` | Ollama model name (default: `llama3`) |

## Completer Interface

All LLM-powered logic is parameterized by a `Completer` interface:

```go
// Completer abstracts LLM inference
type Completer interface {
    Complete(ctx context.Context, prompt string) (string, error)
}

// Server-side LLM: calls provider API directly
type LLMCompleter struct { provider Provider }

// Provider implementations: Anthropic, OpenAI, Gemini, Ollama
type Provider interface {
    Complete(ctx context.Context, prompt string) (string, error)
}
```

Every internal package that needs LLM inference accepts a `Completer`. This means:
- No package knows or cares which LLM backend is in use
- Unit tests inject a `MockCompleter` that returns fixture responses
- Adding a new LLM provider only requires implementing the `Provider` interface

## generate Pipeline

The primary CLI pipeline. Ingests any input, produces validated rules on disk.

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

In the **interactive workflow**, the client LLM performs steps 1-2 itself (reading the guide, identifying patterns), then calls `construct_rule` for step 3 and `validate_rules` for step 4.

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

`rulegen test` includes an autonomous fix loop:

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
         │ (Completer)  │
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
├── server/        MCP server setup, 4 tool registrations, SSE transport
├── tools/         MCP tool handlers (construct, validate, help) + pipeline logic (generate)
├── llm/           Completer interface, LLMCompleter, providers (Anthropic, OpenAI, Gemini, Ollama)
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
