# patterns.json Schema

This is the contract between the agent (which extracts migration patterns) and `go run ./cmd/construct` (which builds rule YAML files).

## Contents

- Format
- Top-Level Fields
- Pattern Fields
- Source Artifact Resolution
- What `go run ./cmd/construct` Does With This
- Metadata Auto-Detection
- Guidelines for Pattern Extraction
- What Counts as an Extractable Pattern

## Format

```json
{
  "sources": ["framework-v3", "framework"],
  "targets": ["framework-v4", "framework"],
  "language": "java",
  "patterns": [
    {
      "source_pattern": "com.example.old.MyClass",
      "target_pattern": "com.example.new.MyClass",
      "source_fqn": "com.example.old.MyClass",
      "location_type": "IMPORT",
      "source_artifact": {
        "group_id": "com.example",
        "artifact_id": "example-core",
        "version": "3.5.0"
      },
      "rationale": "com.example.old package renamed to com.example.new in v4",
      "complexity": "trivial",
      "category": "mandatory",
      "concern": "core",
      "provider_type": "java",
      "documentation_url": "https://example.com/migration-guide"
    },
    {
      "source_pattern": "old-starter-library",
      "dependency_name": "com.example.old-starter-library",
      "upper_bound": "4.0.0",
      "rationale": "Library support removed in v4",
      "complexity": "high",
      "category": "mandatory",
      "concern": "dependencies",
      "provider_type": "java"
    },
    {
      "source_pattern": "loaderImplementation CLASSIC",
      "xpath": "//*[local-name()='loaderImplementation']",
      "namespaces": {"m": "http://maven.apache.org/POM/4.0.0"},
      "xpath_filepaths": ["pom.xml"],
      "rationale": "Classic loader removed in v4",
      "complexity": "trivial",
      "category": "mandatory",
      "concern": "build",
      "provider_type": "builtin"
    }
  ]
}
```

The examples above use Java, but the schema works for all languages — substitute `provider_type` and language-specific fields as needed.

## Top-Level Fields

| Field | Required | Description |
|---|---|---|
| `sources` | yes | Array of source technology labels (e.g., `["spring-boot3", "spring-boot"]`). Each becomes a `konveyor.io/source=` label on every rule. First element is the "primary" used for naming (rule IDs, directory, ruleset name) |
| `targets` | yes | Array of target technology labels (e.g., `["spring-boot4", "spring-boot"]`). Each becomes a `konveyor.io/target=` label on every rule. First element is the "primary" |
| `language` | no | Programming language: `java`, `go`, `nodejs`, `csharp`. Auto-detected from provider_type if omitted |
| `patterns` | yes | Array of MigrationPattern objects |

## Pattern Fields

