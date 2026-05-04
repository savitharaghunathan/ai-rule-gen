# Multi-Language Scalability Design

Date: 2026-05-04

## Problem

The ai-rule-gen pipeline has been validated end-to-end for Java only. It supports 4 languages (Java, Go, Node.js, C#) at the code level, but language-specific knowledge is scattered across hardcoded Go maps and monolithic agent reference docs. Adding a new language requires changes in 5 places across 3 directories. Non-Java languages have thin documentation that would cause agent failures during extraction, test generation, and fix loops.

## Goals

1. Make each language a self-contained plugin directory (zero Go code changes for new languages after initial refactor)
2. Fix documentation gaps for Go, Node.js, and C# so the pipeline is ready when someone brings a migration guide
3. Validate the pipeline end-to-end for Node.js/TypeScript with a real migration guide
4. Document Go container limitations as kantra-side issues (no workarounds in this repo)

## Non-Goals

- Adding new languages beyond the existing 4 (the plugin system enables this but we're not doing it now)
- Fixing kantra's Go container toolchain issue (tracked upstream)
- Generalizing `construct.go` condition builders (current switch works fine)

## Design

### Language Plugin Directories

Each language gets a self-contained directory under `languages/` at the repo root:

```
languages/
  java/
    config.json            # scaffold config + provider metadata
    condition-types.md     # java.referenced, java.dependency docs
    test-data-guide.md     # project structure, pom.xml rules, version bounds
    fix-strategies.md      # fix reference for validator
  go/
    config.json
    condition-types.md
    test-data-guide.md
    fix-strategies.md
  nodejs/
    config.json
    condition-types.md
    test-data-guide.md
    fix-strategies.md
  csharp/
    config.json
    condition-types.md
    test-data-guide.md
    fix-strategies.md
```

#### config.json Format

```json
{
  "language": "java",
  "providers": ["java", "builtin"],
  "scaffold": {
    "build_file": "pom.xml",
    "build_file_type": "xml",
    "source_dir": "src/main/java/com/example",
    "main_file": "Application.java",
    "main_file_type": "java"
  },
  "dependency_resolution": {
    "command": "",
    "note": "Do NOT run mvn compile. Kantra parses pom.xml directly."
  },
  "compilation_check": {
    "command": "",
    "note": "JDTLS in source-only mode resolves references without compilation."
  }
}
```

Language-specific examples:
- **Go:** `dependency_resolution.command = "go mod tidy && go mod vendor"`, `compilation_check.command = "go build ./..."`
- **Node.js:** `dependency_resolution.command = "npm install"`, `compilation_check.command = "npx tsc --noEmit"`
- **C#:** `dependency_resolution.command = "dotnet restore"`, `compilation_check.command = "dotnet build --no-restore"`

`config.json` files are hand-authored and checked into the repo. Adding a new language = creating a new directory with these 4 files.

### Go Code Changes

#### scaffold.go

Replace the hardcoded `languageConfigs` map with a disk-based loader:

```go
func LoadLanguageConfig(language, languagesDir string) (*LanguageConfig, error) {
    path := filepath.Join(languagesDir, language, "config.json")
    data, err := os.ReadFile(path)
    if err != nil {
        // Fall back to embedded defaults for backward compatibility
        if cfg, ok := defaultConfigs[language]; ok {
            return &cfg, nil
        }
        return nil, fmt.Errorf("no config for language %q: %w", language, err)
    }
    var cfg LanguageConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parsing %s: %w", path, err)
    }
    return &cfg, nil
}
```

The existing hardcoded map is kept as `defaultConfigs` for backward compatibility — if `languages/` doesn't exist, the pipeline still works.

Estimated change: ~40 lines new, ~10 lines modified.

#### construct.go

No changes. The `buildSingleCondition()` switch on `provider_type` handles known providers. New kantra providers (e.g., `python.referenced`) would need a new switch case (~3 lines), but this is a kantra provider addition, not a language plugin addition.

### Agent Skill Reference Changes

#### Current state (monolithic)

- `agents/rule-writer/references/condition-types.md` — 207 lines, all languages
- `agents/test-generator/references/test-data-guide.md` — 241 lines, all languages
- `agents/rule-validator/references/providers/<lang>.md` — already per-language

#### After (split)

**Shared references stay in skill directories:**
- `agents/rule-writer/references/builtin-conditions.md` — builtin.filecontent, builtin.xml, builtin.file, builtin.json, combinators, chaining (language-agnostic)
- `agents/rule-writer/references/rule-schema.md` — unchanged
- `agents/rule-writer/references/patterns-json-schema.md` — unchanged
- `agents/test-generator/references/test-data-common.md` — condition matching table, output format, manifest.json reading, XML sanitization, merging groups (language-agnostic)
- `agents/rule-validator/references/fix-strategies.md` — unchanged (language-agnostic flow)

**Language-specific content moves to plugins:**
- `languages/<lang>/condition-types.md` — provider-specific conditions (e.g., `java.referenced` locations, `java.dependency` version bounds)
- `languages/<lang>/test-data-guide.md` — project structure, dependency resolution, compilation, version rules
- `languages/<lang>/fix-strategies.md` — condition-type fix lookup for the validator

**SKILL.md updates:**
- rule-writer: "Read `languages/<language>/condition-types.md` and `references/builtin-conditions.md`"
- test-generator: "Read `languages/<language>/test-data-guide.md` and `references/test-data-common.md`"
- rule-validator: "Read `languages/<language>/fix-strategies.md`" (replaces `references/providers/<lang>.md`)

### Known Gaps Per Language

#### Go
- **Container limitation:** kantra v0.9.0-alpha.6 has gopls but no Go toolchain. `go.referenced` rules fail in container mode. `cmd/test` already uses `--run-local` for Go. This is a kantra-side fix — we document it but don't work around it.
- **Vendor directory required:** gopls can't download modules in the container. After test generation, `go mod tidy && go mod vendor` is mandatory. Currently mentioned in docs but should be in `config.json` as the `dependency_resolution.command`.
- **`go.dependency` untested end-to-end.** Construct code handles it, but no pipeline run has validated it.

#### Node.js / TypeScript
- **No `nodejs.dependency` kantra provider.** Detecting package.json dependency versions requires `builtin.json` with XPath, not a first-class provider. Must be documented in `languages/nodejs/condition-types.md`.
- **`npm install` always required.** Kantra's TypeScript analyzer needs `node_modules` to resolve types. Current docs say "only if needed" — should be "always".
- **Fix strategies are thin.** Current `nodejs.md` is 24 lines. Needs expansion: common failures (missing `node_modules`, unresolved types, JSX vs TSX), fix patterns.
- **Test data guide is minimal.** Java has detailed version bounds, BOM management, discontinued artifacts sections. Node.js needs equivalent depth: valid package.json structure, version ranges, TypeScript config.

#### C#
- **No `csharp.dependency` kantra provider.** Detecting `.csproj` package versions requires `builtin.xml` with XPath. Must be documented.
- **Limited location support.** Only `ALL`, `METHOD`, `FIELD`, `CLASS` (vs Java's 14). Must be clear in docs.
- **Fix strategies are thin.** Current `csharp.md` is 21 lines. Needs expansion.
- **`dotnet restore` always required.** Same vagueness as Node.js.

#### Cross-cutting
- Monolithic docs bury language-specific gotchas — fixed by the plugin split.
- Java documentation depth is 10x other languages — fixed by enriching per-language docs.

### Validation Plan: Node.js End-to-End

After the plugin structure and doc enrichment are in place:

1. **Pick a migration guide:** PatternFly 4 → PatternFly 5 (or similar React/Angular migration)
2. **Run full pipeline:** ingestion → extraction → construct → scaffold → test-gen → kantra test → validate
3. **What this validates:**
   - Pattern extraction with `provider_type: nodejs`
   - Construct generating `nodejs.referenced` conditions
   - Scaffold creating `package.json` + `src/App.tsx`
   - Test-generator writing valid TypeScript
   - Kantra running Node.js provider rules
   - Validator fix loop for Node.js failures
4. **Fix whatever breaks** and update docs accordingly

## Phases

### Phase 1: Plugin System + Gap Fixes
- Create `languages/` directory structure with 4 language plugins
- Move existing content from monolithic docs into per-language files
- Split shared content into `builtin-conditions.md` and `test-data-common.md`
- Refactor `scaffold.go` to read `config.json` from disk (with fallback)
- Enrich Node.js and C# docs to match Java's depth
- Update all 3 SKILL.md files to reference `languages/<lang>/`
- Update CLAUDE.md project structure
- Add tests for config loading

### Phase 2: Node.js Validation
- Select a real Node.js/TypeScript migration guide
- Run the pipeline end-to-end
- Fix failures
- Document findings

## File Impact Summary

| Area | Files Changed | Files Added | Files Moved |
|------|--------------|-------------|-------------|
| Go code | scaffold.go (~50 lines) | scaffold config loader (~40 lines) | — |
| Language plugins | — | 16 (4 languages x 4 files) | 4 (providers/*.md → languages/*/fix-strategies.md) |
| Agent skills | 3 SKILL.md files | 2 shared references | — |
| Docs | CLAUDE.md | — | — |
| Tests | — | config loader tests | — |
