# Java Extraction Examples

## Example 1: `java.referenced` with CONSTRUCTOR_CALL location

### Guide Excerpt

> ### JNDI Lookups
>
> JNDI lookups via `InitialContext` are not supported in Quarkus. Replace
> `new InitialContext()` and `Context.lookup()` calls with CDI `@Inject`.
>
> ```java
> // Before
> InitialContext ctx = new InitialContext();
> DataSource ds = (DataSource) ctx.lookup("java:jboss/datasources/MyDS");
>
> // After
> @Inject
> AgroalDataSource dataSource;
> ```

### Checklist

Section: "JNDI Lookups" -> EXTRACT: removed API usage (item 1); two detectable artifacts: InitialContext constructor and lookup() method call

### patterns.json

Two patterns -- one per detectable artifact:

```json
{
  "source_pattern": "JNDI InitialContext not supported in Quarkus",
  "target_pattern": "CDI @Inject",
  "source_fqn": "javax.naming.InitialContext",
  "location_type": "CONSTRUCTOR_CALL",
  "rationale": "JNDI lookups via InitialContext are not supported in Quarkus; use CDI @Inject",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "di",
  "provider_type": "java",
  "documentation_url": "https://quarkus.io/guides/cdi-reference"
}
```

```json
{
  "source_pattern": "JNDI lookup() method not supported in Quarkus",
  "target_pattern": "CDI @Inject",
  "source_fqn": "javax.naming.Context.lookup*",
  "location_type": "METHOD_CALL",
  "rationale": "JNDI lookup() calls are not supported in Quarkus; use CDI @Inject",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "di",
  "provider_type": "java",
  "documentation_url": "https://quarkus.io/guides/cdi-reference"
}
```

Note: `location_type` is `CONSTRUCTOR_CALL` for `new InitialContext()` and `METHOD_CALL` for `ctx.lookup()`. Choosing the right location type ensures kantra matches at the correct code site rather than just at the import.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: java-ee-to-quarkus-00010
  description: JNDI lookups via InitialContext are not supported in Quarkus; use CDI @Inject
  category: mandatory
  effort: 5
  labels:
    - konveyor.io/source=java-ee
    - konveyor.io/target=quarkus
  when:
    java.referenced:
      pattern: javax.naming.InitialContext
      location: CONSTRUCTOR_CALL
```

### Test Data (what triggers this rule)

```java
package com.example;

import javax.naming.InitialContext;
import javax.naming.NamingException;

public class JndiService {
    public Object lookup() throws NamingException {
        InitialContext ctx = new InitialContext();
        return ctx.lookup("java:app/MyService");
    }
}
```

---

## Example 2: `java.dependency` with version bounds

### Guide Excerpt

> ### Spring Web Dependency
>
> Replace the Spring Web artifact with the Quarkus `spring-web` extension.
> Remove both `org.springframework:spring-web` and
> `org.springframework.boot:spring-boot-starter-web` from your POM and add
> `io.quarkus:quarkus-spring-web` instead.

### Checklist

Section: "Spring Web Dependency" -> EXTRACT: dependency replacement (item 3); two artifacts named

### patterns.json

Each dependency artifact is a separate pattern. Do NOT use `alternative_fqns` for dependency patterns -- `alternative_fqns` only works with `*.referenced` conditions (it clones the pattern and replaces `source_fqn`, but leaves `dependency_name` unchanged, producing a duplicate).

```json
{
  "source_pattern": "Spring Web dependency replaced by Quarkus spring-web extension",
  "target_pattern": "io.quarkus:quarkus-spring-web",
  "dependency_name": "org.springframework.spring-web",
  "lower_bound": "0.0.0",
  "rationale": "Replace Spring Web with Quarkus spring-web extension",
  "complexity": "trivial",
  "category": "mandatory",
  "concern": "web",
  "provider_type": "java",
  "documentation_url": "https://quarkus.io/guides/spring-web"
}
```

```json
{
  "source_pattern": "Spring Boot starter-web dependency replaced by Quarkus spring-web extension",
  "target_pattern": "io.quarkus:quarkus-spring-web",
  "dependency_name": "org.springframework.boot.spring-boot-starter-web",
  "lower_bound": "0.0.0",
  "rationale": "Replace Spring Boot starter-web with Quarkus spring-web extension",
  "complexity": "trivial",
  "category": "mandatory",
  "concern": "web",
  "provider_type": "java",
  "documentation_url": "https://quarkus.io/guides/spring-web"
}
```

Note: `lower_bound: "0.0.0"` means "any version" -- the dependency's mere presence is the signal. The `dependency_name` format is `groupId.artifactId` (dot-separated, not colon-separated).

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: springboot-to-quarkus-00010
  description: Replace Spring Web with Quarkus spring-web extension
  category: mandatory
  effort: 1
  labels:
    - konveyor.io/source=springboot
    - konveyor.io/target=quarkus
  when:
    java.dependency:
      name: org.springframework.spring-web
      lowerbound: 0.0.0
```

