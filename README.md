# ai-rule-gen

An MCP server and CLI for generating [Konveyor](https://www.konveyor.io/) analyzer rules using AI. Point it at a migration guide, code snippets, or any description of migration concerns — it generates validated rules, tests, and confidence scores ready for the [konveyor/rulesets](https://github.com/konveyor/rulesets) repo.

Works as an MCP server (connect from Claude Code, Kai, or any MCP client) or as a standalone CLI for CI/CD pipelines.

## Tools

| Tool | Description |
|------|-------------|
| `generate_rules` | Ingests any input (URL, code, changelog, text), extracts migration patterns via LLM, constructs valid YAML rules, saves to disk |
| `validate_rules` | Structural validation: required fields, category, effort, regex, labels, duplicates |
| `generate_test_data` | Scaffolds `.test.yaml` and generates compilable test source code |
| `run_tests` | Executes `kantra test` with autonomous test-fix loop (fixes test data, not rules) |
| `score_confidence` | LLM-as-judge scoring with adversarial rubric — accept/review/reject verdicts with evidence |

## Prerequisites

- **Go 1.22+**
- **kantra** — required for `run_tests` (must be on PATH where the server runs)

## Build

```bash
go build -o rulegen ./cmd/rulegen/
```

## Usage

### MCP Server

No API key needed — uses the client's LLM via MCP sampling.

```bash
./rulegen serve --port 8080
```

Connect from your MCP client:

```json
{
  "mcpServers": {
    "rulegen": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

Then call tools interactively:

```
generate_rules({
  "guide_url": "https://spring.io/blog/migration-guide",
  "source": "spring-boot-3",
  "target": "spring-boot-4",
  "language": "java"
})
```

### CLI

Requires an LLM API key:

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
