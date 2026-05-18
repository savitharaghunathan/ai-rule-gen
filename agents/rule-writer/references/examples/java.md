# Java Extraction Examples

## Contents

- Example 1: `java.referenced` with CONSTRUCTOR_CALL location
- Example 2: `java.dependency` with version bounds
- Example 3: `builtin.xml` with XPath and namespaces
- Example 4: `java.referenced` with `source_artifact` for deterministic verification
- Example 5: Package-level consolidation with `PACKAGE` location
- Example 6: `java.referenced` with FIELD location
- Example 7: `java.referenced` with `annotated` sub-condition
- Example 8: `PACKAGE` with asterisk for subpackage matching

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
>
> The following table shows the key class mappings:
>
> | Old Class (4.x) | New Class (5.x) |
> |---|---|
> | `org.apache.http.client.HttpClient` | `org.apache.hc.client5.http.classic.HttpClient` |
> | `org.apache.http.client.methods.HttpGet` | `org.apache.hc.client5.http.classic.methods.HttpGet` |
> | `org.apache.http.impl.client.HttpClients` | `org.apache.hc.client5.http.impl.classic.HttpClients` |
> | `org.apache.http.HttpResponse` | `org.apache.hc.client5.http.classic.ClassicHttpResponse` |
> | `org.apache.http.HttpEntity` | `org.apache.hc.core5.http.HttpEntity` |
>
> **API changes:** In HttpClient 5, `HttpResponse.getStatusLine().getStatusCode()`
> has been replaced by `ClassicHttpResponse.getCode()`. The `StatusLine` class
> has been removed entirely.

### Checklist

Section: "HttpClient Migration" -> EXTRACT: entire package renamed (item 2); reference table illustrates the package rename — NOT separate patterns (item 4 exception); one genuine API change: getStatusLine() removed, replaced by getCode() (items 1, 4)

**Why the table does NOT produce 5 separate rules:**
- The lead paragraph says "re-import HttpClient classes from the `org.apache.hc.httpclient5` package namespace" — this is a package-level rename
- Every row in the table maps an old class under `org.apache.http` to a new class under `org.apache.hc` — same migration action (change the import)
- A single `PACKAGE` rule on `org.apache.http` catches all of these

**Why `getStatusLine()` DOES get its own rule:**
- The method name changed: `getStatusLine().getStatusCode()` → `getCode()`. This is not a simple import change — the method call itself must be rewritten
- The PACKAGE rule only flags the import; it cannot detect that `response.getStatusLine()` needs to become `response.getCode()`

### patterns.json

Two patterns — one PACKAGE rule for the namespace move, one METHOD_CALL for the genuine API change:

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

```json
{
  "source_pattern": "HttpResponse.getStatusLine() removed in HttpClient 5",
  "target_pattern": "ClassicHttpResponse.getCode()",
  "source_fqn": "org.apache.http.HttpResponse.getStatusLine",
  "location_type": "METHOD_CALL",
  "source_artifact": {
    "group_id": "org.apache.httpcomponents",
    "artifact_id": "httpclient",
    "version": "4.5.14"
  },
  "rationale": "HttpResponse.getStatusLine() is removed in HttpClient 5; use ClassicHttpResponse.getCode() instead",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "http",
  "provider_type": "java",
  "documentation_url": "https://example.com/migration-guide#httpclient"
}
```

**Why one PACKAGE rule instead of five:** The guide says "re-import HttpClient classes from `org.apache.hc.httpclient5`" — every row in the table has the same migration action (change the import prefix). One PACKAGE rule on `org.apache.http` catches all of them. Creating 5 separate rules from the table rows would be wrong — the table illustrates the package rename with examples, not 5 independent migration paths.

**Why `getStatusLine()` gets its own METHOD_CALL rule:** The method name changed (`getStatusLine().getStatusCode()` → `getCode()`). The PACKAGE rule flags the import but cannot detect that the method call itself must be rewritten.

**No asterisk needed here:** `org.apache.http` works without `*` because classes like `HttpResponse` and `HttpEntity` live directly in that package. See Example 8 for when `*` is required.

**Watch for class renames disguised as method rows:** A table row like `HttpResponse.getEntity()` → `ClassicHttpResponse.getEntity()` looks like a method-only change because `getEntity()` is unchanged. But the *class name* changed (`HttpResponse` → `ClassicHttpResponse`) — this needs an IMPORT rule, not a METHOD_CALL rule, and the PACKAGE rule does NOT cover it. The user can't find `HttpResponse` with `getEntity()` in the new package. Always decompose table rows: check the class name first, then the method name.

**When to consolidate vs per-class:**
- Guide says "re-import everything from package X to Y" → ONE package-level rule
- Guide says "ClassA moved to X, ClassB moved to Y, ClassC was removed" → separate per-class rules (different targets)
- Guide says "package X moved to Y" AND "method `foo()` renamed to `bar()`" → ONE package-level rule + ONE method-level rule for the rename

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

