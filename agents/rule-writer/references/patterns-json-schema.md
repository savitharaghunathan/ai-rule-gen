# patterns.json Schema

This is the contract between the agent (which extracts migration patterns) and `go run ./cmd/construct` (which builds rule YAML files).

## Format

```json
{
  "source": "spring-boot-3",
  "target": "spring-boot-4",
  "language": "java",
  "patterns": [
    {
      "source_pattern": "javax.servlet.http.HttpServlet",
      "target_pattern": "jakarta.servlet.http.HttpServlet",
      "source_fqn": "javax.servlet.http.HttpServlet",
      "location_type": "IMPORT",
      "alternative_fqns": ["javax.servlet.Servlet"],
      "rationale": "javax.servlet renamed to jakarta.servlet in Jakarta EE 9+",
      "complexity": "trivial",
      "category": "mandatory",
      "concern": "web",
      "provider_type": "java",
      "documentation_url": "https://jakarta.ee/specifications/servlet/"
    },
    {
      "source_pattern": "spring-boot-starter-undertow",
      "dependency_name": "org.springframework.boot.spring-boot-starter-undertow",
      "upper_bound": "4.0.0",
      "rationale": "Undertow support removed in Spring Boot 4",
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
      "rationale": "Classic uber-jar loader removed in Spring Boot 4",
      "complexity": "trivial",
      "category": "mandatory",
      "concern": "build",
      "provider_type": "builtin"
    }
  ]
}
```

## Top-Level Fields

| Field | Required | Description |
|---|---|---|
| `source` | yes | Source technology (e.g., `spring-boot-3`, `java-ee-8`, `go-non-fips`) |
| `target` | yes | Target technology (e.g., `spring-boot-4`, `jakarta-ee-9`, `go-fips`) |
| `language` | no | Programming language: `java`, `go`, `nodejs`, `csharp`. Auto-detected from provider_type if omitted |
| `patterns` | yes | Array of MigrationPattern objects |

## Pattern Fields

| Field | Required | Description |
|---|---|---|
| `source_pattern` | yes | What to detect in the source code (API, annotation, class, config, etc.) |
| `target_pattern` | no | The replacement in the target technology (null if simply removed) |
| `source_fqn` | no | Fully qualified name for matching (e.g., `javax.ejb.Stateless`). Used as the `pattern` field in the rule condition |
| `location_type` | no | Where this appears in code. Java: `TYPE`, `INHERITANCE`, `METHOD_CALL`, `CONSTRUCTOR_CALL`, `ANNOTATION`, `IMPLEMENTS_TYPE`, `ENUM`, `RETURN_TYPE`, `IMPORT`, `VARIABLE_DECLARATION`, `PACKAGE`, `FIELD`, `METHOD`, `CLASS`. C#: `ALL`, `METHOD`, `FIELD`, `CLASS` |
| `alternative_fqns` | no | Alternative FQNs for the same migration (creates `or` combinator) |
| `rationale` | yes | Why this migration is needed |
| `complexity` | yes | One of: `trivial`, `low`, `medium`, `high`, `expert` |
| `category` | yes | One of: `mandatory`, `optional`, `potential` |
| `concern` | no | Grouping key (e.g., `web`, `security`, `config`). Rules with the same concern go in the same YAML file |
| `provider_type` | no | One of: `java`, `go`, `nodejs`, `csharp`, `builtin`. Determines condition type |
| `file_pattern` | no | Go regex restricting which files `builtin.filecontent` searches (e.g., `.*\\.properties`, `application.*\\.yml`). Must be valid Go regex — do NOT use glob syntax (`*.properties` is invalid; use `.*\\.properties`) |
| `example_before` | no | Short code example showing the source pattern |
| `example_after` | no | Short code example showing the target pattern |
| `documentation_url` | no | URL to relevant migration documentation |
| `message` | no | Custom message text. If empty, auto-generated from `source_pattern: rationale` |
| `dependency_name` | no | Maven/Go coordinate as `groupId.artifactId`. When set, produces a `*.dependency` condition |
| `upper_bound` | no* | Version upper bound (exclusive) for dependency conditions. *At least one bound required when `dependency_name` is set |
| `lower_bound` | no* | Version lower bound (inclusive) for dependency conditions. *At least one bound required when `dependency_name` is set |
| `xpath` | no | XPath expression for `builtin.xml` conditions. When set, produces a `builtin.xml` condition |
| `namespaces` | no | Namespace map for XPath (e.g., `{"m": "http://maven.apache.org/POM/4.0.0"}`) |
| `xpath_filepaths` | no | File paths to restrict XPath matching (e.g., `["pom.xml"]`) |

