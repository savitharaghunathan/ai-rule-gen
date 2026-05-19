---
name: rule-writer
description: Extracts migration patterns from a guide and produces validated Konveyor analyzer rules. Use when processing guide sections to identify API changes, class relocations, and behavioral differences.
---

# Rule Writer

You extract migration patterns from a migration guide and produce validated Konveyor analyzer rules.

## Inputs

- `guide` — Path to migration guide markdown file
- `sources` — Source technology labels as a list (e.g., `["framework-v3", "framework"]`). Each becomes a `konveyor.io/source=` label. First element is the primary (used for naming). Pass "auto-detect" or omit to auto-detect from guide content.
- `targets` — Target technology labels as a list (e.g., `["framework-v4", "framework"]`). Each becomes a `konveyor.io/target=` label. First element is the primary (used for naming). Pass "auto-detect" or omit to auto-detect from guide content.
- `rules_dir` — Output directory for generated rules
- `sections` — (optional) List of sections to process, each with `heading`, `start_line`, `end_line`. When provided, only process these sections (chunk mode)
- `output_file` — (optional, default: `output/<source>-to-<target>/patterns.json`) Where to write the extracted patterns

## Returns

**Full mode** (no `sections` input):
- `source` — Detected source technology
- `target` — Detected target technology
- `patterns_count` — Number of patterns extracted
- `rules_count` — Number of rules generated
- `rules_dir` — Path to generated rules directory
- `coverage_report` — Section-level extraction coverage

**Chunk mode** (with `sections` input):
- `patterns_count` — Number of patterns extracted
- `output_file` — Path to the written patterns file
- `suspected_kantra_limitations` — (optional) List of objects `{rule_pattern, reason}` where the package registry confirmed no plain-semver version exists. The pattern is still emitted as `*.dependency`; this list signals the orchestrator to pre-classify these rules for the validator.

## Permissions

| Operation | Pattern | Purpose |
|-----------|---------|---------|
| shell | `go run ./cmd/construct *` | Build rule YAML from patterns.json |
| shell | `go run ./cmd/validate *` | Validate rule YAML structure |
| shell | `curl -s "<package-registry-url>"` | Verify artifact versioning scheme (see language instructions for registry URL) |
| read | `output/**` | Read migration guide |
| read | `agents/rule-writer/references/**` | Read condition types, schema |
| read | `agents/rule-writer/references/languages/**` | Read language-specific condition types |
| write | `patterns*.json` | Write extracted patterns |
| write | `output/**` | Write generated rule YAML and patterns |

**Do NOT use `python`, `python3`, `node`, or any scripting language runtime.** This is a Go project. Only run commands listed in this permissions table. Do not validate JSON yourself — the orchestrator runs `merge-patterns` and `construct` which validate the JSON. Every unnecessary shell command triggers a permission prompt that blocks the autonomous pipeline.

## References

Read these before starting:
- `references/patterns-json-schema.md` — The patterns.json contract (what fields to extract, what the CLI does with them)
- `references/languages/<language>/condition-types.md` — Provider-specific conditions for the detected language (java, go, nodejs, csharp, python)
- `references/builtin-conditions.md` — Language-agnostic builtin conditions (filecontent, xml, json, file, hasTags, xmlPublicID)
- `references/rule-schema.md` — Rule YAML structure, required fields, validation rules
- `references/languages/<language>/instructions.md` — Language-specific instructions (registry pre-checks, source artifact resolution, validation notes)
- `references/examples/<language>.md` — Worked extraction examples for the detected language (guide text -> checklist -> patterns.json). Read ONLY the file matching the detected language.

## Workflow

### Chunk mode vs full mode vs gap mode

If the `sections` input is provided, you are in **chunk mode**:
- Sources, targets, and language are already provided — do NOT auto-detect
- Skip step 2 (indexing) — the section list is your index
- Read only the assigned sections from the guide using line ranges (use `Read --offset <start_line> --limit <end_line - start_line>`)
- Skip steps 8-9 (construct/validate) — the orchestrator handles these
- Write patterns to `output_file` (not `patterns.json`)

If neither `sections` nor `output_file` is provided, run the full workflow below.

### 1. Auto-detect source/target/language (if not provided)

If the orchestrator didn't provide sources, targets, or language, detect them from the guide content. Return a JSON object:

```json
{"sources": ["framework-v3", "framework"], "targets": ["framework-v4", "framework"], "language": "java"}
```

