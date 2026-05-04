# Java Test Data Guide

## Project Structure

```
<data-dir>/
├── pom.xml                                        # Maven build file
└── src/main/java/com/example/Application.java     # Source code
```

## How the Analyzer Matches Java Conditions

| Condition Type | Location | What the Test Code Must Do |
|---|---|---|
| `java.referenced` | `ANNOTATION` | USE the annotation on a class, method, or field (e.g., `@MockBean private Object svc;`). **An `import` statement alone is NOT enough** — the annotation must appear as `@AnnotationName` on an actual element. |
| `java.referenced` | `IMPORT` | Include the import statement (e.g., `import javax.servlet.http.HttpServlet;`) |
| `java.referenced` | `TYPE` | Declare or use the type (e.g., `HttpServlet servlet;` or a cast) |
| `java.referenced` | `METHOD_CALL` | Call the method on an explicitly typed variable — do NOT chain calls (e.g., use `Foo f = Foo.get(); f.bar();` not `Foo.get().bar()`). JDTLS in source-only mode cannot resolve return types of chained calls without the dependency JAR. |
| `java.referenced` | `CONSTRUCTOR_CALL` | Use `new ClassName()` |
| `java.referenced` | `INHERITANCE` | `class Foo extends TargetClass` |
| `java.referenced` | `IMPLEMENTS_TYPE` | `class Foo implements TargetInterface` |
| `java.referenced` | `FIELD` | Declare a field of that type |
| `java.referenced` | `VARIABLE_DECLARATION` | Declare a local variable of that type |
| `java.referenced` | `RETURN_TYPE` | Declare a method with that return type |
| `java.dependency` | — | The `pom.xml` must declare the dependency with a version that satisfies the rule's bounds. **No source code needed** — only the pom.xml matters. |
| `builtin.xml` | — | The XML file must contain elements matching the XPath expression. If `filepaths` is set, the file must be at that path. |

## Dependency Resolution

Do NOT run `mvn compile`, `mvn dependency:resolve`, or any Maven command. Do NOT import the project into an IDE. Kantra resolves `java.dependency` rules by parsing the pom.xml directly, and source-only analysis resolves IMPORT/ANNOTATION/TYPE patterns without downloaded JARs. Running Maven or IDE imports creates `.classpath`, `.project`, `.settings/`, `target/`, and `.factorypath` artifacts that pollute the test data.

### Prefer minimal dependencies

JDTLS runs inside the kantra container with limited memory. Prefer the lightest dependency that provides the class you need — e.g., `spring-boot-autoconfigure` instead of a full starter like `spring-boot-starter-web`.

### java.dependency and builtin.xml tests

These condition types do not use JDTLS. Test YAML files for `java.dependency` and `builtin.xml` rules should omit `analysisParams: mode: source-only`.

## java.dependency Version Bounds

This is the most common cause of test failures.

A `java.dependency` rule matches when:
- The pom.xml declares the artifact AND
- The declared version is **strictly less than** the `upperbound` (if set) AND
- The declared version is **greater than or equal to** the `lowerbound` (if set)

**If the declared version is >= upperbound, the rule does NOT match and the test FAILS.**

**Rules for choosing test versions:**
1. The version MUST be strictly less than the `upperbound`
2. Use a realistic version that actually exists on Maven Central
3. **Use plain numeric versions** (e.g., `2.3.0`) — kantra cannot compare qualified versions like `2.3-groovy-4.0` against numeric bounds
4. When in doubt, use a version in the `3.x` range
5. Use the `name` field as `groupId.artifactId` (dot-separated)

**Handling artifacts that only publish qualified versions:**

Some artifacts never publish plain numeric versions (e.g., Spock: `2.3-groovy-3.0`, Hibernate: `6.4.0.Final`). For these, use the Spring Boot parent BOM to manage the version — declare the dependency without a `<version>` tag.

**Handling discontinued artifacts:**

Use the last published groupId and version. If no version was ever published under the rule's groupId, flag it.

### Use realistic dependency versions

Do NOT fabricate version numbers. Common mistakes:
- `elasticsearch-rest-client:3.9.0` — never had a 3.x release
- `hibernate-proxool:6.4.0.Final` — dropped before Hibernate 6
- `spock-spring:2.3.0` — plain `2.3.0` does not exist

When unsure, use a Spring Boot parent BOM (`3.2.0` or `3.3.0`) with managed dependencies.

**Version verification checklist:**
1. Is this version plain numeric? If not, prefer BOM-managed
2. Does this groupId + artifactId + version actually exist?
3. Is the version strictly below the upperbound?
4. Was this artifact ever published under this groupId?