| Field | Required | Description |
|---|---|---|
| `source_pattern` | yes | What to detect in the source code (API, annotation, class, config, etc.) |
| `target_pattern` | no | The replacement in the target technology (null if simply removed) |
| `source_fqn` | no | Fully qualified name for matching (e.g., `javax.ejb.Stateless`). Used as the `pattern` field in the rule condition |
| `location_type` | no | Where this appears in code. Java: `TYPE`, `INHERITANCE`, `METHOD_CALL`, `CONSTRUCTOR_CALL`, `ANNOTATION`, `IMPLEMENTS_TYPE`, `ENUM`, `RETURN_TYPE`, `IMPORT`, `VARIABLE_DECLARATION`, `PACKAGE`, `FIELD`, `FIELD_DECLARATION`, `METHOD`, `CLASS`. C#: `ALL`, `METHOD`, `FIELD`, `CLASS`. Note: `FIELD` and `FIELD_DECLARATION` are aliases — both match field declarations by type, NOT static field access |
| `alternative_fqns` | no | Alternative FQNs for the same migration (creates `or` combinator) |
| `rationale` | yes | Short summary of what changed (becomes the rule `description`). One sentence, max ~15 words. State the change, not the fix — e.g., "Removed in 5.x" or "Renamed to X". Migration guidance goes in `message`. |
| `complexity` | yes | One of: `trivial`, `low`, `medium`, `high`, `expert` |
| `category` | yes | One of: `mandatory`, `optional`, `potential` |
| `concern` | no | Grouping key (e.g., `web`, `security`, `config`). Rules with the same concern go in the same YAML file |
| `provider_type` | no | One of: `java`, `go`, `nodejs`, `csharp`, `builtin`. Determines condition type |
| `file_pattern` | no | Go regex restricting which files `builtin.filecontent` searches (e.g., `.*\\.properties`, `application.*\\.yml`). Must be valid Go regex — do NOT use glob syntax (`*.properties` is invalid; use `.*\\.properties`) |
| `example_before` | no | Short code example showing the source pattern |
| `example_after` | no | Short code example showing the target pattern |
| `documentation_url` | recommended | URL to the migration guide section or relevant documentation. Always populate this — construct emits it as a `links:` entry in the rule YAML so users can find the original guidance |
| `message` | recommended | Detailed migration guidance shown to the user (what to replace with, caveats, behavioral differences). If empty, auto-generated from `source_pattern: rationale`. Always provide when the fix is non-obvious. |
| `dependency_name` | no | Package coordinate in dot notation (e.g., `groupId.artifactId` for Java/Maven, `module/path` for Go). When set, produces a `*.dependency` condition |
| `upper_bound` | no* | Version upper bound (exclusive) for dependency conditions. *At least one bound required when `dependency_name` is set |
| `lower_bound` | no* | Version lower bound (inclusive) for dependency conditions. *At least one bound required when `dependency_name` is set |
| `xpath` | no | XPath expression for `builtin.xml` conditions. When set, produces a `builtin.xml` condition |
| `namespaces` | no | Namespace map for XPath (e.g., `{"m": "http://maven.apache.org/POM/4.0.0"}`) |
| `xpath_filepaths` | no | File paths to restrict XPath matching (e.g., `["pom.xml"]`) |
| `source_artifact` | no | Package coordinates of the library that publishes the `source_fqn`. Object with `group_id`, `artifact_id`, `version`. Emit for all `*.referenced` patterns when the source library and version are known from the guide. The verifier downloads this artifact and checks that `source_fqn` exists in it. For METHOD_CALL patterns the verifier checks the class component (e.g., verifies `org.apache.http.HttpResponse` for `org.apache.http.HttpResponse.getStatusLine`). For PACKAGE patterns it checks that at least one class exists under the package prefix |

**Minimum required fields per pattern:** `source_pattern`, `rationale`, `complexity`, `category`.

**Response format:** The extraction output must be valid JSON — no explanations, no markdown fences.

## Source Artifact Resolution

For `*.referenced` patterns (`java.referenced`, `go.referenced`, etc.), emit `source_artifact` so the deterministic verifier can confirm the FQN exists in the published library. This catches hallucinated FQNs before rules are constructed.

**When to emit:**
- `*.referenced` patterns: ALWAYS when the source library and version are known from the guide context
- `*.dependency` patterns: NOT needed (already verified by registry pre-check)
- `builtin.*` patterns: NOT applicable

**How to determine coordinates:**
1. Read the migration guide for the source framework version (e.g., "migrating from v3.5.x" → version `3.5.0`)
2. Map the FQN's package to the correct artifact in the language's package registry
3. Use the **source** version (the version being migrated FROM), not the target version

See `references/languages/<language>/instructions.md` for language-specific coordinate format and examples.

**If unsure:** Omit `source_artifact` — the verifier skips gracefully with status `skipped`. A missing `source_artifact` is better than a wrong one.

## What `go run ./cmd/construct` Does With This

The CLI handles all mechanical transformation:

1. **Condition type selection** (checked in order):
   - If `dependency_name` is set → `java.dependency` (or `go.dependency` if `provider_type: go`)
   - If `xpath` is set → `builtin.xml` with namespaces and filepaths
   - If `provider_type: java` → `java.referenced` with `location_type`
   - If `provider_type: go` → `go.referenced`
   - If `provider_type: nodejs` → `nodejs.referenced`
   - If `provider_type: csharp` → `csharp.referenced` with `location_type`
   - If `provider_type: builtin` → `builtin.filecontent` with `file_pattern`
   - Default (no provider + no location): `builtin.filecontent`
   - Default (no provider + has location): `java.referenced`

