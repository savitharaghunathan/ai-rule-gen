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
| `METHOD_CALL` | Method invocation | `initialContext.lookup("java:comp/env")` — supports three pattern styles (see METHOD_CALL pattern styles below). In source-only mode, the call must be on an explicitly typed variable. |
| `CONSTRUCTOR_CALL` | `new` expression | `new InitialContext()` |
| `INHERITANCE` | `extends` clause | `class MyServlet extends HttpServlet` |
| `IMPLEMENTS_TYPE` | `implements` clause | `class MyBean implements SessionBean` |
| `FIELD` | Field declared with this type — pattern is the FQN of the field's **type**, not the field name. `FIELD_DECLARATION` is an alias. | `private DataSource ds;` matches pattern `javax.sql.DataSource`. Does **NOT** match static field access like `Config.MY_CONSTANT`. |
| `METHOD` | Method definition site — matches where the method is **defined**, not where it's called. Use `METHOD_CALL` for invocations. | `public void doWork()` matches pattern `com.example.MyService.doWork` |
| `CLASS` | Class declaration — typically used with `annotated:` to find classes carrying a specific annotation | `@Controller public class MyController` with `annotated: {pattern: org.springframework.stereotype.Controller}` |
| `RETURN_TYPE` | Method return type | `public EntityManager getEM()` matches pattern `javax.persistence.EntityManager` |
| `VARIABLE_DECLARATION` | Local variable type | `DataSource ds = ...;` matches pattern `javax.sql.DataSource` |
| `ENUM` | Enum type or constant reference — uses generic type matching | `SessionCreationPolicy.IF_REQUIRED` or full FQN `org.springframework.security.config.http.SessionCreationPolicy` |
| `PACKAGE` | Package usage — matches any reference to types in the package (imports, fully qualified names). Use asterisk suffix for subpackage matching: `org.apache.http` matches types directly in that package, `com.fasterxml.jackson*` matches types in all subpackages too. Always append `*` when the target classes live in subpackages of the pattern. | `import org.apache.http.HttpResponse;` matches pattern `org.apache.http`; `import com.fasterxml.jackson.databind.ObjectMapper;` matches pattern `com.fasterxml.jackson*` (asterisk required because `ObjectMapper` is in subpackage `databind`) |

**Critical matching rules:**
- For `METHOD_CALL`: See "METHOD_CALL pattern styles" and "METHOD_CALL known limitations" below. **Static imports are NOT matched by METHOD_CALL**, so never rely on static import + unqualified call.
- For `TYPE`/`ANNOTATION`/`IMPORT`: The pattern is the FQN of the type itself. The analyzer matches the resolved FQN regardless of how the code references it (short name, fully qualified, wildcard import).
- For `INHERITANCE`/`IMPLEMENTS_TYPE`: The pattern is the FQN of the superclass/interface.
- For `FIELD` (or `FIELD_DECLARATION`): The pattern is the FQN of the field's **declared type**. It matches when a class field is declared with that type (e.g., `private DataSource ds` is matched by pattern `javax.sql.DataSource`). It does **NOT** detect static field/constant access (e.g., `HttpCoreContext.HTTP_TARGET_HOST` or `Config.MY_CONSTANT`). For static constant access, use `builtin.filecontent` with a regex pattern.
- For `METHOD` vs `METHOD_CALL`: `METHOD` matches the **definition** site (where the method is declared). `METHOD_CALL` matches **invocation** sites (where the method is called). For migration rules, you almost always want `METHOD_CALL` — use `METHOD` only when you need to find where a method is defined (rare).

### METHOD_CALL pattern styles

Three pattern styles are supported. Choose based on how the method is called in real-world code:

| Style | Pattern | Matches | Use when |
|---|---|---|---|
| **FQN** | `org.hibernate.Session.createQuery` | Only calls on variables declared as `Session` | The variable is always declared as the exact type containing the method |
| **Short method name** | `addInterceptor` | Calls on ANY class with that method name | The method may be called on concrete subtypes, via builder chains, or when the FQN would miss matches (see limitations below) |
| **Method signature** | `getForObject(URI,Class<Source>)` | Calls matching the method name AND parameter types | Disambiguating overloaded methods or matching specific signatures |

**Examples:**

```yaml
# FQN — exact type match
java.referenced:
    pattern: org.hibernate.Session.createQuery
    location: METHOD_CALL

# Short method name — avoids type hierarchy issues
java.referenced:
    pattern: getRedirectUriTemplate
    location: METHOD_CALL

# Method signature — disambiguates overloads
java.referenced:
    pattern: 'getForObject(URI,Class<Source>)'
    location: METHOD_CALL
```

### METHOD_CALL pattern style

**Always use FQN patterns** (e.g., `org.apache.http.HttpResponse.getStatusLine`). The Java provider in analyzer-lsp supports fully qualified method patterns — the `source_fqn` should always include the class that declares or inherits the method.

