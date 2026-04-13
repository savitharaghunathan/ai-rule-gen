# ai-rule-gen

An MCP server and CLI for generating [Konveyor](https://www.konveyor.io/) analyzer rules using AI. Point it at a migration guide, code snippets, or any description of migration concerns — it generates validated rules ready for the [konveyor/rulesets](https://github.com/konveyor/rulesets) repo.

Two entry points, shared internals:
- **MCP server** — 4 deterministic tools for interactive rule construction from Claude Code, Cursor, Kai, or any MCP client. No server-side LLM needed.
- **CLI** — E2E pipeline for CI/CD automation with server-side LLM. Auto-detects source/target/language from content.

## MCP Tools

| Tool | Description |
|------|-------------|
| `construct_rule` | Takes rule parameters (ruleID, condition type, pattern, location, message, etc.), validates, returns valid YAML |
| `construct_ruleset` | Takes name, description, labels, returns ruleset metadata YAML |
| `validate_rules` | Structural validation: required fields, category, effort, regex, labels, duplicates |
| `get_help` | Documentation on condition types, valid locations, label format, categories, examples |

## CLI Commands

| Command | Description | Status |
|---------|-------------|--------|
| `rulegen generate` | Ingest input (URL, file, text) → extract patterns via LLM → construct rules → validate → save | Implemented |
| `rulegen validate` | Structural validation of rule YAML (directory or file); prints JSON, no LLM | Implemented |
| `rulegen test` | Generate test data, run kantra, auto-fix **test data** via LLM hints (up to `--max-iterations`) | Implemented |
| `rulegen score` | Run kantra tests for functional confidence + optional LLM-as-judge | Experimental |

## Prerequisites

- **Go 1.22+**
- **kantra** — required for `rulegen test` (must be on PATH)

## Build

```bash
go build -o rulegen ./cmd/rulegen/
```

## Usage

### MCP Server

Start the server — no API key needed. Supports two transports:

```bash
# stdio (default) — for local MCP clients
./rulegen serve

# Streamable HTTP — for remote/shared deployments
./rulegen serve --transport http --port 8080
```

#### Connect from Claude Code (stdio — recommended)

Stdio is the MCP-recommended transport for local servers. The client launches the server as a subprocess — no separate process to manage, and access is restricted to just the MCP client.

Add `.mcp.json` to your project root:

```json
{
  "mcpServers": {
    "rulegen": {
      "type": "stdio",
      "command": "./rulegen",
      "args": ["serve"]
    }
  }
}
```

#### Connect from Claude Code (Streamable HTTP)

Use Streamable HTTP when the server runs remotely, is shared across multiple clients, or you want to manage the server lifecycle independently (e.g., for debugging). Requires starting the server separately with `./rulegen serve --transport http --port 8080`.

```json
{
  "mcpServers": {
    "rulegen": {
      "type": "streamable-http",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

#### Connect from Cursor

Streamable HTTP (server must be running separately):

```json
{
  "mcpServers": {
    "rulegen": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

If your Cursor version supports stdio MCP servers, you can use the same `.mcp.json` as in **Connect from Claude Code (stdio — recommended)** above.

#### Example: Generate rules interactively

Once connected, ask your MCP client:

```
Use the rulegen MCP server to generate Konveyor analyzer rules for this migration guide:
https://gist.github.com/savitharaghunathan/52198c722b807f3862af38b72e6d7331

Save the rules to the output folder with source and target labels.
```

The client LLM will:
1. Call `get_help` to learn about condition types and locations
2. Read the migration guide content
3. Call `construct_rule` for each migration pattern it identifies
4. Call `construct_ruleset` to create ruleset metadata
5. Call `validate_rules` to verify the output

No server-side LLM or API key is needed — the client's LLM does all the thinking.

### CLI

Set your LLM provider and API key:

```bash
export GEMINI_API_KEY=your-key
```

Generate rules (source/target/language auto-detected from content):

```bash
./rulegen generate \
  --input "https://gist.github.com/savitharaghunathan/52198c722b807f3862af38b72e6d7331" \
  --provider gemini
```

Or specify everything explicitly:

```bash
./rulegen generate \
  --input "https://spring.io/blog/migration-guide" \
  --source spring-boot-3 \
  --target spring-boot-4 \
  --language java \
  --output ./output \
  --provider anthropic
```

#### `generate` flags

| Flag | Description | Required |
|------|-------------|----------|
| `--input` | URL, file path, or text content | Yes |
| `--source` | Source technology (auto-detected if omitted) | No |
| `--target` | Target technology (auto-detected if omitted) | No |
| `--language` | Programming language: java, go, nodejs, csharp (auto-detected if omitted) | No |
| `--output` | Output directory (default: `output`) | No |
| `--provider` | LLM provider: `anthropic`, `openai`, `gemini`, `ollama` (overrides `RULEGEN_LLM_PROVIDER` env var) | Yes |

### Validate rules

Validate existing rule YAML without an LLM (same structural checks as the `validate_rules` MCP tool):

```bash
./rulegen validate --rules ./output/my-ruleset/rules
```

Use a directory of `.yaml` files or a single rule file. Prints JSON to stdout; exits with a non-zero status if validation fails.

### Test Rules

Generate test data, run kantra tests, and auto-fix **test data** (not rule YAML) when the compile or kantra steps fail:

```bash
./rulegen test \
  --rules output/golang-non-fips-crypto-to-golang-fips-crypto/rules \
  --output output/golang-non-fips-crypto-to-golang-fips-crypto \
  --provider gemini \
  --max-iterations 3
```

The test-fix loop:
1. Generates test source code that should trigger each rule
2. **Phase A — Compile fix**: Checks compilation (`go build`, `mvn compile`, `npx tsc`, `dotnet build`), feeds errors + API docs back to the LLM, retries up to 5 times
3. **Phase B — Kantra test**: Runs `kantra test` on generated test data
4. When tests still fail, asks the LLM for code hints, regenerates test data, and re-runs (up to `--max-iterations`)
5. **Consistency check**: Verifies every rule has a test case and every test references a real rule

### Score Confidence (Experimental)

> Requires `--experimental` flag: `./rulegen --experimental score ...`

Score rules by running kantra tests (primary signal — does the rule actually work?):

```bash
./rulegen --experimental score \
  --tests output/go-non-fips-crypto-to-go-fips-140-compliance/tests
```

Add LLM-as-judge as a secondary quality signal:

```bash
./rulegen --experimental score \
  --tests output/go-non-fips-crypto-to-go-fips-140-compliance/tests \
  --rules output/go-non-fips-crypto-to-go-fips-140-compliance/rules \
  --provider gemini
```

Verdict logic:
- kantra fail → **reject** (rule doesn't match test data)
- kantra pass + judge reject → **review** (works but quality concerns)
- kantra pass + judge accept → **accept**

#### `score` flags

| Flag | Description | Required |
|------|-------------|----------|
| `--tests` | Directory containing `.test.yaml` files | Yes |
| `--rules` | Rules directory; required when using `--provider` (LLM judge) | With `--provider` |
| `--output` | Project root for `confidence/scores.yaml` (default: print only) | No |
| `--kantra` | Path to `kantra` binary (default: `kantra` on `PATH`) | No |
| `--timeout` | Kantra timeout in seconds (default: `900`) | No |
| `--provider` | LLM provider for judge: `anthropic`, `openai`, `gemini`, `ollama` | No |

#### LLM Provider Configuration

| Provider | API Key Env Var | Model Env Var | Default Model |
|----------|----------------|---------------|---------------|
| `anthropic` | `ANTHROPIC_API_KEY` | `ANTHROPIC_MODEL` | `claude-sonnet-4-5` |
| `openai` | `OPENAI_API_KEY` | `OPENAI_MODEL` | `gpt-4o` |
| `gemini` | `GEMINI_API_KEY` | `GEMINI_MODEL` | `gemini-2.5-flash` |
| `ollama` | — | `OLLAMA_MODEL` | `llama3` |

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
    └── scores.yaml  # kantra test results + optional LLM judge scores
```

## Supported Condition Types

Java (`java.referenced`, `java.dependency`), Go (`go.referenced`, `go.dependency`), Node.js (`nodejs.referenced`), C# (`csharp.referenced`), and builtin (`filecontent`, `file`, `xml`, `json`, `hasTags`, `xmlPublicID`), plus `and`/`or` combinators.

## Testing

```bash
go test ./internal/...                                    # Unit tests
go test -tags=integration ./test/integration/...          # Integration tests (mock LLM)
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