- ruleID: java-ee-to-quarkus-00060
  description: "HttpResponse.getStatusLine() is removed in HttpClient 5; use ClassicHttpResponse.getCode() instead"
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
      pattern: org.apache.http.HttpResponse.getStatusLine
      location: METHOD_CALL
```

### Test Data (what triggers these rules)

```java
package com.example;

import org.apache.http.client.HttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.HttpResponse;

public class ApiClient {
    public int fetch() throws Exception {
        HttpClient client = HttpClients.createDefault();
        HttpGet request = new HttpGet("https://api.example.com/data");
        HttpResponse response = client.execute(request);
        // PACKAGE rule catches all four imports above
        // METHOD_CALL rule catches this specific call:
        return response.getStatusLine().getStatusCode();
    }
}
```

---

## Example 6: `java.referenced` with FIELD location

### Guide Excerpt

> ### JMS Queue Injection
>
> JMS `Queue` and `Topic` field declarations must be replaced with
> SmallRye Reactive Messaging `Emitter` fields. Replace any field
> declared as `javax.jms.Queue` with an `@Channel`-annotated `Emitter`.
>
> ```java
> // Before
> @Resource(lookup = "java:/jms/queue/MyQueue")
> private Queue myQueue;
>
> // After
> @Channel("my-queue")
> Emitter<String> myQueueEmitter;
> ```

### Checklist

Section: "JMS Queue Injection" -> EXTRACT: field type replacement (item 1); the field's declared TYPE changed from `javax.jms.Queue` to `Emitter`

### patterns.json

```json
{
  "source_pattern": "JMS Queue field declaration must be replaced with Emitter",
  "target_pattern": "SmallRye Reactive Messaging Emitter",
  "source_fqn": "javax.jms.Queue",
  "location_type": "FIELD",
  "rationale": "JMS Queue fields must be replaced with SmallRye Reactive Messaging Emitter in Quarkus",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "messaging",
  "provider_type": "java",
  "documentation_url": "https://quarkus.io/guides/jms"
}
```

Note: `location_type: FIELD` matches when a **field is declared with** the specified type. The pattern `javax.jms.Queue` will match `private Queue myQueue;` because the field's declared type is `javax.jms.Queue`. `FIELD` and `FIELD_DECLARATION` are aliases — both map to the same analyzer behavior.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: jms-to-reactive-quarkus-00030
  description: JMS Queue fields must be replaced with SmallRye Reactive Messaging Emitter in Quarkus
  category: mandatory
  effort: 5
  labels:
    - konveyor.io/source=java-ee
    - konveyor.io/target=quarkus
  when:
    java.referenced:
      pattern: javax.jms.Queue
      location: FIELD
```

### Test Data (what triggers this rule)

```java
package com.example;

import javax.annotation.Resource;
import javax.jms.Queue;

public class MessageService {
    @Resource(lookup = "java:/jms/queue/MyQueue")
    private Queue myQueue;
}
```

### What FIELD does NOT match

**Common mistake:** Using `FIELD` to detect static field/constant access. For example, this rule would NOT work:

```yaml
# WRONG — FIELD does not match static constant access
when:
  java.referenced:
    pattern: org.apache.http.protocol.HttpCoreContext.HTTP_TARGET_HOST
    location: FIELD
```

The pattern `org.apache.http.protocol.HttpCoreContext.HTTP_TARGET_HOST` is a static constant, not a type. `FIELD` only matches field declarations by their **type** (e.g., `private HttpCoreContext ctx;` would match pattern `org.apache.http.protocol.HttpCoreContext`).

**Correct approach for static constant access:** Use `builtin.filecontent`:

```json
{
  "source_fqn": "HttpCoreContext\\.HTTP_TARGET_HOST",
  "file_pattern": ".*\\.java",
  "provider_type": "builtin",
  "rationale": "HttpCoreContext.HTTP_TARGET_HOST replaced by HttpClientContext.getHttpRoute().getTargetHost()"
}
```

---

## Example 7: `java.referenced` with `annotated` sub-condition

### Guide Excerpt

> ### MDB ActivationConfig for Queues
>
> Message-driven beans using `@ActivationConfigProperty` with
> `destinationLookup` must be migrated to SmallRye `@Incoming` channels.
>
> ```java
> // Before
> @MessageDriven(activationConfig = {
>     @ActivationConfigProperty(
>         propertyName = "destinationLookup",
>         propertyValue = "java:/jms/queue/MyQueue")
> })
> public class MyMDB implements MessageListener { ... }
>
> // After
> @ApplicationScoped
> public class MyConsumer {
>     @Incoming("my-queue")
>     public void onMessage(String message) { ... }
> }
> ```

### Checklist

Section: "MDB ActivationConfig for Queues" -> EXTRACT: annotation with specific element values (item 1); the `@ActivationConfigProperty` annotation with `propertyName="destinationLookup"` is the detectable signal