When the method is defined on an interface but code may use concrete subtypes, add `alternative_fqns` covering the interface + known concrete types.

When you need to disambiguate overloaded methods, use a method signature pattern (e.g., `getForObject(URI,Class<Source>)`).

**Never use short method names** (e.g., bare `getStatusLine`) — these match on any class with that method name, producing false positives.

**Never use `builtin.filecontent`** to detect method calls. Regexes break on multi-line builder chains because Go's `.` does not match newlines. Use `java.referenced` with `location: METHOD_CALL`.

### METHOD_CALL pattern style decision framework

1. Is the method defined on an interface but called on concrete subtypes?
   - **Yes →** Use FQN with `alternative_fqns` covering the interface + known concrete types.
   - **No →** Continue to step 2.

2. Do you need to disambiguate overloaded methods, or does the method have argument types that changed in the migration?
   - **Yes →** Use method signature pattern (e.g., `getForObject(URI,Class<Source>)`). Include argument types when the method name is common or overloaded — this prevents false positives on unrelated classes with the same method name.
   - **No →** Use FQN pattern (`org.example.ClassName.methodName`).

## Condition combinators

Rules can use `or`, `and`, `as`/`from`, and `not` to combine multiple conditions. These are standard in production rulesets.

### `or` — match ANY of several patterns

Use for type hierarchy (interface + concrete types) or multiple entry points:

```yaml
when:
  or:
    - java.referenced:
        pattern: javax.jms.ConnectionFactory.createConnection
        location: METHOD_CALL
    - java.referenced:
        pattern: javax.jms.QueueConnectionFactory.createQueueConnection
        location: METHOD_CALL
```

Can mix providers:

```yaml
when:
  or:
    - java.referenced:
        pattern: 'java.sql.DriverManager.getConnection(java.lang.String,java.lang.String,java.lang.String)'
        location: METHOD_CALL
    - builtin.filecontent:
        pattern: 'jdbc:postgresql:.*(user=.+|password=.+)'
        filePattern: '.*\.java'
```

### `and` with `as`/`from` — scope conditions to matching files

Use to limit a short method name to files that import from a specific package:

```yaml
when:
  and:
    - java.referenced:
        pattern: javax.servlet*
        location: PACKAGE
      as: servletFile
    - java.referenced:
        pattern: getRequestDispatcher
        location: METHOD_CALL
      from: servletFile
```

The `as` names the first condition's match set. `from` restricts the second condition to files where the first matched. Note: Java rules require uppercase `Filepaths` in interpolation (`"{{servletFile.Filepaths}}"`)

### `not` — exclude matches

```yaml
when:
  and:
    - java.referenced:
        pattern: org.springframework.web.bind.annotation.RequestMapping
        location: ANNOTATION
      as: annotation
    - java.referenced:
        pattern: org.springframework.stereotype.Controller
        location: ANNOTATION
        filepaths: "{{annotation.Filepaths}}"
      not: true
      from: annotation
```

### Nested combinators

`or` can contain `and` blocks and vice versa to arbitrary depth:

```yaml
when:
  or:
    - and:
        - java.referenced:
            pattern: org.springframework.stereotype.Controller
            location: ANNOTATION
          as: class
          ignore: true
        - java.referenced:
            pattern: getAllHeaders
            location: METHOD_CALL
            filepaths: "{{class.Filepaths}}"
          from: class
    - java.referenced:
        pattern: 'getForObject(URI,Class<Source>)'
        location: METHOD_CALL
```

Use `ignore: true` on intermediate conditions that serve only as scoping — they won't produce their own incidents.

**Optional fields:**
- `annotated` — Filter: only match if the matched element also has a specific annotation. Sub-fields: `pattern` (FQN of the annotation), `elements` (list of `{name, value}` pairs for annotation element values).
- `filepaths` — Restrict matching to specific file paths.

## JDK modules are NOT Maven dependencies

JDK modules (e.g., `jdk.random`, `jdk.jsobject`, `jdk.httpserver`) are part of the JDK platform, not Maven artifacts. They do NOT appear as `<dependency>` entries in `pom.xml`. Using `java.dependency` for a JDK module produces a rule that will never fire.

**For removed/deprecated JDK modules**, use `java.referenced` with `location: PACKAGE` or `IMPORT` matching the module's exported packages:
- Example: module `jdk.jsobject` exports package `netscape.javascript` → detect `netscape.javascript` with `location: IMPORT`
- Example: module `jdk.random` exports package `jdk.random` → detect `jdk.random` with `location: PACKAGE`

**How to tell the difference:**
- Maven dependency → has a `groupId:artifactId` coordinate, appears in `pom.xml`/`build.gradle` → use `java.dependency`
- JDK module → starts with `java.` or `jdk.`, is part of the JDK platform distribution → use `java.referenced` with the module's exported package

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
