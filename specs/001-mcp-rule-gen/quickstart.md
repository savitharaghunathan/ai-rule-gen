# Quickstart: Phase 1 MCP Server for AI-Powered Rule Generation

**Date**: 2026-03-19 | **Plan**: [plan.md](plan.md)

## Prerequisites

- **Go 1.22+**: Build and run the MCP server / CLI
- **kantra**: Required for `run_tests` tool. Must be installed on the machine where the server runs.

## Build & Run

```bash
# Build
go build -o rulegen ./cmd/rulegen/

# Start MCP server (SSE on localhost:8080)
./rulegen serve --port 8080

# Or with custom bind address
./rulegen serve --addr 0.0.0.0:8080
```

## Connect from Claude Code

Add to your MCP client config:

```json
{
  "mcpServers": {
    "rulegen": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

## Interactive Use (MCP)

No API key needed — uses the client's LLM via MCP sampling.

### 1. Generate rules from a migration guide

```
generate_rules({
  "guide_url": "https://spring.io/blog/migration-guide",
  "source": "spring-boot-3",
  "target": "spring-boot-4",
  "language": "java"
})
```

Output saved to `output/spring-boot-3-to-spring-boot-4/rules/`.

### 2. Validate generated rules

```
validate_rules({
  "rules_path": "./output/spring-boot-3-to-spring-boot-4/rules/web.yaml"
})
```

### 3. Generate test data

```
generate_test_data({
  "rules_path": "./output/spring-boot-3-to-spring-boot-4/rules/web.yaml",
  "language": "java"
})
```

Output saved to `output/spring-boot-3-to-spring-boot-4/tests/`.

### 4. Run tests (with autonomous fix loop)

```
run_tests({
  "test_file": "./output/spring-boot-3-to-spring-boot-4/tests/web.test.yaml",
  "max_iterations": 3
})
```

Requires `kantra` installed locally.

### 5. Score confidence

```
score_confidence({
  "rules_path": "./output/spring-boot-3-to-spring-boot-4/rules/web.yaml"
})
```

Output saved to `output/spring-boot-3-to-spring-boot-4/confidence/scores.yaml`.

## CLI Use (Pipeline / CI)

Requires a configured LLM API key.

```bash
export RULEGEN_LLM_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-...

# Full pipeline: generate + validate + test + score
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

## Output Structure

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

Rules output is directly submittable as a PR to `konveyor/rulesets`.