**Minimum required fields per pattern:** `source_pattern`, `rationale`, `complexity`, `category`.

**Response format:** The extraction output must be valid JSON — no explanations, no markdown fences.

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

5. **Initial labels** — Adds 5 labels: `source=`, `target=`, `generated-by=ai-rule-gen`, `test-result=untested`, `review=unreviewed`

6. **Description** — Truncated from `rationale` (max 120 chars)

7. **Message** — Uses `message` if provided, otherwise `source_pattern: rationale`

8. **Links** — Creates a link from `documentation_url` if provided (title: "Migration Documentation")

9. **Grouping** — Groups rules into files by `concern` (empty concern → `general.yaml`)

10. **Ruleset** — Writes `ruleset.yaml` with name `<target>/<source>`

11. **Validation** — Validates all rules before writing. Fails if any validation errors.

## Metadata Auto-Detection

If `source` and `target` are not known, the agent can auto-detect them from the migration guide content. The detection should return a JSON object:

```json
{"source": "...", "target": "...", "language": "..."}
```

Use lowercase, hyphenated names (e.g., `spring-boot-3` not `Spring Boot 3`).

## Guidelines for Pattern Extraction

- Extract EVERY migration pattern found in the guide — API, annotation, config, dependency, and build changes
- One pattern per distinct change — don't combine unrelated changes
- Use specific FQNs — `javax.ejb.Stateless` not `javax.ejb.*`
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
| Method removed or renamed | `*.referenced` | `source_fqn.method` + METHOD_CALL |
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

### Common extraction mistakes

0. **Wrong version bounds for `*.dependency`** — Choose bounds from the *semantics* of the migration, not from artifact version knowledge: (a) If the artifact is **removed or renamed** (a different artifact replaces it), use `lower_bound: 0.0.0` with no `upper_bound` — any version of the old artifact is a problem, so the artifact's presence is the signal. (b) If the **same artifact** continues to exist but changes behavior across source→target versions, use `upper_bound` equal to the framework's target version. Rule: `upper_bound` = target framework version is only valid for the framework's own artifacts — never apply it to third-party libraries (Hibernate, Elasticsearch, Spock, etc.) that have independent version numbering.

1. **Skipping dependency renames** — A renamed artifact is a `*.dependency` pattern on the old name
2. **Skipping feature removals** — If a removed feature had a dedicated dependency, detect it via `*.dependency`. "Removed" always means detectable
3. **Labeling removed integrations as "informational"** — If support for a library/framework is dropped, detect the library's dependency and warn. "External leadership change" or "not yet compatible" still means the user's code is affected
4. **Treating tables as purely informational** — Rename/removal tables often contain actionable dependency or API changes buried in individual rows
5. **Merging too aggressively** — Each distinct rename/removal is its own pattern, even when listed together in one section
6. **Silently skipping sections** — Every section must either produce patterns or have an explicit skip reason
7. **Missing the lead paragraph** — The first paragraph of a section often states the biggest change (e.g., an entire package rename). Don't jump straight to bullet points and miss the foundational change
8. **Claiming "not detectable" without trying** — If a behavioral change affects users of a specific class or dependency, detect that class/dependency and warn. Detect the affected artifact, not the missing fix
9. **Skipping behavioral default changes** — When a default flips (e.g., feature enabled→disabled, auto-config provided→removed), detect the affected class or dependency as a `potential` pattern
