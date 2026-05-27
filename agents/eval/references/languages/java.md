# Java — Eval Judge Language Reference

## Migration map example

When extracting actionable migration patterns from the guide, capture Java-specific details:

```
old_api: org.apache.http.HttpResponse.getStatusLine()
new_api: response.getCode() or new StatusLine(response)
guide_section: "Migration to classic APIs"
action_type: method_removal
severity: high
code_before: response.getStatusLine().getStatusCode()
code_after: response.getCode()
```

### Action types

- `class_rename` — class moved or renamed (e.g., `DefaultHttpClient` → `HttpClientBuilder`)
- `package_change` — entire package relocated (e.g., `org.apache.http.client` → `org.apache.hc.client5`)
- `method_rename` — method name changed, same semantics
- `method_removal` — method removed, replacement has different shape
- `signature_change` — same method name, different parameter types
- `behavioral_change` — same API surface, different runtime behavior
- `config_change` — configuration property or file format change
- `dependency_change` — Maven/Gradle coordinate change

### Sources to mine for patterns

- Migration recipes tables (these are gold — each row is a specific version mapping)
- Code snippets showing before/after
- Prose describing API changes
- Preparation steps listing deprecated APIs

## Condition accuracy — Java-specific checks

### Location type appropriateness

Java rules use `java.referenced` with a `location` field. Verify the location matches the detection intent:

| Location | Appropriate for | Example |
|----------|----------------|---------|
| `IMPORT` | Detecting use of an old class/package via its import | `import org.apache.http.HttpResponse;` |
| `TYPE` | Variable types, generics, casts | `HttpResponse response = ...;` |
| `METHOD_CALL` | Method invocations on a typed receiver | `response.getStatusLine()` |
| `ANNOTATION` | Annotation usage (not just import) | `@Stateless` |
| `CONSTRUCTOR_CALL` | `new` expressions | `new InitialContext()` |
| `INHERITANCE` | `extends` clause | `class X extends HttpServlet` |
| `IMPLEMENTS_TYPE` | `implements` clause | `class X implements SessionBean` |
| `FIELD` | Field declared with this type | `private DataSource ds;` |
| `PACKAGE` | Any reference to types in the package | `import org.apache.http.HttpResponse;` matches `org.apache.http` |

Common mistakes:
- Using `IMPORT` when `METHOD_CALL` is needed (detects the class import but not the specific method being migrated)
- Using `TYPE` for annotations (should be `ANNOTATION`)
- Using `METHOD` (definition site) when `METHOD_CALL` (invocation site) is intended

### FQN qualification

Check whether the pattern is fully qualified when it should be:

- A pattern like `getStatusLine` at `METHOD_CALL` matches ANY class with that method — flag as `warn` if the guide is specific about the owning class (e.g., `HttpResponse.getStatusLine()`)
- A pattern like `org.apache.http.HttpResponse.getStatusLine` at `METHOD_CALL` is properly qualified
- Short method names are acceptable when the method name is unique to the source library or when FQN matching has known limitations (builder chains, type hierarchy mismatches)

### Pattern breadth

- `org.apache.http*` at `PACKAGE` is intentionally broad — covers all types in the package hierarchy. Appropriate for package-level migration rules.
- `close` at `METHOD_CALL` is too broad — matches `close()` on any class (streams, connections, unrelated resources).
- For dependency rules (`java.dependency`), check that version bounds are semantically correct: removed artifacts use `lowerbound: 0.0.0` with no upperbound; same-artifact behavior changes use `upperbound` set to the target framework version.

## Calibration examples

### Good rule — pass/pass (silent in report)

Rule 00040 matches `org.apache.http.conn.ssl.SSLConnectionSocketFactory` at `IMPORT`. Message says to replace with `ClientTlsStrategyBuilder.create().setSslContext(...).setTlsVersions(TLS.V_1_3).buildClassic()`.

The guide says exactly this. FQN is fully qualified, location is correct, message includes concrete replacement. This rule does NOT appear in the Findings table — it passes both dimensions.

### Precision issue — detection too broad (appears under "Precision issues")

Rule 00080 matches `getStatusLine` at `METHOD_CALL`. The guide says `HttpResponse.getStatusLine()` is removed. The method name is correct but unqualified — could match any class. How this appears in the report:

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `00080` | too broad | ok | Unqualified `getStatusLine` matches any class, not just HttpResponse | Qualify with FQN: `org.apache.http.HttpResponse.getStatusLine` |

### Coherence issue — detection and guidance don't match (appears under "Coherence issues")

Hypothetical rule matches `org.apache.http.protocol.HttpCoreContext` at `IMPORT` but the message only describes the `getAttribute(HTTP_TARGET_HOST)` → `getHttpRoute().getTargetHost()` change. The condition fires on any import of HttpCoreContext, but the message only covers one use case — misleading for developers who import it for other reasons.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `00150` | ok | wrong scope | Import-level detection but message only covers target host retrieval | Narrow condition to target-host-specific detection; or broaden message to cover all HttpCoreContext migration |

### Wrong replacement — fail (appears under "Coherence issues")

Hypothetical rule says "replace `BasicHttpContext` with `BasicHttpContext` from the new package". The guide says replace with `HttpCoreContext` or `HttpClientContext` — it's a class rename, not a package move.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `00099` | ok | wrong API | Message says package move; guide says replacement is HttpCoreContext/HttpClientContext | Fix message: replacement is HttpCoreContext (core) or HttpClientContext (client), not a package move |

### Missing rule (appears under "Missing rules")

The guide's migration recipes table says `HttpClients.custom().setRetryHandler()` → `HttpClients.custom().setRetryStrategy()`. No rule covers this.

| What the guide says to migrate | Guide section | Impact | Suggested detection |
|-------------------------------|---------------|--------|---------------------|
| `setRetryHandler()` → `setRetryStrategy()` | Migration recipes table | high | `pattern: setRetryHandler`, `location: METHOD_CALL` — HttpRequestRetryHandler replaced by HttpRequestRetryStrategy |
