## Eval Judge Report -- scribe / opus / httpclient4-to-httpclient5

**Guide:** https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide (5 sub-pages: preparation, classic, async-simple, async-streaming, async-http2)
**Language:** java
**Ground truth:** japicmp-derived, 380 entries, reviewed 2026-05-27

### How the rules performed

| Metric | Value |
|--------|-------|
| Total rules | 30 |
| Quality score | avg 5.7/6 |
| Links present | 30/30 |
| Before/after code | 30/30 |
| Overlaps | 2 (SSLConnectionSocketFactory: 00004 import + 00016 constructor) |

### Rules that need attention

**26 of 30 rules passed.** The 4 below need fixes:

> **What "detection" and "guidance" mean:**
> - **Detection** = the rule's `when` condition -- does it find the right code?
> - **Guidance** = the rule's `message` -- does it tell the developer the correct fix?
>
> **Issue types:**
> - **Precision** -- detection is too broad (e.g., matches unrelated code). The guidance is still correct when the rule fires. Fix is usually mechanical.
> - **Coherence** -- detection and guidance don't match. The rule fires on one thing but advises about something else. Needs a design rethink.

#### Precision issues

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `httpcomponents-00023` | ok (warn) | ok | Pattern `closeExpiredConnections(*)` uses interface FQN `HttpClientConnectionManager` — kantra resolves from declared type, so calls on concrete `PoolingHttpClientConnectionManager` variable may not match | Acceptable risk; could add a second rule for `org.apache.http.impl.conn.PoolingHttpClientConnectionManager.closeExpiredConnections(*)` |
| `httpcomponents-00025` | ok (warn) | ok | Pattern `addInterceptorLast(*)` on `HttpClientBuilder` is correct FQN, but kantra may not match fluent chains where static type is inferred | Acceptable risk with current kantra limitations |

#### Coherence issues

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `httpcomponents-00025` | ok | narrow scope | Detection fires on any `addInterceptorLast()` call, but message only explains the logging use case (replace with `addExecInterceptorFirst`). Non-logging interceptors may need different replacement. | Broaden message: add "If this interceptor is not for logging, evaluate whether a protocol interceptor (`addRequestInterceptorLast`) or execution interceptor (`addExecInterceptorAfter`) is more appropriate in 5.x." |
| `httpcomponents-00010` | narrow | too broad | Detection targets only `org.apache.http.HttpEntity` import, but message discusses HttpResponse, HttpHost, HttpRequest, HttpEntityEnclosingRequest->HttpEntityContainer rename, getAllHeaders->getHeaders rename, and getStatusLine removal -- far more than what `HttpEntity` import implies. | Either (a) broaden pattern to `org.apache.http.Http*` or `org.apache.http.*` at IMPORT, or (b) narrow message to only cover HttpEntity package relocation to `org.apache.hc.core5.http.HttpEntity`. Option (b) recommended. |

#### Cross-rule issues

| Rules | What's wrong | Severity | How to fix |
|-------|-------------|----------|------------|
| `00004` + `00016` | Both target SSLConnectionSocketFactory with nearly identical messages. Import rule (00004) and constructor rule (00016) detect different code locations but give the same guidance. | warn | Differentiate: 00004 should focus on "this import has been removed, the replacement is `ClientTlsStrategyBuilder`"; 00016 should focus on "this constructor pattern should be replaced with the builder pattern shown below." |
| `00015` + `00024` | Both cover retry handler replacement. 00015 detects the constructor, 00024 detects `setRetryHandler()`. Messages overlap significantly. | warn | Add cross-reference: "See also rule for `setRetryHandler()`" in 00015 and "See also rule for `DefaultHttpRequestRetryHandler` constructor" in 00024. |
| `00028` + `00029` | Both cover timeout relocation from RequestConfig to ConnectionConfig. Messages are structurally identical. | warn | Minor; add "This is a companion change to setSocketTimeout relocation" in 00028 and vice versa. |

### Kantra-specific technical notes

