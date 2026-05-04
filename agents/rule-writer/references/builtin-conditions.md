# Builtin Conditions

These work for any language — they match file contents, file existence, or file structure rather than resolved symbols.

## builtin.filecontent

Matches regex patterns in file contents. Use this for config files, properties, XML config, or when no language-specific provider can detect the pattern.

**Fields:**
- `pattern` (required) — Regex pattern to match in file contents (e.g., `javax\.servlet`, `spring\.jpa\.hibernate\.ddl-auto`). Must be a valid Go regex.
- `filePattern` (optional) — Regex restricting which files to search. Must be a valid Go regex — do NOT use glob syntax (`*.properties` is invalid regex; use `.*\\.properties`). For application config properties, always use `application.*\\.(properties|yml)` to cover both formats. Never use `.*\\.properties` alone (too broad — matches any `.properties` file) or `application.*\\.properties` alone (misses YAML configs). Omit to search all files.
- `filepaths` (optional) — Restrict to specific file paths.

## builtin.file

Matches file existence by name pattern.

**Fields:**
- `pattern` (required) — File name glob (e.g., `persistence.xml`, `web.xml`, `struts-config.xml`).

## builtin.xml

Matches XPath expressions in XML files. Use for structured XML content like POM sections, Spring XML config, web.xml entries.

**Fields:**
- `xpath` (required) — XPath expression (e.g., `//*[local-name()='persistence-unit']`).
- `namespaces` (optional) — Map of prefix→URI for namespace-aware XPath.
- `filepaths` (optional) — Restrict to specific XML files.

## builtin.json

Matches XPath-like expressions in JSON files.

**Fields:**
- `xpath` (required) — Path expression.
- `filepaths` (optional) — Restrict to specific JSON files.

## builtin.hasTags

Checks for tags on matched code elements. Used in combination with chaining (`from`/`as`).

**Fields:** A string array of tag names to check for.

## builtin.xmlPublicID

Matches DOCTYPE public ID declarations in XML files.

**Fields:**
- `regex` (required) — Regex matching the public ID string. Must be valid Go regex.
- `namespaces` (optional) — Namespace mappings.
- `filepaths` (optional) — Restrict to specific files.

## Combinators

### or

Matches if ANY child condition matches. Use when a migration pattern has alternative APIs, alternative FQNs, or multiple entry points that all need the same migration.

```yaml
when:
  or:
    - java.referenced:
        pattern: javax.ejb.Stateless
        location: ANNOTATION
    - java.referenced:
        pattern: javax.ejb.Stateful
        location: ANNOTATION
```

**Note:** You don't need to create `or` conditions manually in `patterns.json`. If you set `alternative_fqns` in a pattern, `go run ./cmd/construct` wraps them in an `or` automatically.

### and

Matches if ALL child conditions match. Use for multi-signal detection (e.g., both an import AND a config entry must be present).

```yaml
when:
  and:
    - java.referenced:
        pattern: javax.servlet.http.HttpServlet
        location: INHERITANCE
    - builtin.filecontent:
        pattern: doGet|doPost
        filePattern: "*.java"
```

## Chaining Fields

Any condition can include these fields for advanced matching:
- `from` — Chain from a previous condition's result set (use with `as` on the prior condition)
- `as` — Name this condition's result set (referenced by `from` on a later condition)
- `ignore` — If `true`, the match is recorded but doesn't produce a violation
- `not` — If `true`, matches when the condition does NOT match (negation)

## Choosing the Right Condition Type

| Scenario | Condition Type | patterns.json Fields |
|---|---|---|
| Java API/annotation migration | `java.referenced` | `source_fqn` + `location_type` + `provider_type: java` |
| Java dependency version check | `java.dependency` | `dependency_name` + `upper_bound` (and/or `lower_bound`) |
| Go package/symbol migration | `go.referenced` | `source_fqn` + `provider_type: go` |
| Go module version check | `go.dependency` | `dependency_name` + `upper_bound` + `provider_type: go` |
| Node.js/React/Angular migration | `nodejs.referenced` | `source_fqn` + `provider_type: nodejs` |
| C# / .NET migration | `csharp.referenced` | `source_fqn` + `provider_type: csharp` |
| Python migration | `python.referenced` | `source_fqn` + `provider_type: python` |
| Config files (properties, YAML) | `builtin.filecontent` | `source_fqn` (regex) + `file_pattern` + `provider_type: builtin` |
| XML structure (POM, Spring config) | `builtin.xml` | `xpath` + `namespaces` + `xpath_filepaths` |
| JSON structure (package.json) | `builtin.json` | `xpath` + `filepaths` |
| File existence (web.xml, etc.) | `builtin.file` | Not yet in patterns.json — use raw rule YAML |
| Multiple alternatives → same migration | `or` combinator | Set `alternative_fqns` in patterns.json |
| Co-occurring patterns required | `and` combinator | Not yet in patterns.json — use raw rule YAML |
