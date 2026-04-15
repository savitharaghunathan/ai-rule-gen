# Condition Types Reference

Konveyor analyzer rules use `when` conditions to match source code patterns. Each condition type maps to a language provider or a builtin capability.

## Language-Specific Conditions

### java.referenced

Matches Java type, class, or annotation references by fully qualified name.

**Fields:**
- `pattern` (required) — Fully qualified Java class/type name (e.g., `javax.ejb.Stateless`, `org.springframework.boot.autoconfigure.SpringBootApplication`). Use exact FQNs, not wildcards.
- `location` (required for accurate matching) — Where the reference appears in code:

| Location | What It Matches | Example Code That Triggers It |
|---|---|---|
| `TYPE` | Type usage (variable types, generics, casts) | `Stateless s;` or `(Stateless) obj` — the type must appear as a declared/used type |
| `IMPORT` | Import statement | `import javax.servlet.http.HttpServlet;` |
| `ANNOTATION` | Annotation usage | `@Stateless` on a class/method/field |
| `METHOD_CALL` | Method invocation | `initialContext.lookup("java:comp/env")` — pattern must be FQN including class: `javax.naming.InitialContext.lookup` |
| `CONSTRUCTOR_CALL` | `new` expression | `new InitialContext()` |
| `INHERITANCE` | `extends` clause | `class MyServlet extends HttpServlet` |
| `IMPLEMENTS_TYPE` | `implements` clause | `class MyBean implements SessionBean` |
| `FIELD` | Field declaration type | `@Inject private DataSource ds;` |
| `METHOD` | Method declaration (return type or annotation) | Method with matching annotation or return type |
| `CLASS` | Class declaration (annotation on class) | `@Stateless public class MyBean` |
| `RETURN_TYPE` | Method return type | `public Response handle()` |
| `VARIABLE_DECLARATION` | Local variable type | `DataSource ds = ...;` |
| `ENUM` | Enum constant reference | `CascadeType.ALL` |
| `PACKAGE` | Package declaration | `package javax.ejb;` |

**Critical matching rules:**
- For `METHOD_CALL`: The pattern must include the class FQN + method name (e.g., `javax.naming.InitialContext.lookup`). The analyzer resolves fully qualified names — **static imports are NOT matched by METHOD_CALL**, so never rely on static import + unqualified call.
- For `TYPE`/`ANNOTATION`/`IMPORT`: The pattern is the FQN of the type itself. The analyzer matches the resolved FQN regardless of how the code references it (short name, fully qualified, wildcard import).
- For `INHERITANCE`/`IMPLEMENTS_TYPE`: The pattern is the FQN of the superclass/interface.

**Optional fields:**
- `annotated` — Filter: only match if the matched element also has a specific annotation. Sub-fields: `pattern` (FQN of the annotation), `elements` (list of `{name, value}` pairs for annotation element values).
- `filepaths` — Restrict matching to specific file paths.

### java.dependency

Matches Maven dependencies by groupId.artifactId and version range.

**Fields:**
- `name` — Maven coordinate as `groupId.artifactId` (e.g., `org.springframework.boot.spring-boot-starter`). One of `name` or `nameRegex` is required.
- `nameRegex` — Regex alternative to `name` for matching multiple artifacts.
- `upperbound` — Version upper bound (exclusive). E.g., `3.0.0` matches deps below 3.0.0. At least one of `upperbound` or `lowerbound` is required.
- `lowerbound` — Version lower bound (inclusive). At least one of `upperbound` or `lowerbound` is required.

**Note:** `java.dependency` requires full analysis (not `source-only` mode) since it uses Maven dependency resolution, not JDTLS.

### go.referenced

Matches Go symbol references — package imports, function calls, type usage.

**Fields:**
- `pattern` (required) — Full import path, optionally with symbol name (e.g., `golang.org/x/crypto/md4`, `net.IP`, `crypto/md5.New`).

**Important:** kantra v0.9.0-alpha.6 container does NOT include a Go toolchain — only gopls. `go.referenced` rules fail with a "no views" error in the container. Use `kantra test --run-local` or `kantra analyze --run-local` for Go rules.

### go.dependency

Matches Go module dependencies from `go.mod`.

**Fields:**
- `name` — Module path (e.g., `golang.org/x/crypto`). One of `name` or `nameRegex` required.
- `nameRegex` — Regex alternative.
- `upperbound`, `lowerbound` — Version bounds.

### nodejs.referenced

Matches Node.js/TypeScript symbol references — imports, component usage.

**Fields:**
- `pattern` (required) — Package + exported symbol (e.g., `express.Router`, `@patternfly/react-core.Button`).

### csharp.referenced

Matches C# symbol references.

**Fields:**
- `pattern` (required) — Fully qualified name (e.g., `System.Web.HttpContext`, `Microsoft.EntityFrameworkCore.DbContext`).
- `location` (optional) — One of: `ALL` (default), `METHOD`, `FIELD`, `CLASS`.

## Builtin Conditions

These work for any language — they match file contents, file existence, or file structure rather than resolved symbols.

### builtin.filecontent

Matches regex patterns in file contents. Use this for config files, properties, XML config, or when no language-specific provider can detect the pattern.

**Fields:**
- `pattern` (required) — Regex pattern to match in file contents (e.g., `javax\.servlet`, `spring\.jpa\.hibernate\.ddl-auto`). Must be a valid Go regex.
- `filePattern` (optional) — Glob restricting which files to search (e.g., `*.properties`, `*.xml`, `application.*\\.yml`). Omit to search all files.
- `filepaths` (optional) — Restrict to specific file paths.

### builtin.file

Matches file existence by name pattern.

**Fields:**
- `pattern` (required) — File name glob (e.g., `persistence.xml`, `web.xml`, `struts-config.xml`).

### builtin.xml

Matches XPath expressions in XML files. Use for structured XML content like POM sections, Spring XML config, web.xml entries.

**Fields:**
- `xpath` (required) — XPath expression (e.g., `//*[local-name()='persistence-unit']`).
- `namespaces` (optional) — Map of prefix→URI for namespace-aware XPath.
- `filepaths` (optional) — Restrict to specific XML files.

### builtin.json

Matches XPath-like expressions in JSON files.

**Fields:**
- `xpath` (required) — Path expression.
- `filepaths` (optional) — Restrict to specific JSON files.

### builtin.hasTags

Checks for tags on matched code elements. Used in combination with chaining (`from`/`as`).

**Fields:** A string array of tag names to check for.

### builtin.xmlPublicID

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
| Config files (properties, YAML) | `builtin.filecontent` | `source_fqn` (regex) + `file_pattern` + `provider_type: builtin` |
| XML structure (POM, Spring config) | `builtin.xml` | `xpath` + `namespaces` + `xpath_filepaths` |
| File existence (web.xml, etc.) | `builtin.file` | Not yet in patterns.json — use raw rule YAML |
| Multiple alternatives → same migration | `or` combinator | Set `alternative_fqns` in patterns.json |
| Co-occurring patterns required | `and` combinator | Not yet in patterns.json — use raw rule YAML |