1. **IMPORT-level wildcard patterns** (rules 00001-00009, 00011-00012, 00027): Kantra's `java.referenced` with `location: IMPORT` and wildcard patterns like `org.apache.http.client.methods.*` is valid syntax. The `*` matches any class within the package. These will correctly fire on `import org.apache.http.client.methods.HttpGet;` etc. **Verdict: correct.**

2. **METHOD_CALL fully qualified patterns** (rules 00020-00025, 00028-00029): Kantra requires FQN for `METHOD_CALL` location: `org.apache.http.HttpResponse.getStatusLine(*)`. The `(*)` wildcard matches any argument list. Kantra resolves the type from the declared variable type, not the runtime type. These patterns are syntactically correct and will match when the declared type matches the pattern's class. **Verdict: correct, with the caveat that type inference in fluent chains may miss some calls.**

3. **Maven XPath** (rule 00030): The XPath `//dependencies/dependency[groupId='org.apache.httpcomponents' and (artifactId='httpclient' or artifactId='httpcore' or artifactId='httpmime' or artifactId='httpasyncclient')]` is valid for `builtin.xml`. Kantra's XML provider applies XPath against pom.xml files. The `//` prefix ensures it matches regardless of namespace prefix presence. **Verdict: correct.**

4. **Single-class IMPORT patterns** (rules 00010, 00012, 00027): These use exact class names like `org.apache.http.HttpEntity` at IMPORT location. This only matches explicit imports of that exact class, not star-imports or other classes in the same package. **Verdict: correct and appropriately narrow.**

### Missing rules

These migration patterns from the guide have no corresponding rule. A missing rule means affected code gets no warning at all.

| What the guide says to migrate | Guide section | Impact | Suggested detection |
|-------------------------------|---------------|--------|---------------------|
| `org.apache.http.auth.*` (AuthScope, Credentials, UsernamePasswordCredentials, NTCredentials, etc.) | Classic APIs | high | `java.referenced` pattern `org.apache.http.auth.*` at IMPORT -- 22 classes in ground truth |
| `org.apache.http.cookie.*` (Cookie, CookieStore, CookieSpec, ClientCookie, etc.) | Classic APIs | high | `java.referenced` pattern `org.apache.http.cookie.*` at IMPORT -- 18 classes in ground truth |
| `org.apache.http.impl.auth.*` (BasicScheme, DigestScheme, NTLMScheme, etc.) | Classic APIs | high | `java.referenced` pattern `org.apache.http.impl.auth.*` at IMPORT -- 22 classes in ground truth |
| `org.apache.http.impl.cookie.*` (RFC6265CookieSpec, BasicClientCookie, etc.) | Classic APIs | medium | `java.referenced` pattern `org.apache.http.impl.cookie.*` at IMPORT -- 34 classes in ground truth, many RFC 2965 classes removed entirely |
| `org.apache.http.client.entity.*` (UrlEncodedFormEntity, EntityBuilder, GzipCompressingEntity, etc.) | Classic APIs | high | `java.referenced` pattern `org.apache.http.client.entity.*` at IMPORT -- 11 classes in ground truth |
| `org.apache.http.conn.routing.*` (HttpRoute, HttpRoutePlanner, RouteInfo, etc.) | Classic APIs | medium | `java.referenced` pattern `org.apache.http.conn.routing.*` at IMPORT -- 8 classes in ground truth |
| `org.apache.http.conn.scheme.*` (SchemeRegistry, PlainSocketFactory, etc.) | Classic APIs | medium | `java.referenced` pattern `org.apache.http.conn.scheme.*` at IMPORT -- 10 classes removed entirely |
| `org.apache.http.impl.conn.tsccm.*` (ThreadSafeClientConnManager, etc.) | Classic APIs | high | `java.referenced` pattern `org.apache.http.impl.conn.tsccm.*` at IMPORT -- removed entirely, replace with PoolingHttpClientConnectionManagerBuilder |
| `org.apache.http.impl.execchain.*` (execution chain internals) | Classic APIs | medium | `java.referenced` pattern `org.apache.http.impl.execchain.*` at IMPORT |
| `org.apache.http.conn.socket.*` (ConnectionSocketFactory, PlainConnectionSocketFactory) | Classic APIs | medium | `java.referenced` pattern `org.apache.http.conn.socket.*` at IMPORT -- 3 classes |
| `org.apache.http.conn.util.*` (PublicSuffixMatcher, InetAddressUtils, etc.) | Classic APIs | low | `java.referenced` pattern `org.apache.http.conn.util.*` at IMPORT -- 7 classes |
| `org.apache.http.conn.params.*` (ConnManagerParams, ConnRouteParams, etc.) | Classic APIs | medium | `java.referenced` pattern `org.apache.http.conn.params.*` at IMPORT -- 10 classes, all removed |
| `org.apache.http.client.params.*` (deprecated params classes) | Classic APIs | low | `java.referenced` pattern `org.apache.http.client.params.*` at IMPORT -- 7 classes, all removed |
| Async migration: `CloseableHttpAsyncClient`, `HttpAsyncClients.custom()`, `SimpleHttpRequest`, `SimpleRequestBuilder`, `FutureCallback` | Async Simple/Streaming | medium | These are new 5.x APIs, not detectable via old-API patterns. No rule needed unless detecting classic-to-async migration. |
| `client.close()` to `client.close(CloseMode.GRACEFUL)` | Async Simple/Streaming/HTTP2 | low | Not easily detectable -- `close()` is too generic a method name to target. |

