# Deterministic Rule Verification

**Date:** 2026-05-11
**Status:** Approved
**Scope:** Language-agnostic artifact verification for extracted patterns

## Background

When the LLM generates both rules and test data, it can create internally consistent but externally invalid pairs — the test passes, but the rule targets a hallucinated FQN or non-existent artifact. The kantra test loop validates internal consistency (rule matches its test data), not external validity (rule reflects a real API change).

Stakeholder feedback (John Matthews, 2026-05-11) confirmed this is a known problem: Todd's PatternFly rules looked correct but contained noops and false positives that weren't caught until manual testing against real applications.

### Current Mitigations

- Maven Central pre-check for `java.dependency` patterns (verifies artifact exists, flags non-semver versions)
- 8-item extraction checklist constraining LLM pattern extraction
- Source FQN verification prompt ("would this FQN appear in unmigrated code?")
- Per-language example banks grounding extraction in real patterns

### Gap

No deterministic check verifies that a `*.referenced` pattern's FQN actually exists in the published source library. The LLM can hallucinate a plausible-looking FQN that passes all current checks.

## Proposal

Add a deterministic verification layer that validates extracted patterns against real published library artifacts. Three phases, implemented incrementally:

| Phase | Name | What It Validates | When It Runs |
|-------|------|-------------------|--------------|
| 1 | Artifact API Verification | FQN exists in source library | Post-extraction, before construct |
| 2 | Source+Target Diff Verification | FQN changed between versions | Post-extraction, before construct |
| 3 | Compilation-Based Verification | Test code compiles against real deps | Post-test-generation |

Phase 1 is the immediate implementation target. Phases 2 and 3 are documented for future work.

## Phase 1: Artifact API Verification

### Pipeline Integration

```
extract → merge-patterns → **verify** → construct → scaffold → test-gen → test → fix → stamp → report
```

The orchestrator runs `cmd/verify` after `merge-patterns` and before `construct`. Verification is informational, not blocking — unverified patterns are still constructed into rules but flagged.

### New CLI Command

```bash
go run ./cmd/verify --patterns patterns.json --language java --output verify-results.json
```

**Input:** `patterns.json` (merged output of rule-writer agents)
**Output:** `verify-results.json` with per-pattern verification status

### Architecture

```
internal/verify/
├── verify.go       # Verifier interface, orchestration, registry selection
├── registry.go     # Package registry abstraction (download, list, cache)
├── java.go         # Java verifier: Maven Central + jar tf
├── go.go           # Go verifier: proxy.golang.org / go doc
├── nodejs.go       # Node.js verifier: npm registry + exports/types
├── csharp.go       # C# verifier: NuGet API + assembly inspection
└── result.go       # Result types
```

### Verifier Interface

```go
type Verifier interface {
    Verify(pattern Pattern) (Result, error)
    Language() string
}

type Result struct {
    PatternID    string
    SourceFQN    string
    Status       string   // "verified", "not_found", "registry_offline", "skipped"
    Evidence     string   // e.g., "found in httpcomponents-client-4.5.14.jar"
    Suggestions  []string // close matches if not_found
}
```

### Verification Flow (Java — Tier 1)

1. Read pattern from `patterns.json`
2. For `*.dependency` patterns: already verified by existing Maven pre-check → status = `verified` (inherits pre-check result; patterns flagged as `suspected_kantra_limitation` by Maven pre-check retain that flag)
3. For `*.referenced` patterns:
   a. Read `source_artifact` metadata from pattern (groupId, artifactId, version)
   b. Download JAR from Maven Central: `https://repo1.maven.org/maven2/{group}/{artifact}/{version}/{artifact}-{version}.jar`
   c. Run `jar tf <jar>` to list all `.class` entries
   d. Convert FQN to path: `org.apache.http.client.HttpClient` → `org/apache/http/client/HttpClient.class`
   e. Check if path exists in class listing
   f. If not found, search for close matches (same class name, different package) → populate `Suggestions`
4. Cache JAR class listings in `<workspace>/verify-cache/` keyed by `groupId:artifactId:version`

### Artifact Resolution

The rule-writer must emit `source_artifact` metadata for each pattern:

```json
{
  "source_fqn": "org.apache.http.client.HttpClient",
  "source_artifact": {
    "groupId": "org.apache.httpcomponents",
    "artifactId": "httpclient",
    "version": "4.5.14"
  },
  "condition_type": "java.referenced",
  ...
}
```

This is populated from the migration guide context (which already names the source library and version). The `source_artifact` field is optional — if absent, the verifier skips with status `skipped`.

### patterns.json Schema Change

Add optional `source_artifact` to the pattern schema:

```json
{
  "source_artifact": {
    "type": "object",
    "properties": {
      "groupId": { "type": "string" },
      "artifactId": { "type": "string" },
      "version": { "type": "string" }
    },
    "required": ["groupId", "artifactId", "version"]
  }
}
```

### Result Handling

**Rule labeling:** Rules constructed from verified patterns get stamped:
- `konveyor.io/source-verified=true` — pattern confirmed in published source library
- `konveyor.io/source-verified=false` — pattern not found in published source library
- No label if verification didn't run (offline, no source_artifact, unsupported language)

**Report additions:**
```yaml
verification:
  verified: 15
  unverified: 3
  skipped: 2
  unverified_rules:
    - rule_id: spring-boot-3-to-4-00012
      source_fqn: org.example.NonExistent
      reason: "not found in spring-boot-autoconfigure-3.2.4.jar"
```

**Pipeline log:** Each verification result logged via `cmd/log` with evidence.

### Offline / Air-Gapped Behavior

