# ai-rule-gen

Generate [Konveyor](https://www.konveyor.io/) analyzer migration rules from any migration guide. Point your AI coding agent at a guide and get validated, tested rules ready for the [konveyor/rulesets](https://github.com/konveyor/rulesets) repo.

## Install

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [kantra](https://github.com/konveyor-ecosystem/kantra) (for rule testing)
- An AI coding agent ([Claude Code](https://claude.ai/code), [Cursor](https://cursor.sh), [Goose](https://github.com/block/goose), or similar)

### Add the skill to Claude Code

```bash
# Clone the repo
git clone https://github.com/konveyor/ai-rule-gen.git
cd ai-rule-gen

# Add as a project skill (from inside your target project)
claude project add-skill /path/to/ai-rule-gen/.claude/commands/generate-rules.md
```

Or copy `.claude/commands/generate-rules.md` into your project's `.claude/commands/` directory.

## Usage

From Claude Code, invoke the skill:

```
/generate-rules https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
```

The input can be a URL, a file path, or pasted migration guide text.

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
├── rules/                # Rule YAML files ready for konveyor/rulesets
│   ├── ruleset.yaml
│   ├── web.yaml
│   └── ...
├── tests/                # Kantra test suites
├── patterns.json         # Extracted migration patterns
├── guide.md              # Ingested migration guide
└── report.yaml           # Summary report
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
| **CLI commands** (`cmd/`) | Ingest, construct, validate, scaffold, sanitize, stamp, report | No |

### Agent skills

| Skill | Role |
|-------|------|
| **rule-writer** | Reads migration guide, extracts migration patterns into `patterns.json` |
| **test-generator** | Reads `manifest.json`, generates compilable test source code |
| **rule-validator** | Runs kantra, interprets results, generates fix hints |
| **orchestrator** | Coordinates the full end-to-end pipeline |

Each skill has reference docs in its `references/` directory.

## Related Projects

- [analyzer-rule-generator (ARG)](https://github.com/konveyor-ecosystem/analyzer-rule-generator) — Python rule generation pipeline
- [kantra](https://github.com/konveyor-ecosystem/kantra) — Rule testing CLI
- [analyzer-lsp](https://github.com/konveyor/analyzer-lsp) — Rule engine and analyzer

## License

Apache-2.0
