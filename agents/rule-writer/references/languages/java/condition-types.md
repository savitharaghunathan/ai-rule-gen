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

### METHOD_CALL known limitations

Kantra resolves METHOD_CALL patterns by matching the **declared type of the receiver variable**, not the runtime type or the interface/superclass where the method is defined. This causes two failure modes in real-world code:

**1. Type hierarchy mismatch**

The method is defined on an interface, but code uses a concrete implementation:

```java
// Rule pattern: javax.jms.MessageProducer.send
// Variable declared as concrete type — kantra sees ActiveMQMessageProducer, NOT the interface
ActiveMQMessageProducer producer = session.createProducer(dest);
producer.send(message);  // DOES NOT MATCH the interface FQN pattern
```

**Workarounds:** Use a short method name pattern (`send`), or use `alternative_fqns` to create an `or` condition covering both the interface and known concrete types.

**2. Builder chain resolution**

Methods called on builder chains where kantra can't resolve the return type:

```java
// Rule pattern: org.springframework.security.config.annotation.web.builders.HttpSecurity.authorizeRequests
// kantra can't resolve the return type of http.csrf().disable() back to HttpSecurity
http.csrf().disable().authorizeRequests();  // DOES NOT MATCH
```

**Workaround:** Use a short method name pattern (`authorizeRequests`).

**3. Inner class / Builder class FQN resolution**

FQN patterns that include an inner class name (e.g., `Config.Builder.method`) fail for the same reason as builder chains — kantra can't resolve factory method return types:

```java
// Rule pattern: com.example.ClientConfig.Builder.setReadTimeout
// kantra can't resolve ClientConfig.builder() → ClientConfig.Builder
ClientConfig.builder()
    .setReadTimeout(5000);  // DOES NOT MATCH
```

This is a **hard failure**, not a corner case. The factory method returns an inner `Builder` type, but kantra sees only the factory call and can't map it. The pattern compiles without error but silently matches nothing.

**Workaround:** Use a short method name pattern (`setReadTimeout`). If the method name also exists in the target API, accept the false positive risk — a rule that fires on already-migrated code is better than one that never fires. For hand-written rules (not generated via patterns.json), use `and`/`as`/`from` scoping to restrict to files importing the source package.

### When NOT to use builtin.filecontent for method detection

Do not use `builtin.filecontent` to detect method calls, especially in builder chains.

**Problem:** A regex like `RequestConfig.*\.setConnectTimeout` assumes the class name and method call are on the same line. In real code, builder chains break across lines:

```java
RequestConfig.custom()
    .setConnectTimeout(60000)  // on a different line — regex misses this
```

Go regex `.` does not match newlines, so `.*` stops at the line break and the pattern silently fails.

**Fix:** Use `java.referenced` with `location: METHOD_CALL` and a short method name pattern. If the method name is shared with the target API, scope with `and`/`as`/`from` to restrict matching to files importing the source package:

```yaml
when:
  and:
    - java.referenced:
        pattern: com.old.library*
        location: PACKAGE
      as: oldLibFile
    - java.referenced:
        pattern: setReadTimeout
        location: METHOD_CALL
      from: oldLibFile
```

**Rule of thumb:** If you're detecting a method call, use `METHOD_CALL`. Reserve `builtin.filecontent` for configuration files (`.properties`, `.yml`) and build files where no semantic analyzer exists.

### METHOD_CALL pattern style decision framework

1. Is the method name unique to the migration source library? (i.e., unlikely to appear in unrelated code)
   - **Yes →** Use short method name. Simple, resilient to type hierarchy and builder chains.
   - **No →** Continue to step 2.

2. Can users call this method on concrete subtypes or via builder chains?
   - **Yes →** Use short method name with `and`/`as`/`from` scoping to limit false positives (see Condition combinators below).
   - **No (always on the exact declared type) →** Use FQN pattern.

3. Do you need to disambiguate overloaded methods?
   - **Yes →** Use method signature pattern (e.g., `getForObject(URI,Class<Source>)`).

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