2. **Or combinators** — If `alternative_fqns` is non-empty, wraps all FQNs (primary + alternatives) in an `or` condition

3. **Effort conversion** — Maps `complexity` to numeric effort: trivial=1, low=3, medium=5, high=7, expert=9 (default=5)

4. **Rule ID generation** — Creates sequential IDs: `<source>-to-<target>-00010`, `00020`, `00030`, etc. (increments of 10)

5. **Initial labels** — Adds one `konveyor.io/source=` label per source, one `konveyor.io/target=` label per target, and `konveyor.io/generated-by=ai-rule-gen`

6. **Description** — Uses `rationale` as-is. Keep it short — the description appears in summary views. Migration guidance belongs in `message`.

7. **Message** — Uses `message` if provided, otherwise `source_pattern: rationale`. Use `message` for detailed migration guidance (what to replace it with, caveats, behavioral differences).

8. **Links** — Creates a link from `documentation_url` if provided (title: "Migration Documentation")

9. **Grouping** — Groups rules into files by `concern` (empty concern → `general.yaml`)

10. **Ruleset** — Writes `ruleset.yaml` with name `<target>/<source>`

11. **Validation** — Validates all rules before writing. Fails if any validation errors.

## Metadata Auto-Detection

If `sources` and `targets` are not known, the agent can auto-detect them from the migration guide content. The detection should return a JSON object:

```json
{"sources": ["framework-v3", "framework"], "targets": ["framework-v4", "framework"], "language": "java"}
```

Use lowercase, hyphenated names (e.g., `spring-boot3` not `Spring Boot 3`, `express4` not `Express 4`). Include both a version-specific label and a generic label when appropriate (following Konveyor rulesets conventions).

## Guidelines for Pattern Extraction

- Extract EVERY migration pattern found in the guide — API, annotation, config, dependency, and build changes
- One pattern per distinct change — don't combine unrelated changes
- **Package-level vs per-class rules** — When an entire package/module is renamed or removed, create a SINGLE rule matching the old package with `location_type: PACKAGE`, not one rule per class. Per-class rules are only needed when individual classes within a package have different migration paths. If the migration is "everything under `com.foo` moves to `com.bar`", one package-level rule is correct and sufficient. **Asterisk rule for PACKAGE patterns:** append `*` to the `source_fqn` when target classes live in subpackages. `org.apache.http` matches types directly in that package (e.g., `org.apache.http.HttpResponse`), but `com.fasterxml.jackson*` is needed to match types in subpackages like `com.fasterxml.jackson.databind.ObjectMapper`. When in doubt, always append `*`.
- Use specific FQNs — not wildcard patterns — UNLESS the entire package is being renamed/removed (see above)
- **`source_fqn` must be the OLD (pre-migration) path** — this is what the rule matches in user code. Never use the target/new path. The migration guide often shows both; use the "Before" path. Verification: "Would this FQN appear in code that has NOT been migrated?"
- Set `provider_type` — the CLI uses this to pick the right condition type
- Set `location_type` for Java/C# — critical for accurate matching
- For removed/renamed dependencies, use `dependency_name` + `upper_bound` — don't try to detect dependencies via `builtin.filecontent`
- For POM/XML structure changes, use `xpath` + `namespaces` + `xpath_filepaths`
- Use `alternative_fqns` for APIs with multiple entry points to the same migration
- Group by `concern` — related changes together
- Deduplicate — same `source_fqn` or `dependency_name` should only appear once

## What Counts as an Extractable Pattern

Any guide item where a user's code, config, or build file could be automatically detected is extractable. The categories are:

