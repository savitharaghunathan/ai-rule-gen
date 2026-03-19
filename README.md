# ai-rule-gen

An MCP server and CLI for generating [Konveyor](https://www.konveyor.io/) analyzer rules using AI. Point it at a migration guide, code snippets, or any description of migration concerns — it generates validated rules, tests, and confidence scores ready for the [konveyor/rulesets](https://github.com/konveyor/rulesets) repo.

Two entry points, shared internals:
- **MCP server** — 4 deterministic tools for interactive rule construction from Claude Code, Cursor, Kai, or any MCP client. No server-side LLM needed.
- **CLI** — E2E pipeline for CI/CD automation with server-side LLM.

## MCP Tools

| Tool | Description |
|------|-------------|
| `construct_rule` | Takes rule parameters (ruleID, condition type, pattern, location, message, etc.), validates, returns valid YAML |
| `construct_ruleset` | Takes name, description, labels, returns ruleset metadata YAML |
| `validate_rules` | Structural validation: required fields, category, effort, regex, labels, duplicates |
| `get_help` | Documentation on condition types, valid locations, label format, categories, examples |

## CLI Commands (require `RULEGEN_LLM_PROVIDER`)

| Command | Description |
|---------|-------------|
| `rulegen generate` | Ingests any input (URL, code, changelog, text), extracts migration patterns via LLM, constructs valid YAML rules, saves to disk |
| `rulegen test` | Generates test data + runs `kantra test` with autonomous test-fix loop |
| `rulegen score` | LLM-as-judge scoring with adversarial rubric — accept/review/reject verdicts |

## Prerequisites

- **Go 1.22+**
- **kantra** — required for `rulegen test` (must be on PATH)

## Build

```bash
go build -o rulegen ./cmd/rulegen/
```

## Usage

### MCP Server

Start the server — no API key needed:

```bash
./rulegen serve --port 8080
```

#### Connect from Claude Code

Add `.mcp.json` to your project root:

```json
{
  "mcpServers": {
    "rulegen": {
      "type": "sse",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

#### Connect from Cursor

Add `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "rulegen": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

#### Example: Generate rules from a migration guide

Once connected, ask your MCP client:

```
Use the rulegen MCP server to generate Konveyor analyzer rules for this migration guide:
https://gist.github.com/savitharaghunathan/52198c722b807f3862af38b72e6d7331

Save the rules to the output/spring-boot-3-to-spring-boot-4/ folder with source and target labels.
```

The client LLM will:
1. Read the migration guide
2. Call `get_help` to learn about condition types and locations
3. Call `construct_rule` for each migration pattern it identifies
4. Call `construct_ruleset` to create ruleset metadata
5. Call `validate_rules` to verify the output

No server-side LLM or API key is needed — the client's LLM does all the thinking.

### CLI

Requires a server-side LLM provider:

```bash
export RULEGEN_LLM_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-...

# Full pipeline
rulegen generate \
  --guide-url https://spring.io/blog/migration-guide \
  --source spring-boot-3 \
  --target spring-boot-4 \
  --language java \
  --output ./output/

# Individual operations
rulegen validate --rules ./output/spring-boot-3-to-spring-boot-4/rules/
rulegen test --test-file ./output/.../tests/web.test.yaml --max-iterations 3
rulegen score --rules ./output/spring-boot-3-to-spring-boot-4/rules/
```

Supported providers: `anthropic`, `openai`, `gemini`, `ollama` (local models).

## Output

Output matches the [konveyor/rulesets](https://github.com/konveyor/rulesets) layout — directly submittable as a PR.

```
output/spring-boot-3-to-spring-boot-4/
├── rules/
│   ├── ruleset.yaml
│   ├── web.yaml
│   └── security.yaml
├── tests/
│   ├── web.test.yaml
│   └── data/web/
│       ├── pom.xml
│       └── src/main/java/com/example/App.java
└── confidence/
    └── scores.yaml
```

## Supported Condition Types

Java (`java.referenced`, `java.dependency`), Go (`go.referenced`, `go.dependency`), Node.js (`nodejs.referenced`), C# (`csharp.referenced`), and builtin (`filecontent`, `file`, `xml`, `json`, `hasTags`, `xmlPublicID`), plus `and`/`or` combinators.

## Testing

```bash
go test ./internal/...                                    # Unit tests
go test -tags=integration ./internal/integration/...      # Integration tests
go test -tags=e2e ./test/e2e/...                          # E2E tests (real LLM + kantra)
```

## Related Projects

| Project | Description |
|---------|-------------|
| [analyzer-rule-generator (ARG)](https://github.com/konveyor-ecosystem/analyzer-rule-generator) | Python, LLM-powered rule generation pipeline |
| [Scribe](https://github.com/sshaaf/scribe) | Java/Quarkus MCP server for rule construction |
| [analyzer-lsp](https://github.com/konveyor/analyzer-lsp) | Rule engine and analyzer |
| [kantra](https://github.com/konveyor-ecosystem/kantra) | Rule testing CLI |

## License

Apache-2.0
