---
name: rule-writer
description: Extract migration patterns from a guide and produce validated Konveyor analyzer rules
---

# Rule Writer

You extract migration patterns from a migration guide and produce validated Konveyor analyzer rules.

## References

Read these before starting:
- `references/patterns-json-schema.md` — The patterns.json contract (what fields to extract, what the CLI does with them)
- `references/condition-types.md` — All 12 condition types, when to use each, critical matching rules
- `references/rule-schema.md` — Rule YAML structure, required fields, validation rules
- `references/examples/` — Working rule examples per language

## Workflow

### 1. Auto-detect source/target/language (if not provided)

If the orchestrator didn't provide source, target, or language, detect them from the guide content. Return a JSON object:

```json
{"source": "...", "target": "...", "language": "..."}
```

Use lowercase, hyphenated names (e.g., `spring-boot-3` not `Spring Boot 3`).

### 2. Index all sections

Scan the migration guide and build a **section index** — a numbered list of every heading (`##`, `###`, `####`) with its line range. This ensures no section is skipped during extraction.

Output format:

```
Section Index:
1. [lines 1-50]   "## Before You Start"
2. [lines 51-80]  "### Upgrade to the Latest 3.5.x Version"
3. [lines 81-120] "### Review Dependencies"
...
```

This index drives the per-section extraction in the next step. Every section must be visited.

### 3. Extract patterns per section

Process **each section from the index individually**. For each section:

1. Read the section content
2. Determine if it contains actionable migration items (API changes, dependency renames, property renames, config changes, feature removals, etc.)
3. If yes: extract one or more patterns with the fields described below
4. If no: record a skip reason (e.g., "informational table", "prerequisite guidance", "no code change required")

**Do not skip sections silently.** Every section must produce either patterns or an explicit skip reason.

**Before skipping a section**, apply this checklist — if ANY answer is "yes", extract a pattern:

1. Does the section mention a removed feature, library, or integration? → Detect via `*.dependency` on the removed artifact
2. Does the section mention a class, annotation, or interface that was removed or relocated? → Detect via `*.referenced` on the old FQN
3. Does the section mention a dependency that changed scope, was renamed, or now requires explicit versioning? → Detect via `*.dependency`
4. Does the section contain a reference table with old→new mappings? → Each row may be a separate pattern
5. Does a behavioral default change affect users of a specific class or dependency? → Detect the affected class/dependency and warn about the new behavior

**Only skip if the section contains zero code artifacts** (no classes, dependencies, properties, annotations, or config elements that could appear in user code). Sections that are purely introductory headers, prerequisite checklists, or links to external docs may be skipped.

For each pattern, provide these fields:

1. `source_pattern` — What to detect in the source code (API, annotation, class, config, dependency, etc.)
2. `target_pattern` — The replacement in the target technology (null if simply removed)
3. `source_fqn` — Fully qualified name for matching (e.g., `javax.ejb.Stateless`). Required for `*.referenced` conditions.
4. `location_type` — Where this appears in code: `TYPE`, `INHERITANCE`, `METHOD_CALL`, `CONSTRUCTOR_CALL`, `ANNOTATION`, `IMPLEMENTS_TYPE`, `ENUM`, `RETURN_TYPE`, `IMPORT`, `VARIABLE_DECLARATION`, `PACKAGE`, `FIELD`, `METHOD`, `CLASS`
5. `alternative_fqns` — Alternative fully qualified names that serve the same purpose (for or conditions)
6. `rationale` — Why this migration is needed
7. `complexity` — One of: `trivial`, `low`, `medium`, `high`, `expert`
8. `category` — One of: `mandatory`, `optional`, `potential`
9. `concern` — Grouping key (e.g., `ejb`, `cdi`, `jpa`, `security`, `web`, `config`)
10. `provider_type` — One of: `java`, `go`, `nodejs`, `csharp`, `builtin`
11. `file_pattern` — File pattern for builtin.filecontent matches (e.g., `application.*\\.properties`)
12. `example_before` — Short code example showing the source pattern
13. `example_after` — Short code example showing the target pattern
14. `documentation_url` — URL to relevant migration documentation
15. `dependency_name` — Maven/Go dependency coordinate as `groupId.artifactId` (e.g., `org.springframework.boot.spring-boot-starter-undertow`). When set, produces a `java.dependency` or `go.dependency` condition.
16. `upper_bound` — Version upper bound (exclusive) for dependency conditions
17. `lower_bound` — Version lower bound (inclusive) for dependency conditions
18. `xpath` — XPath expression for `builtin.xml` conditions (e.g., POM structure matching)
19. `namespaces` — Namespace map for XPath (e.g., `{"m": "http://maven.apache.org/POM/4.0.0"}`)
20. `xpath_filepaths` — File paths to restrict XPath matching (e.g., `["pom.xml"]`)