### patterns.json

```json
{
  "source_pattern": "@ActivationConfigProperty with destinationLookup",
  "target_pattern": "@Incoming channel annotation",
  "source_fqn": "javax.ejb.ActivationConfigProperty",
  "location_type": "ANNOTATION",
  "annotated_pattern": null,
  "annotated_elements": [
    {"name": "propertyName", "value": "destinationLookup"}
  ],
  "rationale": "MDB with destinationLookup ActivationConfigProperty must use SmallRye @Incoming",
  "complexity": "complex",
  "category": "mandatory",
  "concern": "messaging",
  "provider_type": "java",
  "documentation_url": "https://quarkus.io/guides/jms"
}
```

Note: The `annotated_elements` field filters the match to only `@ActivationConfigProperty` annotations where `propertyName` equals `"destinationLookup"`. Without this filter, the rule would match ALL `@ActivationConfigProperty` annotations regardless of their property name. The `annotated` sub-condition supports both `pattern` (FQN of a meta-annotation) and `elements` (name-value pairs for annotation element values).

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: jms-to-reactive-quarkus-00020
  description: MDB with destinationLookup ActivationConfigProperty must use SmallRye @Incoming
  category: mandatory
  effort: 7
  labels:
    - konveyor.io/source=java-ee
    - konveyor.io/target=quarkus
  when:
    java.referenced:
      pattern: javax.ejb.ActivationConfigProperty
      location: ANNOTATION
      annotated:
        elements:
          - name: propertyName
            value: destinationLookup
```

### Test Data (what triggers this rule)

```java
package com.example;

import javax.ejb.ActivationConfigProperty;
import javax.ejb.MessageDriven;
import javax.jms.Message;
import javax.jms.MessageListener;

@MessageDriven(activationConfig = {
    @ActivationConfigProperty(
        propertyName = "destinationLookup",
        propertyValue = "java:/jms/queue/MyQueue"),
    @ActivationConfigProperty(
        propertyName = "destinationType",
        propertyValue = "javax.jms.Queue")
})
public class MyMDB implements MessageListener {
    @Override
    public void onMessage(Message message) {
        // process message
    }
}
```

---

## Example 8: `PACKAGE` with asterisk for subpackage matching

### Guide Excerpt

> ### Upgrading Jackson
>
> Spring Boot 4 uses Jackson 3 as its preferred JSON library. Jackson 3 uses
> new group IDs and package names — `com.fasterxml.jackson` becomes
> `tools.jackson`. An exception is `jackson-annotations` which continues to
> use `com.fasterxml.jackson.core` group ID.

### Checklist

Section: "Upgrading Jackson" -> EXTRACT: entire package namespace renamed (item 2); `com.fasterxml.jackson` → `tools.jackson`

### patterns.json

```json
{
  "source_pattern": "Jackson 2 com.fasterxml.jackson packages replaced by tools.jackson",
  "target_pattern": "tools.jackson",
  "source_fqn": "com.fasterxml.jackson*",
  "location_type": "PACKAGE",
  "source_artifact": {
    "group_id": "com.fasterxml.jackson.core",
    "artifact_id": "jackson-databind",
    "version": "2.18.0"
  },
  "rationale": "Jackson 3 uses tools.jackson package namespace; com.fasterxml.jackson is replaced",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "jackson",
  "provider_type": "java",
  "documentation_url": "https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide#upgrading-jackson"
}
```

**Why the asterisk is required:** No class lives directly in `com.fasterxml.jackson` — all Jackson classes are in subpackages like `com.fasterxml.jackson.databind`, `com.fasterxml.jackson.core`, `com.fasterxml.jackson.annotation`. Without `*`, the pattern `com.fasterxml.jackson` matches nothing. Appending `*` makes it `com.fasterxml.jackson*`, which matches all subpackages.

**When to use `*` vs not:**
- `org.apache.http` (no `*`) — works because `HttpResponse`, `HttpEntity` live directly in that package
- `com.fasterxml.jackson*` (with `*`) — required because all classes are in subpackages (`databind`, `core`, etc.)
- When in doubt, always append `*` — it never hurts

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: spring-boot-3-to-spring-boot-4-00320
  description: Jackson 3 uses tools.jackson package; com.fasterxml.jackson packages replaced
  category: mandatory
  effort: 5
  labels:
    - konveyor.io/source=spring-boot-3
    - konveyor.io/target=spring-boot-4
  links:
    - url: https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide#upgrading-jackson
      title: Migration Documentation
  when:
    java.referenced:
      pattern: com.fasterxml.jackson*
      location: PACKAGE
```

### Test Data (what triggers this rule)

```java
package com.example;

import com.fasterxml.jackson.databind.ObjectMapper;

public class Application {
    public static void main(String[] args) {
        ObjectMapper mapper = new ObjectMapper();
    }
}
```
