# Java Condition Types

## java.referenced

Matches Java type, class, or annotation references by fully qualified name.

**Fields:**
- `pattern` (required) — Fully qualified Java class/type name (e.g., `javax.ejb.Stateless`, `org.springframework.boot.autoconfigure.SpringBootApplication`). Use exact FQNs, not wildcards.
- `location` (required for accurate matching) — Where the reference appears in code:

| Location | What It Matches | Example Code That Triggers It |
|---|---|---|
| `TYPE` | Type usage (variable types, generics, casts) | `Stateless s;` or `(Stateless) obj` — the type must appear as a declared/used type |
| `IMPORT` | Import statement | `import javax.servlet.http.HttpServlet;` |
| `ANNOTATION` | Annotation usage (not just import) | `@Stateless` on a class/method/field — must USE the annotation, `import` alone is not enough |
| `METHOD_CALL` | Method invocation | `initialContext.lookup("java:comp/env")` — pattern must be FQN including class: `javax.naming.InitialContext.lookup`. In source-only mode, the call must be on an explicitly typed variable (no chained calls). |
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

## java.dependency

Matches Maven dependencies by groupId.artifactId and version range.

**Fields:**
- `name` — Maven coordinate as `groupId.artifactId` (e.g., `org.springframework.boot.spring-boot-starter`). One of `name` or `nameRegex` is required.
- `nameRegex` — Regex alternative to `name` for matching multiple artifacts.
- `upperbound` — Version upper bound (exclusive). E.g., `3.0.0` matches deps below 3.0.0. At least one of `upperbound` or `lowerbound` is required.
- `lowerbound` — Version lower bound (inclusive). At least one of `upperbound` or `lowerbound` is required.

**Version bound derivation — use semantics, not artifact knowledge:**

Choose bounds based on *why* the dependency is flagged, not by looking up the artifact's version history:

1. **Artifact removed or renamed** (a different groupId/artifactId replaces it, or it is dropped entirely):
   Set `lowerbound: 0.0.0` with **no upperbound**. The artifact's presence is the signal regardless of version. There is no "safe" version of a removed artifact.

2. **Same artifact, behavior changes between source and target versions** (the artifact continues to exist but behaves differently):
   Set `upperbound` to the framework's target version (e.g., `4.0.0` for a Spring Boot 4 migration). This correctly scopes to the pre-migration version range.

3. **`upperbound` = target framework version is only valid for the framework's own artifacts** (e.g., `org.springframework.*`, `org.springframework.boot.*`).
   Third-party libraries (Flyway, Liquibase, Hibernate, Elasticsearch, Spock, AspectJ, etc.) use independent versioning — Flyway is at 10.x, Liquibase at 4.x. Applying the framework version (e.g., `4.0.0`) as their upperbound produces a bound that silently misses real projects. Use `lowerbound: 0.0.0` with no upperbound for third-party artifacts.

**Note:** `java.dependency` requires full analysis (not `source-only` mode) since it uses Maven dependency resolution, not JDTLS.