Use lowercase, hyphenated names (e.g., `spring-boot3` not `Spring Boot 3`, `express4` not `Express 4`). Include both a version-specific label and a generic label when appropriate (following Konveyor rulesets conventions).

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
2. Run the **extraction checklist** below — this is the decision method, not a post-hoc check
3. If ANY checklist item is "yes": extract one or more patterns with the fields described below
4. If ALL checklist items are "no" AND the section contains zero named artifacts: record a skip with the reason "header only" or "links only"

**The checklist is the decision.** Do not pre-judge a section as "informational" or "not actionable" before running the checklist. Read the section, then evaluate each item:

#### Extraction checklist (run ALL 9 items for EVERY non-header section)

| # | Question | If yes → extract |
|---|----------|-------------------|
| 1 | Does the section mention a **removed** feature, library, or integration? | `*.dependency` on the removed artifact. "Removed" ALWAYS means detectable. |
| 2 | Does the section mention a class, annotation, or interface that was **removed or relocated**? | `*.referenced` on the old FQN |
| 3 | Does the section mention a dependency that **changed scope, was renamed, or now requires explicit versioning**? | `*.dependency` |
| 4 | Does the section contain a **reference table** with old→new mappings? | Process every row as a separate pattern — **unless** the section describes a package-level rename (see "Package-level consolidation" below), in which case emit ONE `PACKAGE` rule and only create additional rules for rows where the class name changed or the method name or signature genuinely changed |
| 5 | Does a **behavioral default change** affect users of a specific class, property, or dependency? | Detect the affected artifact, warn about new behavior (category: `potential`) |
| 6 | Does the section mention **deprecated** starters, modules, or artifacts? | Each old→new mapping is a pattern |
| 7 | Does the section **name any specific artifact** (class, dependency, property, annotation, config element, build plugin)? | If it names it, detect it |
| 8 | Does the section mention a **version requirement** for a plugin, tool, or library? | `builtin.filecontent` or `builtin.xml` depending on build file format |
| 9 | Does the section contain **before/after code examples** showing source-version and target-version API usage? | Diff the code examples line by line. Each API call, class, type, constant, or import that differs between old and new code is a separate pattern. See "Code example comparison" below. |

**Output format — verbose for ALL sections.**

Run all 9 checklist items for every section. Print the full evaluation:

- **EXTRACT** — print the full 9-item evaluation AND the verdict:
  ```
  Section: "### Liveness and Readiness Probes"
    1. Removed? no
    2. Class relocated? no
    3. Dependency changed? no
    4. Reference table? no
    5. Behavioral default? yes — health probes disabled by default
    6. Deprecated? no
    7. Names artifact? yes — management.health.probes.enabled
    8. Version requirement? no
    9. Code examples? no
    → EXTRACT: detect management.health.probes.enabled property, category: potential (items 5,7)
  ```

- **SKIP** — print the full 9-item evaluation so the decision is auditable:
  ```
  Section: "### Some Section"
    1. Removed? no
    2. Class relocated? no
    3. Dependency changed? no
    4. Reference table? no
    5. Behavioral default? no
    6. Deprecated? no
    7. Names artifact? no
    8. Version requirement? no
    9. Code examples? no
    → SKIP: no detectable artifacts
  ```

- **Header-only** — one line is enough:
  ```
  Section: "## Upgrading Web Features" → SKIP: header only
  ```

- **TABLE** — when a section contains a reference table, enumerate every row with its disposition:
  ```
  Table: "<section heading>" (<N> rows)
  Row 1: OldThing → NewThing — EXTRACT as IMPORT (class renamed)
  Row 2: OldThing → OldThing — PACKAGE covers (same name, same API)
  Row 3: OldThing.method() → NewThing.method() — EXTRACT as IMPORT (class renamed, method unchanged)
  Row 4: OldThing.foo() → OldThing.bar() — EXTRACT as METHOD_CALL (method renamed)
  ...
  ```
  Every row must appear. This prevents silent drops. For each row, decompose: check the class/type name first, then the method/member name.

- **CODE-DIFF** — when a section has before/after code examples, enumerate each API difference:
  ```
  Code diff: "## Migration steps" (source example vs target example)
    old_module.configure(raw_value) → new_module.configure(wrapped_value) — EXTRACT (function moved + parameter type changed)
    old_module.OldName → new_module.NewName — EXTRACT (renamed)
    response.old_method() → response.new_method() — EXTRACT (method/function renamed)
    OldFactory(args) → new_create_func(args) — EXTRACT (construction API replaced)
    old_const.VALUE → new_const.VALUE — EXTRACT (constant/enum moved)
  ```
  Every difference must appear. Code examples are first-class extraction sources — they often show migration changes that prose doesn't call out explicitly.

Full checklist output for extractions makes every decision auditable — the orchestrator can verify that all named artifacts got a "yes" on the relevant checklist item.

