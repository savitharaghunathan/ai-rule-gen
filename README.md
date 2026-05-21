# ai-rule-gen

Generate [Konveyor](https://www.konveyor.io/) analyzer migration rules from any migration guide. Point your AI coding agent at a guide and get validated, tested rules ready for the [konveyor/rulesets](https://github.com/konveyor/rulesets) repo.

## Install

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [kantra](https://github.com/konveyor-ecosystem/kantra) (for rule testing)
- An AI coding agent ([Claude Code](https://claude.ai/code), [OpenCode](https://opencode.ai), [Goose](https://goose-docs.ai), [Codex](https://developers.openai.com/codex), or similar)

### Add the skill

```bash
git clone https://github.com/konveyor/ai-rule-gen.git
```

Skills follow the [Agent Skills](https://agentskills.io) format and are bundled in the repo. Open the repo in your agent — no registration step needed.

## Usage

### Claude Code

```
/generate-rules https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

### OpenCode

```
/generate-rules https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

### Goose

```
/skills generate-rules https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

### Codex

```
$generate-rules https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

### Any other agent

```
Read and follow agents/generate-rules/SKILL.md. Input: https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

The input can be a URL, a file path, or pasted migration guide text.

You can optionally pass explicit source and target labels (multiple of each are supported):

```
/generate-rules sources=["spring-boot3","spring-boot"] targets=["spring-boot4","spring-boot"] https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

Each source and target becomes its own `konveyor.io/source=` or `konveyor.io/target=` label on every generated rule, matching the [konveyor/rulesets](https://github.com/konveyor/rulesets) label conventions. If omitted, sources and targets are auto-detected from the guide.

The agent runs the full pipeline automatically:

1. **Ingest** the migration guide into clean markdown
2. **Extract** every migration pattern (API changes, dependency updates, config renames, POM structure)
3. **Construct** Konveyor rule YAML files from the patterns
4. **Scaffold** test directories and generate test source code
5. **Run kantra** to validate every rule finds incidents
6. **Fix** any failing rules (up to 3 iterations)
7. **Report** final pass rate and output locations

### Output

```
output/
└── <primary-source>-to-<primary-target>-<timestamp>/
    ├── guide.md          # Ingested migration guide
    ├── patterns.json     # Extracted migration patterns (sources/targets arrays)
    ├── rules/            # Rule YAML files ready for konveyor/rulesets
    │   ├── ruleset.yaml  # Ruleset with all source/target labels
    │   ├── web.yaml
    │   └── ...
    ├── tests/            # Kantra test suites
    └── report.yaml       # Summary report
```

## Architecture

All LLM orchestration lives in agent skills. The Go CLI is purely deterministic — no LLM calls, no API keys, no prompt templates.

```
Migration Guide → Agent extracts patterns → CLI constructs rules → CLI scaffolds tests
    → Agent generates test code → kantra validates → Agent fixes failures → Tested rules
```

| Layer | What | LLM? |
|-------|------|------|
| **Agent skills** (`agents/`) | Read guides, extract patterns, generate test code, fix failures | Yes |
| **CLI commands** (`cmd/`) | Ingest, construct, validate, scaffold, sanitize, test, report | No |

### Agent skills

| Skill | Role |
|-------|------|
| **generate-rules** | Orchestrates the full end-to-end pipeline |
| **rule-writer** | Reads migration guide, extracts migration patterns into `patterns.json` |
| **test-generator** | Reads `manifest.json`, generates compilable test source code |
| **rule-validator** | Runs kantra, interprets results, generates fix hints |
| **eval-judge** | LLM judge — compares guide vs rules, finds missed patterns and false positives |

Each skill follows the [Agent Skills](https://agentskills.io) format with a `SKILL.md` and optional `references/` directory.

## Eval

Measure the quality of generated rules with deterministic checks and an optional LLM judge.

### Quick start

```bash
# Quality-only (no app needed)
go run ./cmd/eval --rules-dir output/<migration>/rules

# With app coverage analysis
go run ./cmd/eval \
  --rules-dir output/<migration>/rules \
  --app-dir /path/to/app

# With LLM judge (invoke via your agent)
# Read and follow agents/eval-judge/SKILL.md
```

### What it measures

**Quality scoring (4-point scale per rule):**

| Check | Points | What it looks for |
|-------|--------|-------------------|
| Message | 1 | Rule has a non-empty migration message |
| Links | 1 | Rule includes documentation links |
| Effort | 1 | Rule has an effort rating |
| Before/after | 1 | Message contains migration guidance (e.g., "replace", "use", "instead of") |

**App coverage (requires `--app-dir`):**

Runs `kantra analyze` against a real application and cross-references results:

| Metric | Meaning |
|--------|---------|
| **Rules fired** | Rules that matched code in the app (raw kantra output) |
| **Effective match** | Fired rules / rules whose API is present in the app — excludes rules where the app simply doesn't use that API |
| **In app but unmatched** | API is in the source but kantra didn't match — potential broken rule or engine limitation |
| **Not in app** | App doesn't use the API — rule is correct, just not exercised by this app |
| **Incidents** | Total code locations matched across all fired rules |

**LLM judge (via eval-judge skill):**

Compares the source migration guide against the generated rules. Produces:
- **Missed patterns** — guide actions with no corresponding rule (severity: high/medium/low)
- **False positives** — rules that would fire incorrectly
- **Quality notes** — rules that work but could be improved

### Example report

```
======================================================================
EVAL REPORT
======================================================================

## Rules: 33

## Quality (avg 3.9/4)
   Messages:           33/33
   Links:              33/33
   Effort rating:      33/33
   Before/after:       29/33
   httpclient4-to-httpclient5-00010: missing [before_after_guidance]
   httpclient4-to-httpclient5-00020: missing [before_after_guidance]

## App Coverage
   Rules fired:      26/33 (78%)
   Effective match:  26/27 (96%)  — excludes rules where API is absent from app
   Incidents:        121

   In app but unmatched (1 rules):
     - httpclient4-to-httpclient5-00060 (BasicHttpContext) → src/test/java/...

   Not in app (6 rules):
     - httpclient4-to-httpclient5-00070 (HttpRequestBase)
     - httpclient4-to-httpclient5-00200 (URIUtils)
     ...

======================================================================
```

The report goes to stderr. Structured JSON goes to stdout for programmatic consumption.

### Eval config

Each eval case lives in `evals/<migration>/eval_config.yaml`:

```yaml
guide_url: https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide
app_repo: https://github.com/savitharaghunathan/httpclient-migration
source: httpclient4
target: httpclient5
```

See `evals/httpclient4-to-httpclient5/example-report.txt` for a full sample output.

## Related Projects

- [analyzer-rule-generator (ARG)](https://github.com/konveyor-ecosystem/analyzer-rule-generator) — Python rule generation pipeline
- [kantra](https://github.com/konveyor-ecosystem/kantra) — Rule testing CLI
- [analyzer-lsp](https://github.com/konveyor/analyzer-lsp) — Rule engine and analyzer

## License

Apache-2.0