If the package registry is unreachable (network timeout, DNS failure), the verifier:
1. Logs a warning: "Maven Central unreachable, skipping artifact verification"
2. Sets all patterns to status `skipped` with reason `registry_offline`
3. Pipeline continues normally — no verification labels applied
4. Report notes that verification was skipped

No failure, no blocking. The verification layer is additive.

### Caching

Downloaded JARs and their class listings are cached in `<workspace>/verify-cache/`:

```
verify-cache/
├── org.apache.httpcomponents/
│   └── httpclient/
│       └── 4.5.14/
│           ├── httpclient-4.5.14.jar
│           └── classes.txt      # output of jar tf, one entry per line
```

Cache is workspace-scoped (not global). Multiple patterns referencing the same library reuse the cached listing.

## Phase 2: Source+Target Diff Verification (Future)

### Concept

Download both source and target version artifacts. Compare API surfaces. Every rule must correspond to something that exists in source but is absent/changed in target.

### What It Catches

- Rules where the FQN exists in both versions (no actual migration needed — noop rule)
- Rules where the FQN doesn't exist in either version (hallucinated entirely)
- Auto-discovery of migration patterns the guide missed (classes removed in target but not mentioned in guide)

### Per-Language Implementation

| Language | Source | Target | Diff Method |
|----------|--------|--------|-------------|
| Java | Source-version JAR | Target-version JAR | Compare `jar tf` class listings |
| Go | Source module version | Target module version | Compare exported symbols via `go doc` |
| Node.js | Source npm package | Target npm package | Compare `exports` in `package.json` + `.d.ts` |
| C# | Source NuGet package | Target NuGet package | Compare assembly type lists |

### Artifact Resolution

Requires `target_artifact` in addition to `source_artifact` in patterns.json:

```json
{
  "source_artifact": { "groupId": "...", "artifactId": "...", "version": "4.5.14" },
  "target_artifact": { "groupId": "...", "artifactId": "...", "version": "5.4.0" }
}
```

### Result Statuses

- `confirmed_removal` — exists in source, absent in target
- `confirmed_move` — absent in source artifact, found in target under different package
- `exists_in_both` — FQN unchanged between versions (potential noop)
- `absent_in_both` — hallucinated entirely

### Implementation Notes

- Heavier resource usage (two artifacts per dependency)
- Rename/move detection requires fuzzy matching (same class name, different package)
- Could surface auto-discovered patterns as suggestions to the user

## Phase 3: Compilation-Based Verification (Future)

### Concept

After test data generation, compile the test code against the real source-version dependency (downloaded from the package registry). If it compiles, the APIs referenced in the test code actually exist. If it doesn't, the test data is invalid.

### When It Runs

Post-test-generation, before kantra test execution:

```
test-gen → **compile-verify** → test → fix → stamp → report
```

### Per-Language Implementation

| Language | Build Command | What It Validates |
|----------|--------------|-------------------|
| Java | `mvn compile` with real dependency in pom.xml | Import paths, class names, method signatures |
| Go | `go build ./...` with real module in go.mod | Import paths, function names, type names |
| Node.js | `npm install` + `tsc --noEmit` with real package | Import names, type signatures |
| C# | `dotnet build` with real NuGet package | Using directives, class names, method signatures |

### Artifact Resolution

Uses the same `source_artifact` metadata from patterns.json. The verifier replaces any LLM-generated dependency versions in the build file (pom.xml, go.mod, package.json) with the real source version before compiling.

### Result Handling

- Compilation success → test data verified
- Compilation failure → parse errors, map to specific rules, flag as `test_data_invalid`
- Does NOT attempt to fix — just reports which rules have invalid test data

### Implementation Notes

- Requires language toolchains installed (JDK, Go, Node.js, .NET SDK)
- Compilation can be slow for large test suites
- May conflict with existing test data that intentionally uses minimal stubs
- Consider running in parallel per test group

## Per-Language Verifier Summary

| Language | Phase 1: Registry + Listing | Phase 2: Diff | Phase 3: Compile |
|----------|----------------------------|---------------|------------------|
| Java | Maven Central + `jar tf` | JAR class diff | `mvn compile` |
| Go | proxy.golang.org + module zip | Symbol diff via `go doc` | `go build` |
| Node.js | npm registry + exports/`.d.ts` | Export diff | `tsc --noEmit` |
| C# | NuGet API + assembly types | Assembly diff | `dotnet build` |

Implementation order: Java first (clearest tooling), then Go, Node.js, C# — matching the existing multi-language priority.

## Testing Plan

### Phase 1 Tests

- **Unit tests for Java verifier:**
  - FQN-to-path conversion (`org.apache.http.client.HttpClient` → `org/apache/http/client/HttpClient.class`)
  - Class listing parsing (mock `jar tf` output)
  - Cache hit/miss behavior
  - Offline/timeout handling (registry unreachable → skip gracefully)
  - Close-match suggestions when FQN not found

- **Unit tests for result handling:**
  - Verification label stamping
  - Report generation with verification stats
  - Patterns with missing `source_artifact` → status `skipped`

- **Integration test:**
  - Run `cmd/verify` against a known patterns.json with real Maven Central artifacts
  - Verify that a known-good FQN returns `verified`
  - Verify that a hallucinated FQN returns `not_found`

### End-to-End Validation

- Run the full pipeline on httpcomponents-client 4→5 migration
- Compare pass rates with and without verification
- Check that unverified rules correspond to actual false positives

## Security Implications

- Downloads artifacts from public registries (Maven Central, npm, NuGet, proxy.golang.org) — standard supply chain trust model
- JAR files are only inspected (`jar tf`), never executed — no code execution risk
- Cache directory is workspace-scoped, not shared — no cross-workspace contamination
- Network requests use HTTPS only
- No credentials required for public registry queries