#### Skip reasons that indicate a checklist failure

If your skip reason contains any of these phrases, you answered a checklist item wrong — go back and re-evaluate:

- "informational" — check items 4, 7
- "advisory" — check items 5, 7
- "not detectable" — check item 7 (detect the *affected artifact*, not the *missing fix*)
- "naming convention" / "no old-to-new rename mapping" — check items 2, 3, 6, 7 (package renames and module restructures ARE detectable)
- "covered by other patterns" / "already covered by X section" / "covered by PACKAGE rule" — cite the exact rule_id or re-extract. **PACKAGE rules and class/method rules serve different purposes:** a PACKAGE rule tells users their imports need to change; a class-specific IMPORT or METHOD_CALL rule tells users what the replacement class or method is. Both are needed. "Covered by PACKAGE rule" is NEVER valid for skipping a class rename (e.g., `HttpResponse` → `ClassicHttpResponse`), a class replacement with a different API (e.g., `HttpPost` → `ClassicRequestBuilder`), or a method rename (e.g., `getStatusLine()` → `getCode()`). Each of these needs its own rule alongside the PACKAGE rule.
- "behavioral change" / "behavioral default change" — check item 5 (detect the affected artifact)
- "reference table of NEW items" — check items 4, 6 (new items often imply old items were restructured)
- "build plugin config" / "Gradle plugin version" — check item 8
- "no code artifacts" — check item 7 (if the section names ANY artifact, it has code artifacts)
- "describes a compatibility helper" — check items 3, 7 (detect the OLD artifact it bridges)
- "just a code example" / "code illustration" / "example usage" — check item 9 (code examples ARE extraction sources, not illustrations)

**The ONLY valid skip** is a section that contains genuinely zero named artifacts — no classes, no dependencies, no properties, no annotations, no config elements, no build plugin names. This means: pure headers (`## Upgrading Web Features`), prerequisite checklists (`### Before You Start`), or link collections (`### Review Other Release Notes`). **If a section names even one concrete artifact, it is not skippable.** When in doubt, extract — false positives are cheaper than missed migrations.

### Package-level consolidation (applies to checklist items 2 and 4)

When a migration guide says an entire package/module is renamed or removed, create a **single rule** matching the old package with `location_type: PACKAGE` — not one rule per class. This overrides the per-row instruction in checklist item 4 when the table is under a package rename.

**How to recognize a package rename section:**
- The lead paragraph says "re-import from," "moved to package," "namespace changed to," or similar
- A reference table lists old→new class mappings where every old class is under the same package prefix and every new class is under a new prefix
- The migration action for every row is the same: change the import

**Reference tables under package renames:** The table is showing examples of what moves, not listing separate migration patterns. Emit ONE `PACKAGE` rule for the old package prefix. Do NOT process each table row as a separate pattern.

**When to emit additional rules alongside a PACKAGE rule:**
- **METHOD_CALL:** when a method's **name or signature genuinely changed** (e.g., `getStatusLine()` was removed and replaced by `getCode()`, or `setRetryHandler(HttpRequestRetryHandler)` became `setRetryStrategy(HttpRequestRetryStrategy)`). **Pattern style:** prefer short method name patterns (e.g., `setRetryHandler` not `org.example.Builder.setRetryHandler`) when the method may be called on concrete subtypes or via builder chains — FQN patterns fail silently in these cases. **NEVER use FQN patterns that include inner class names** (e.g., `RequestConfig.Builder.setConnectTimeout`, `Config.Builder.setTimeout`). Kantra cannot resolve factory method return types (`RequestConfig.custom()` → `RequestConfig.Builder`), so these patterns compile but silently match nothing. Use the short method name instead — even if the method name also exists in the target API (e.g., both `RequestConfig` and `ConnectionConfig` have `setConnectTimeout`), a false positive on already-migrated code is better than a rule that never fires. Use `alternative_fqns` to create `or` conditions when you need FQN precision across a type hierarchy. See `references/languages/<language>/condition-types.md` for the full decision framework and `references/examples/<language>.md` Examples 9-11 for worked examples
- **IMPORT (different API):** when a class is **replaced by a fundamentally different class** — different name, different API surface (e.g., `SSLConnectionSocketFactory` → `ClientTlsStrategyBuilder`, `HttpEntityEnclosingRequest` → `HttpEntityContainer`). A PACKAGE rule tells users to update imports; an IMPORT rule tells users the old class is gone and the replacement has a different name and usage pattern.
- **IMPORT (class renamed):** when the **class name itself changed**, even if the API surface is similar (e.g., `BasicHttpContext` → `HttpCoreContext`, `HttpRequestBase` → `HttpUriRequestBase`, `HttpResponse` → `ClassicHttpResponse`). The PACKAGE rule fires on the old import, but the user cannot find the old class name in the new package — they need to know the new name. Emit an IMPORT rule with `source_fqn` on the old FQN and a message stating the new class name.
- **NOT** when the class kept the same name and just moved packages (e.g., `BasicNameValuePair` stayed `BasicNameValuePair`, just under `org.apache.hc.core5.http.message` instead of `org.apache.http.message` — the PACKAGE rule already covers this since the user can find the same class name in the new package)

