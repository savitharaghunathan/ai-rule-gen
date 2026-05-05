# Runtime Permission Examples

Pre-configured permission files for running the generate-rules pipeline
without permission prompts on various agent runtimes.

Each file grants the minimum permissions declared in the skills'
`## Permissions` sections. Review before using.

## Usage

**Important:** Do not blindly overwrite your existing settings. Review the
example file and merge the relevant entries into your existing config.

| Runtime | Example file | Destination |
|---------|-------------|-------------|
| Claude Code | `claude-code.settings.example.json` | `.claude/settings.json` |
| OpenCode | `opencode.example.json` | `opencode.json` |
| Codex | `codex.example.toml` | `.codex/config.toml` |
| Gemini CLI | `gemini-cli.policies.example.toml` | `~/.gemini/policies/generate-rules.toml` |

## Runtimes without file-based permissions

Some runtimes configure permissions through their UI or CLI, not config files:

- **Cursor** — Use Settings UI to add `go run`, `go mod`, `mkdir`, `wc`, `grep` to the command allowlist.
- **Goose** — Run `goose configure` and set the developer shell tools to "Always Allow" for the commands listed in the skills' Permissions tables.
- **Windsurf** — Add command prefixes to `windsurf.cascadeCommandsAllowList` in VS Code settings.

## What these permissions grant

| Operation | Commands/Patterns |
|-----------|-------------------|
| shell | `go run ./cmd/{ingest,sections,merge-patterns,contract-validate,construct,validate,scaffold,sanitize,test,stamp,report,coverage} *`, `go mod *`, `go doc *`, `mkdir`, `wc`, `grep` |
| read | `output/**`, `agents/*/references/**` |
| write/edit | `output/**` |

See each skill's `## Permissions` section in `agents/*/SKILL.md` for
per-skill breakdowns.