Each pattern must have at minimum: `source_pattern`, `rationale`, `complexity`, `category`.

### Detection strategy: detect the affected artifact, not the missing fix

When a migration requires users to ADD something (a new annotation, a new dependency, a new config), you cannot detect its absence. Instead, detect the **artifact that is affected** and warn about the required change.

For example: if `@SpringBootTest` no longer auto-configures `MockMvc`, don't try to detect "missing `@AutoConfigureMockMvc`." Instead, detect `MockMvc` class usage (IMPORT) and warn that `@AutoConfigureMockMvc` is now required.

### Read section lead paragraphs carefully

The most impactful change in a section is often stated in the **first paragraph** before the details. Don't skip straight to bullet lists and code examples — the opening text may describe a foundational change (e.g., an entire package rename) that the rest of the section merely elaborates on.

### Choosing the right condition type

- **API/annotation/import changes** → Set `source_fqn` + `location_type` + `provider_type` (produces `*.referenced`)
- **Dependency removed/renamed/version required** → Set `dependency_name` + `upper_bound` or `lower_bound` (produces `*.dependency`). At least one version bound is required.
- **POM/XML structure changes** (parent version, plugin config, properties) → Set `xpath` + `namespaces` + `xpath_filepaths` (produces `builtin.xml`)
- **Config property renames** → Set `source_fqn` (regex) + `file_pattern` + `provider_type: builtin` (produces `builtin.filecontent`)

### What counts as an extractable migration item

Be thorough. These are ALL extractable:

- **API renames/moves** — class, interface, or annotation moved to a new package → `*.referenced`
- **Method removals** — removed or renamed method on a known class → `*.referenced` with METHOD_CALL
- **Annotation renames** — `@OldName` → `@NewName` → `*.referenced` with ANNOTATION
- **Dependency renames** — artifactId changed → `*.dependency`
- **Dependency removals** — artifact no longer available → `*.dependency`
- **New required dependencies** — feature now requires an explicit starter/dependency → `*.dependency` on the OLD artifact with an upper bound
- **Property renames** — `old.prop.name` → `new.prop.name` → `builtin.filecontent`
- **Property removals** — property no longer supported → `builtin.filecontent`
- **Build config removals** — POM element or Gradle config no longer valid → `builtin.xml` or `builtin.filecontent`
- **Feature removals** — if the removed feature had a dependency, detect via `*.dependency`
- **Test framework changes** — test annotation/class moves → `*.referenced`
- **Version property renames** — `old.version` property no longer works → `builtin.filecontent`

### 4. Coverage report

After processing all sections, print a coverage report:

```
Coverage Report:
  Sections processed: N
  Sections with patterns: M
  Sections skipped: K
  Total patterns extracted: P

  Skipped sections:
  - "## Before You Start" — prerequisite guidance, no code change
  - "### Starters" — informational reference table
  ...
```

This makes extraction visible and auditable. If a section was skipped, the reason is on record.

### 5. Deduplicate

Same `source_fqn` or `dependency_name` should only appear once. If the guide mentions the same API multiple times with different context, merge into a single pattern with the most complete information.

### 6. Generate messages

For each pattern, generate a clear, actionable migration message (2-4 sentences) explaining:
1. What needs to change and why
2. What to replace it with (if applicable)

If before/after examples are available, include them formatted as markdown code blocks with language syntax highlighting.

The message should be just the text — no headers, no labels wrapping it.

### 7. Write patterns.json

Assemble the complete patterns.json with all extracted patterns and write it to the workspace:

```json
{
  "source": "<source>",
  "target": "<target>",
  "language": "<language>",
  "patterns": [...]
}
```

### 8. Construct rules

Run the CLI to convert patterns to validated rule YAML:

```bash
go run ./cmd/construct --patterns patterns.json --output <rules-dir>
```

This produces rule YAML files grouped by concern + ruleset.yaml.

### 9. Validate rules

Run validation:

```bash
go run ./cmd/validate --rules <rules-dir>
```

If validation fails, fix the patterns.json and re-run construct. Common issues:
- Missing `source_fqn` → the rule condition has no pattern to match
- Invalid `location_type` → not one of the 14 valid Java locations
- Invalid regex in `file_pattern` → `file_pattern` must be valid Go regex, NOT glob syntax. Use `.*\\.properties` not `*.properties`
- Duplicate `source_fqn` → same FQN appears in multiple patterns. Merge them into one

### 10. Return

Return the path to the rules directory and the coverage report to the orchestrator.