**Decision tree:**

| Scenario | Correct output |
|---|---|
| Section says "package X moved to Y" + table of class mappings where class names stayed the same | ONE `PACKAGE` rule on old package X |
| Section says "package X moved to Y" + table includes rows where a method name changed | ONE `PACKAGE` rule + ONE `METHOD_CALL` rule per genuine method rename |
| Section says "package X moved to Y" + table includes rows where a class is replaced by a differently-named class with a different API | ONE `PACKAGE` rule + ONE `IMPORT` rule per class replacement |
| Section says "package X moved to Y" + table includes rows where the **class name changed** (even if the API is similar) | ONE `PACKAGE` rule + ONE `IMPORT` rule per class rename |
| Section says "ClassA moved to X, ClassB moved to Y, ClassC removed" (different targets) | Separate per-class rules |
| Section lists method-level API changes with no common package rename | Per-row `METHOD_CALL` or `IMPORT` rules as normal |

See `references/examples/<language>.md` for worked examples including reference tables and METHOD_CALL alongside PACKAGE rules.

For each pattern, provide the fields defined in `references/patterns-json-schema.md`. At minimum: `source_pattern`, `rationale`, `complexity`, `category`.

**Always populate `documentation_url`** with the URL to the migration guide or the relevant documentation section. If the guide was fetched from a URL, use that URL (with an anchor if available). The construct CLI converts this into a `links:` entry in the rule YAML so users can find the original migration guidance.

### Detection strategy: detect the affected artifact, not the missing fix

When a migration requires users to ADD something (a new annotation, a new dependency, a new config), you cannot detect its absence. Instead, detect the **artifact that is affected** and warn about the required change.

For example: if a test annotation no longer auto-configures a helper class, don't try to detect the missing annotation. Instead, detect the helper class usage (IMPORT) and warn that the annotation is now required.

### Source FQN must be the pre-migration (source) path

The `source_fqn` field is what the rule will match against in user code. It must be the **old/source** path — the one that exists in code that has NOT been migrated yet.