### Test Data (what triggers this rule)

```xml
<project>
  <dependencies>
    <dependency>
      <groupId>org.springframework</groupId>
      <artifactId>spring-web</artifactId>
      <version>5.3.31</version>
    </dependency>
  </dependencies>
</project>
```

---

## Example 3: `builtin.xml` with XPath and namespaces

### Guide Excerpt

> ### Spring Boot Parent POM
>
> Replace the Spring Boot parent POM with the Quarkus BOM. If your
> `pom.xml` uses `spring-boot-starter-parent` or `spring-boot-parent`
> as `<parent>`, replace it with a Quarkus `<dependencyManagement>` section.

### Checklist

Section: "Spring Boot Parent POM" -> EXTRACT: build structure change affecting pom.xml (item 8)

### patterns.json

```json
{
  "source_pattern": "Spring Boot parent POM must be replaced with Quarkus BOM",
  "target_pattern": "Quarkus BOM in dependencyManagement",
  "source_fqn": "spring-boot-starter-parent",
  "xpath": "/m:project/m:parent[m:groupId/text() = 'org.springframework.boot' and m:artifactId/text() = 'spring-boot-starter-parent']",
  "namespaces": {
    "m": "http://maven.apache.org/POM/4.0.0"
  },
  "xpath_filepaths": ["pom.xml"],
  "rationale": "Replace Spring Boot parent POM with Quarkus BOM in dependencyManagement",
  "complexity": "trivial",
  "category": "mandatory",
  "concern": "build",
  "provider_type": "builtin",
  "documentation_url": "https://quarkus.io/guides/maven-tooling"
}
```

Note: When `xpath` is set, construct produces a `builtin.xml` condition regardless of `provider_type`. The `namespaces` map defines the XML namespace prefix (`m`) used in the XPath. The `xpath_filepaths` limits matching to `pom.xml` files only.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: springboot-to-quarkus-00030
  description: Replace Spring Boot parent POM with Quarkus BOM in dependencyManagement
  category: mandatory
  effort: 1
  labels:
    - konveyor.io/source=springboot
    - konveyor.io/target=quarkus
  when:
    builtin.xml:
      xpath: "/m:project/m:parent[m:groupId/text() = 'org.springframework.boot' and m:artifactId/text() = 'spring-boot-starter-parent']"
      namespaces:
        m: http://maven.apache.org/POM/4.0.0
      filepaths:
        - pom.xml
```

### Test Data (what triggers this rule)

```xml
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>2.3.2.RELEASE</version>
  </parent>
  <groupId>com.example</groupId>
  <artifactId>sample</artifactId>
