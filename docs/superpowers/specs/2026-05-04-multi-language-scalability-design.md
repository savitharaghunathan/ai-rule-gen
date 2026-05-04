# Multi-Language Scalability Design

Date: 2026-05-04

## Problem

The ai-rule-gen pipeline has been validated end-to-end for Java only. It supports 4 languages (Java, Go, Node.js, C#) at the code level, but language-specific knowledge is scattered across hardcoded Go maps and monolithic agent reference docs. Adding a new language requires changes in 5 places across 3 directories. Non-Java languages have thin documentation that would cause agent failures during extraction, test generation, and fix loops.

## Goals

1. Make each language a self-contained plugin directory (zero Go code changes for new languages after initial refactor)
2. Fix documentation gaps for Go, Node.js, and C# so the pipeline is ready when someone brings a migration guide
3. Validate the pipeline end-to-end for Node.js/TypeScript with a real migration guide

## Non-Goals

- Adding new languages beyond the existing 4 (the plugin system enables this but we're not doing it now)
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
  python/
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

### Current Provider Capabilities (as of 2026-05-04)

Based on analyzer-lsp and kantra source:

| Provider | referenced | dependency | Locations | Container | Notes |
|----------|-----------|-----------|-----------|-----------|-------|
| java | YES | YES | TYPE, ANNOTATION, METHOD_CALL, CONSTRUCTOR_CALL, INHERITANCE, IMPLEMENTS_TYPE, RETURN_TYPE, ENUM | YES | Full implementation via java-external-provider |
| go | YES | YES | N/A (LSP-based) | YES | Has dedicated golang-dependency-provider; `kantra test` defaults to `--run-local=true` |
| python | YES | NO | N/A (LSP-based) | YES | `GetDependencies()` returns nil; SourceOnlyAnalysisMode |
| nodejs | YES | NO | N/A (LSP-based) | YES | `GetDependencies()` stub returns nil (`c7ea23c`); SourceOnlyAnalysisMode |
| csharp | YES | NO | ALL, METHOD | YES | Separate Rust provider; gRPC dependency interface defined but not implemented |
| builtin | YES | N/A | N/A | YES | filecontent, xml, json, file, hasTags — language-agnostic |

### Remaining Gaps

#### Node.js / TypeScript

| Gap | Resolution | Phase |
|-----|-----------|-------|
| No `nodejs.dependency` — `GetDependencies()` is a stub returning nil | Document `builtin.json` with XPath on `package.json` as workaround in `languages/nodejs/condition-types.md` | Phase 1 |
| `npm install` always required but docs say "only if needed" | Fix to "always required" in `config.json` and `languages/nodejs/test-data-guide.md` | Phase 1 |
| Fix strategies are thin (24 lines) | Expand with: missing `node_modules`, unresolved types, JSX vs TSX, common TypeScript errors | Phase 1 |
| Test data guide is minimal | Enrich with: valid `package.json` structure, version ranges, `tsconfig.json` requirements, `node_modules` resolution | Phase 1 |
| Untested end-to-end | Run PatternFly or similar migration guide through full pipeline | Phase 2 |

#### C#

| Gap | Resolution | Phase |
|-----|-----------|-------|
| No `csharp.dependency` — gRPC interface defined but not implemented in Rust provider | Document `builtin.xml` with XPath on `.csproj` as workaround in `languages/csharp/condition-types.md` | Phase 1 |
| Location support is only ALL and METHOD (vs Java's 8) | Document clearly in `languages/csharp/condition-types.md` | Phase 1 |
| Fix strategies are thin (21 lines) | Expand with: NuGet resolution failures, namespace vs type resolution, `.csproj` format issues | Phase 1 |
| `dotnet restore` always required but docs are vague | Fix to "always required" in `config.json` and `languages/csharp/test-data-guide.md` | Phase 1 |

#### Python (new language)

| Gap | Resolution | Phase |
|-----|-----------|-------|
| No `languages/python/` plugin directory | Python external provider exists in analyzer-lsp (`1f32fbe`), kantra has Python language server fix (`cd058fb`). Create plugin with config.json + docs | Phase 1 |
| No `python.dependency` — `GetDependencies()` returns nil | Document `builtin.filecontent` on `requirements.txt`/`pyproject.toml` as workaround | Phase 1 |

#### Cross-cutting

| Gap | Resolution | Phase |
|-----|-----------|-------|
| Monolithic docs bury language-specific gotchas | Fixed by the plugin split — each language gets focused docs | Phase 1 |
| Java documentation depth is 10x other languages | Fixed by enriching per-language docs during plugin creation | Phase 1 |

### Resolved Gaps (no action needed)

| Previously listed gap | Why it's resolved |
|----------------------|-------------------|
| Go container limitation — `go.referenced` fails in container | `kantra test` now defaults to `--run-local=true` (`586d9a4`). Container mode is no longer the default path. |
| Go vendor directory only mentioned in prose | Covered by `config.json` `dependency_resolution.command` — standard part of plugin setup, not a gap |
| `go.dependency` untested end-to-end | `go.dependency` is fully implemented with `golang-dependency-provider`. Works in kantra test. Validated when someone runs a Go guide — not a gap, just untested. |

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
| Language plugins | — | 20 (5 languages x 4 files) | 4 (providers/*.md → languages/*/fix-strategies.md) |
| Agent skills | 3 SKILL.md files | 2 shared references | — |
| Docs | CLAUDE.md | — | — |
| Tests | — | config loader tests | — |