### Ground truth coverage

- **Ground truth entries:** 380 (japicmp-derived, 379 package_change + 1 method_rename)
- **Distinct packages in ground truth:** ~47
- **Packages with dedicated IMPORT rules:** 12 (rules 00001-00012, 00027)
  - `org.apache.http.client.methods.*`, `org.apache.http.impl.client.*`, `org.apache.http.impl.conn.*`, `org.apache.http.conn.ssl.SSLConnectionSocketFactory`, `org.apache.http.entity.*`, `org.apache.http.client.config.*`, `org.apache.http.util.*`, `org.apache.http.message.*`, `org.apache.http.client.protocol.*`, `org.apache.http.HttpEntity`, `org.apache.http.client.utils.*`, `org.apache.http.client.ClientProtocolException`, `org.apache.http.config.SocketConfig`
- **Ground truth entries covered by import rules:** ~176/380 (46%)
- **Uncovered packages (13 packages, ~204 entries):** auth, cookie, impl.auth, impl.cookie, client.entity, conn.routing, conn.scheme, conn.socket, conn.util, conn.params, client.params, impl.conn.tsccm, impl.execchain

### Summary counts

- **2 precision issues**: closeExpiredConnections FQN resolution risk (warn); addInterceptorLast fluent chain resolution (warn)
- **2 coherence issues**: addInterceptorLast message covers only logging case (warn); HttpEntity import rule message scope far exceeds detection (warn)
- **3 cross-rule issues**: SSLConnectionSocketFactory duplicate messages (00004/00016, warn); retry handler overlap (00015/00024, warn); timeout relocation overlap (00028/00029, warn)
- **13 gaps**: 13 uncovered packages from the ground truth, covering ~204 classes. The most impactful are `org.apache.http.auth.*` (22 classes), `org.apache.http.client.entity.*` (11 classes), `org.apache.http.impl.auth.*` (22 classes), and `org.apache.http.impl.conn.tsccm.*` (removed entirely).

### Verdict

The 30 rules are well-crafted with accurate detection patterns and detailed migration guidance. All METHOD_CALL patterns use valid FQNs, the IMPORT wildcards are kantra-compatible, and the Maven XPath is correct. The main weakness is coverage: the rules address ~46% of the ground truth by package, missing auth, cookie, and several conn sub-packages. The coherence and precision issues are all warn-level -- no rule gives wrong advice. The cross-rule overlaps (SSLConnectionSocketFactory, retry handler, timeouts) are acceptable since they detect at different code locations, but messages should be differentiated.

---
*Generated by eval skill (LLM judge), 2026-06-01*
*Findings JSON: /tmp/eval-20260601-scribe-opus/findings.json*