| Change Type | Condition Type | Key Fields |
|---|---|---|
| API/class/interface moved to a new package | `*.referenced` | `source_fqn` + `location_type` |
| Annotation renamed or moved | `*.referenced` | `source_fqn` + ANNOTATION |
| Method removed or renamed | `*.referenced` | `source_fqn` = `package.ClassName.methodName` + METHOD_CALL. The verifier checks the class exists, not the method |
| Dependency renamed or removed | `*.dependency` | `dependency_name` + version bound |
| Feature removed that had a dedicated dependency | `*.dependency` | `dependency_name` + version bound |
| Feature/integration removed (no longer supported) | `*.dependency` | detect the removed library's artifact |
| Previously-transitive dependency now requires explicit declaration | `*.dependency` | old artifact + version bound |
| Library-wide package rename (e.g., `com.foo` → `org.foo`) | `*.referenced` | detect most-used class from old package |
| Behavioral default change affecting users of a class/dependency | `*.dependency` or `*.referenced` | detect the affected artifact, warn about new behavior |
| Test infrastructure that now requires an explicit starter | `*.dependency` | old test artifact + version bound |
| Configuration property renamed or removed | `builtin.filecontent` | regex + `file_pattern` |
| Version override property renamed or removed | `builtin.filecontent` | regex + `file_pattern` |
| Build config element removed from XML | `builtin.xml` | `xpath` + `filepaths` |
| Build config removed from non-XML files (Gradle, etc.) | `builtin.filecontent` | regex + `file_pattern` |
| Minimum language/runtime version requirement | `*.dependency` | detect the framework's core artifact with `upper_bound` at the target version — the version check on the framework artifact gates the migration |

### Common extraction mistakes

0. **Wrong version bounds for `*.dependency`** — Choose bounds from the *semantics* of the migration, not from artifact version knowledge: (a) If the artifact is **removed or renamed** (a different artifact replaces it), use `lower_bound: 0.0.0` with no `upper_bound` — any version of the old artifact is a problem, so the artifact's presence is the signal. (b) If the **same artifact** continues to exist but changes behavior across source→target versions, use `upper_bound` equal to the framework's target version. Rule: `upper_bound` = target framework version is only valid for the framework's own artifacts — never apply it to third-party libraries (Hibernate, Elasticsearch, Spock, etc.) that have independent version numbering.

   **Non-semver third-party artifacts** — Some artifacts never publish plain-semver versions. `*.dependency` is still the correct condition type — do NOT substitute `builtin.xml` or `builtin.filecontent`. Instead, always run the package registry pre-check (see `references/languages/<language>/instructions.md`) before emitting a `dependency_name` pattern. If the registry confirms no plain-semver version exists, flag the pattern as a suspected kantra limitation and include it in `suspected_kantra_limitations`. Kantra's version comparator cannot handle non-semver strings; this is an engine limitation, not a rule design problem.

1. **Skipping dependency renames** — A renamed artifact is a `*.dependency` pattern on the old name
2. **Skipping feature removals** — If a removed feature had a dedicated dependency, detect it via `*.dependency`. "Removed" always means detectable
3. **Labeling removed integrations as "informational"** — If support for a library/framework is dropped, detect the library's dependency and warn. "External leadership change" or "not yet compatible" still means the user's code is affected
4. **Treating tables as purely informational** — Rename/removal tables often contain actionable dependency or API changes buried in individual rows
5. **Merging too aggressively** — Each distinct rename/removal is its own pattern, even when listed together in one section
6. **Silently skipping sections** — Every section must either produce patterns or have an explicit skip reason
7. **Missing the lead paragraph** — The first paragraph of a section often states the biggest change (e.g., an entire package rename). Don't jump straight to bullet points and miss the foundational change
8. **Claiming "not detectable" without trying** — If a behavioral change affects users of a specific class or dependency, detect that class/dependency and warn. Detect the affected artifact, not the missing fix
9. **Skipping behavioral default changes** — When a default flips (e.g., feature enabled→disabled, auto-config provided→removed), detect the affected class or dependency as a `potential` pattern
10. **Skipping system requirements** — When the guide specifies a minimum runtime or language version, extract a `*.dependency` pattern on the framework's core artifact with `upper_bound` at the target version. The version check on the core artifact gates the entire migration and warns users still on an older framework version. Don't dismiss these as "informational" — they're the most impactful migration patterns