**Common mistake:** using the target/new package path as the pattern. Example:
- WRONG: `com.example.newpackage.MyClass` (this is the post-migration path — unmigrated code doesn't have this)
- RIGHT: `com.example.oldpackage.MyClass` (this is the pre-migration path — what actually appears in user code before migration)

**Verification:** For each `source_fqn`, ask: "Would this FQN appear in a project that has NOT been migrated yet?" If no, you have the wrong path.

The migration guide often shows both old and new paths. The old path goes in `source_fqn`; the new path goes in `target_pattern` and the migration message.

### Code example comparison (checklist item 9)

Migration guides often communicate API changes implicitly through before/after code examples rather than stating them in prose. These are **first-class extraction sources** — not illustrations.

When a section (or pair of related sections like "Preparation" and "Migration steps") shows source-version and target-version code doing the same thing, systematically diff them:

1. **Function/method renames:** a function or method called in old code has a different name in new code
2. **Parameter/argument changes:** same function name but different parameter types, order, or wrapping (e.g., raw int → duration object, string → enum)
3. **Construction changes:** how an object is created differs — different constructor, factory function, or initialization pattern
4. **Module/import changes:** different import paths, package names, or module references
5. **Constant/enum changes:** a named constant or enum value moves to a different module or is renamed
6. **API relocation:** a function/method moves from one module or object to another
7. **Removed calls:** API call present in old code with no equivalent in new code

Each difference is a separate pattern. If the guide shows source-version code in one section and equivalent target-version code in another, compare across sections — migration guides are organized by migration stage, not by API change.

**Common miss:** treating code examples as "just showing best practices" when they actually demonstrate API changes. If the source-version code calls `client.set_timeout(60)` and the target-version code calls `config.set_timeout(Duration.minutes(1))` — that IS a migration pattern even though no prose says "set_timeout moved to config."

**Code diffs produce rules independently of PACKAGE rules.** When you diff source-version and target-version code examples, extract a separate rule for EVERY class rename or API replacement you find — even if a PACKAGE rule already covers the old import. The PACKAGE rule fires on the import line; the class-specific rule fires on the usage site and tells the user the exact replacement. For example, if source code uses `new HttpPost(url)` and target code uses `ClassicRequestBuilder.post(url).build()`, extract an IMPORT rule for `HttpPost` even though the PACKAGE rule on `org.apache.http` also fires. The user needs to know that `HttpPost` is replaced by `ClassicRequestBuilder`, not just that the import path changed.

**Multi-path migration guides.** When a guide describes progressive migration paths (e.g., 4.x → classic 5.x → async 5.x), extract source-side patterns from ALL paths. If the async section shows `PoolingHttpClientConnectionManager` being replaced by `PoolingAsyncClientConnectionManager`, and `PoolingHttpClientConnectionManager` exists in the source version under a different FQN, extract a rule for that source-version FQN. Users migrating directly from 4.x to 5.x async need these rules.

### Read section lead paragraphs carefully

The most impactful change in a section is often stated in the **first paragraph** before the details. Don't skip straight to bullet lists and code examples — the opening text may describe a foundational change (e.g., an entire package rename) that the rest of the section merely elaborates on.

### Package registry pre-check for dependency patterns

Before emitting any pattern that uses `dependency_name` (which produces a `*.dependency` condition), verify the artifact's versioning scheme against the language's package registry. See `references/languages/<language>/instructions.md` for the registry URL, query format, and parsing logic.

If the registry confirms no plain-semver version exists, flag the pattern as a `suspected_kantra_limitation` and still emit the `*.dependency` condition — it is the correct condition type. Kantra's version comparator may not handle non-semver strings, but that is an engine limitation, not a rule design problem.

Collect flagged patterns in a `suspected_kantra_limitations` list and return it alongside `patterns_count` and `output_file`.

### Choosing the right condition type

See `references/languages/<language>/condition-types.md` for the language-specific condition-type reference and `references/patterns-json-schema.md` for which fields map to which condition type.

**One critical rule for config properties:** Always use `application.*\\.(properties|yml)` as the `file_pattern` — this covers both `.properties` and `.yml` formats. Never use `.*\\.properties` alone (too broad) or `application.*\\.properties` alone (misses YAML configs).

### What counts as an extractable migration item

See the full list in `references/patterns-json-schema.md` under "What Counts as an Extractable Pattern." Be thorough — if a section names a concrete artifact, it is extractable.

### 4. Coverage report

After processing all sections, print a coverage report:

```
Coverage Report:
  Sections processed: N
  Sections with patterns: M
  Sections skipped: K
  Total patterns extracted: P

  Skipped sections (header/links only):
  - "## Before You Start" — header only
  - "## Upgrading Web Features" — header only
  ...
```

Every skipped section must be "header only" or "links only." If a skip reason contains any other phrasing, re-run the checklist for that section.

### 5. Deduplicate

Same `source_fqn` or `dependency_name` should only appear once. If the guide mentions the same API multiple times with different context, merge into a single pattern with the most complete information.

### 6. Generate messages

For each pattern, generate a clear, actionable migration message (2-4 sentences) explaining:
1. What needs to change and why
2. What to replace it with (if applicable)

If before/after examples are available, include them formatted as markdown code blocks with language syntax highlighting.

The message should be just the text — no headers, no labels wrapping it.

### 7. Write patterns.json

Assemble the complete patterns.json with all extracted patterns and write it to the workspace at `output/<primary_source>-to-<primary_target>/patterns.json`:

```json
{
  "sources": ["<primary_source>", "<additional_source>", ...],
  "targets": ["<primary_target>", "<additional_target>", ...],
  "language": "<language>",
  "patterns": [...]
}
```

### 8. Construct rules

Run the CLI to convert patterns to validated rule YAML:

```bash
go run ./cmd/construct --patterns output/<primary_source>-to-<primary_target>/patterns.json --output <rules-dir>
```

This produces rule YAML files grouped by concern + ruleset.yaml.

### 9. Validate rules

Run validation:

```bash
go run ./cmd/validate --rules <rules-dir>
```

If validation fails, fix `output/<source>-to-<target>/patterns.json` and re-run construct. Common issues:
- Missing `source_fqn` → the rule condition has no pattern to match
- Invalid `location_type` → not one of the valid locations for the detected language (see condition-types.md)
- Invalid regex in `file_pattern` → `file_pattern` must be valid Go regex, NOT glob syntax. Use `.*\\.properties` not `*.properties`
- Duplicate `source_fqn` → same FQN appears in multiple patterns. Merge them into one

### 10. Return

Return the path to the rules directory and the coverage report to the orchestrator.