</project>
```

---

## Example 4: `java.referenced` with `source_artifact` for deterministic verification

### Guide Excerpt

> ### BootstrapRegistry and EnvironmentPostProcessor package changes
>
> `BootstrapRegistry`, `BootstrapContext`, and `BootstrapRegistryInitializer`
> have moved from `org.springframework.boot` to `org.springframework.boot.bootstrap`.
>
> `EnvironmentPostProcessor` has moved from `org.springframework.boot.env` to
> `org.springframework.boot`.

### Checklist

Section: "BootstrapRegistry and EnvironmentPostProcessor package changes" -> EXTRACT: classes relocated to new packages (item 2); four FQNs named

### patterns.json

Each relocated class is a separate pattern. Include `source_artifact` so the verifier can confirm the FQN exists in the published JAR:

```json
{
  "source_pattern": "BootstrapRegistry package relocation",
  "source_fqn": "org.springframework.boot.BootstrapRegistry",
  "target_pattern": "org.springframework.boot.bootstrap.BootstrapRegistry",
  "location_type": "IMPORT",
  "source_artifact": {
    "group_id": "org.springframework.boot",
    "artifact_id": "spring-boot",
    "version": "3.5.0"
  },
  "rationale": "BootstrapRegistry moved from org.springframework.boot to org.springframework.boot.bootstrap",
  "complexity": "low",
  "category": "mandatory",
  "concern": "core",
  "provider_type": "java",
  "documentation_url": "https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide#bootstrapregistry-and-environmentpostprocessor-package-changes"
}
```

Note: `source_artifact` identifies the Maven artifact that contains the source FQN. The version is the source framework version (Spring Boot 3.5.x → `3.5.0`). The verifier downloads this JAR, runs `jar tf`, and confirms `org/springframework/boot/BootstrapRegistry.class` exists. If the FQN were hallucinated, the verifier would flag it as `not_found`.

How to determine `source_artifact`:
- The guide says "migrating from Spring Boot 3.x" → source version is `3.5.0`
- `org.springframework.boot.BootstrapRegistry` lives in the `spring-boot` module → `group_id: org.springframework.boot`, `artifact_id: spring-boot`
- If unsure which module contains the class, omit `source_artifact` (the verifier skips gracefully)

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: spring-boot-3-to-spring-boot-4-00010
  description: BootstrapRegistry moved from org.springframework.boot to org.springframework.boot.bootstrap
  category: mandatory
  effort: 3
  labels:
    - konveyor.io/source=spring-boot-3
    - konveyor.io/target=spring-boot-4
  links:
    - url: https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide#bootstrapregistry-and-environmentpostprocessor-package-changes
      title: Migration Documentation
  when:
    java.referenced:
      pattern: org.springframework.boot.BootstrapRegistry
      location: IMPORT
```

### Test Data (what triggers this rule)

```java
package com.example;

import org.springframework.boot.BootstrapRegistry;

public class AppInitializer implements BootstrapRegistry.InstanceSupplier<String> {
    @Override
    public String get(BootstrapRegistry registry) {
        return "initialized";
    }
}
```

---

## Example 5: Package-level consolidation with `PACKAGE` location

### Guide Excerpt

> ### HttpClient Migration
>
> Apache HttpClient 4.x (`org.apache.http`) is no longer supported. Remove
> old `org.apache.http` imports and re-import HttpClient classes from the
> `org.apache.hc.httpclient5` package namespace.

### Checklist

Section: "HttpClient Migration" -> EXTRACT: entire package renamed (item 2); one package-level rule is sufficient

### patterns.json

When an entire package is renamed or removed, create a **single** package-level pattern — do NOT create one rule per class. The `PACKAGE` location type matches any import from the old package.

```json
{
  "source_pattern": "Apache HttpClient 4.x package removed",
  "target_pattern": "org.apache.hc.httpclient5",
  "source_fqn": "org.apache.http",
  "location_type": "PACKAGE",
  "source_artifact": {
    "group_id": "org.apache.httpcomponents",
    "artifact_id": "httpclient",
    "version": "4.5.14"
  },
  "rationale": "Apache HttpClient 4.x (org.apache.http) is removed; re-import classes from org.apache.hc.httpclient5",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "http",
  "provider_type": "java",
  "documentation_url": "https://example.com/migration-guide#httpclient"
}
```

Note: `location_type: PACKAGE` makes `java.referenced` match any class imported from the `org.apache.http` package. This single rule replaces what would otherwise be dozens of per-class rules (HttpClient, HttpGet, HttpResponse, etc.) that all have the same migration action. Only use per-class rules when individual classes have DIFFERENT migration paths.

**When to consolidate vs per-class:**
- Guide says "re-import everything from package X to Y" → ONE package-level rule
- Guide says "ClassA moved to X, ClassB moved to Y, ClassC was removed" → separate per-class rules (different targets)

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: java-ee-to-quarkus-00050
  description: "Apache HttpClient 4.x (org.apache.http) is removed; re-import classes from org.apache.hc.httpclient5"
  category: mandatory
  effort: 5
  labels:
    - konveyor.io/source=java-ee
    - konveyor.io/target=quarkus
  links:
    - url: https://example.com/migration-guide#httpclient
      title: Migration Documentation
  when:
    java.referenced:
      pattern: org.apache.http
      location: PACKAGE
```

### Test Data (what triggers this rule)

```java
package com.example;

import org.apache.http.client.HttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.client.methods.HttpGet;

public class ApiClient {
    public void fetch() {
        HttpClient client = HttpClients.createDefault();
        HttpGet request = new HttpGet("https://api.example.com/data");
    }
}
```
