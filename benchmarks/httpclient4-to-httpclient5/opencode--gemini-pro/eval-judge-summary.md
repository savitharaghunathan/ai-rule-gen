## Eval Judge Report -- opencode / gemini-pro / httpclient4-to-httpclient5

**Guide:** https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide
**Language:** java
**Generator:** OpenCode with Gemini 3.1 Pro
**Ground truth:** japicmp-derived (380 entries, reviewed 2026-05-27)

### How the rules performed

| Metric | Value |
|--------|-------|
| Total rules | 33 |
| Kantra pass rate | 33/33 (100%) |
| Quality score | avg 4.45/6 |
| Rules with links | 2/33 (6%) |
| Rules with before/after guidance | 26/33 (79%) |

**Quality gaps:** 31 rules missing documentation links; 7 rules missing before/after code guidance.

### Rules that need attention

**27 of 33 rules passed.** The 6 below need fixes:

> **What "detection" and "guidance" mean:**
> - **Detection** = the rule's `when` condition -- does it find the right code?
> - **Guidance** = the rule's `message` -- does it tell the developer the correct fix?
>
> **Issue types:**
> - **Precision** -- detection is too broad (e.g., matches unrelated code). The guidance is still correct when it fires. Fix is usually mechanical.
> - **Coherence** -- detection and guidance don't match. The rule fires on one thing but advises about something else. Needs a design rethink.

#### Precision issues

These rules detect the right thing but cast too wide a net. Guidance is correct when they fire.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `httpclient4-to-httpclient5-00270` | too broad | ok | Fires on any `HttpPost` import but message only covers constructor replacement with `ClassicRequestBuilder.post()`. Code that subclasses `HttpPost` or references it in type declarations gets the same message even though the migration may differ. | Acceptable as-is -- `HttpPost` import is the primary detection point; subclassing is rare. No change needed. Severity: warn. |

#### Coherence issues

These rules have a mismatch between what they detect and what they advise. Developers may see confusing or irrelevant guidance.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `httpclient4-to-httpclient5-00220` | ok | incomplete | Fires on `SSLConnectionSocketFactory` import. Message says "replaced by `ClientTlsStrategyBuilder` (or `DefaultClientTlsStrategy`)" but does not explain that SSL config must now be set on the connection manager via `PoolingHttpClientConnectionManagerBuilder.setTlsSocketStrategy(...)`, not on the client builder directly. Guide says to use `ClientTlsStrategyBuilder.create()...buildClassic()` and pass to the connection manager builder. | Add to message: "Configure TLS on the connection manager: `PoolingHttpClientConnectionManagerBuilder.create().setTlsSocketStrategy(ClientTlsStrategyBuilder.create().setSslContext(...).setTlsVersions(TLS.V_1_3).buildClassic()).build()`". Severity: warn. |
| `httpclient4-to-httpclient5-00130` | wrong scope | ok | Fires on `HttpResponse.getEntity()` as METHOD_CALL but the guide says `HttpResponse.getEntity()` still works on `ClassicHttpResponse` (which extends `HttpResponse` in 5.x). The real issue is that `HttpResponse` in 5.x is in a different package (`org.apache.hc.core5.http`), and `getEntity()` is only on `ClassicHttpResponse`. The rule fires on any call to `getEntity()` on `org.apache.http.HttpResponse`, which is correct for detection. However, the message says "Cast to or use `ClassicHttpResponse.getEntity()`" -- this is misleading because after the package change, the developer will already be using the 5.x `ClassicHttpResponse`. The real migration is the package change, not a method call change. | Narrow message to: "After migrating to the `org.apache.hc.core5.http` package, `getEntity()` is available on `ClassicHttpResponse` (not the base `HttpResponse`). Ensure your response variable is typed as `ClassicHttpResponse`." Severity: warn. |
| `httpclient4-to-httpclient5-00330` | ok | wrong scope | Fires on `CloseableHttpClient` import with category `optional`. Message says "replace `CloseableHttpClient` with `CloseableHttpAsyncClient`". But migrating to async is a separate, optional migration path -- not the primary classic migration. The guide's classic migration keeps `CloseableHttpClient`. This rule fires on every project using `CloseableHttpClient` and tells them to switch to async, which is misleading for classic-path migrations. | Either: (a) change message to clarify this is only for the async migration path: "If migrating to the async API, replace `CloseableHttpClient` with `CloseableHttpAsyncClient` from `HttpAsyncClients.custom()`. For classic API migration, `CloseableHttpClient` remains but is now in `org.apache.hc.client5.http.impl.classic`." or (b) split into two rules -- one for the package change (mandatory) and one for the async option (optional). Severity: warn. |
| `httpclient4-to-httpclient5-00160` | narrow | ok | Fires on `RequestConfig.Builder.setConnectTimeout` as METHOD_CALL. The guide says connect timeout moved from `RequestConfig` to `ConnectionConfig`. The detection pattern `org.apache.http.client.config.RequestConfig.Builder.setConnectTimeout` is correct but very specific -- it targets the Builder's method. Code that sets connect timeout via other patterns (e.g., via `RequestConfig.custom().setConnectTimeout(int)`) should also match. The pattern should work since the analyzer resolves the Builder chain, but if the code uses `RequestConfig.Builder` as a separate variable, it might not resolve. | Monitor for missed detections. Pattern is likely fine for most usage. Severity: warn. |
| `httpclient4-to-httpclient5-00170` | narrow | ok | Same issue as 00160 but for `setSocketTimeout`. Fires on `RequestConfig.Builder.setSocketTimeout` as METHOD_CALL. | Same as 00160 -- monitor for missed detections. Severity: warn. |

### Cross-rule coherence

| Rule IDs | Issue | Severity | Fix type |
|----------|-------|----------|----------|
| `00140`, `00150` | Both cover `getStatusLine()` removal. Rule 00140 fires on `HttpResponse.getStatusLine` METHOD_CALL and advises using `getCode()` or `new StatusLine(response)`. Rule 00150 fires on `StatusLine.getStatusCode` METHOD_CALL and advises using `getCode()` directly. These are complementary, not contradictory -- 00140 catches the first call in the chain, 00150 catches the second. But a developer who has both will see two overlapping messages. | warn | reconcile_messages -- add cross-references: rule 00140 message should note "see also: `StatusLine.getStatusCode()` is replaced by `HttpResponse.getCode()`" and vice versa. |
| `00020`, `00230`-`00310` | The broad PACKAGE rule (00020) fires on `org.apache.http*` at PACKAGE level, which overlaps with every specific IMPORT-level rule (00230-00310, etc.). Developers will see the generic "re-import from 5.x namespace" message AND the specific class-level message. This is by design (layered rules) and not a defect -- the specific rules add value on top of the generic one. | warn | No fix needed -- this is the expected layered pattern. |

### Missing rules

These migration patterns from the guide have no corresponding rule. A missing rule means affected code gets no warning at all.

| What the guide says to migrate | Guide section | Impact | Suggested detection |
|-------------------------------|---------------|--------|---------------------|
| `HttpContext.getAttribute(HttpCoreContext.HTTP_TARGET_HOST)` replaced by `HttpClientContext.getHttpRoute().getTargetHost()` | Migration recipes | medium | `java.referenced` pattern: `org.apache.http.protocol.HttpContext.getAttribute` at METHOD_CALL; message: "Replace `HttpContext.getAttribute(HttpCoreContext.HTTP_TARGET_HOST)` with `HttpClientContext.getHttpRoute().getTargetHost()`." |
| `HttpMessage.getAllHeaders()` replaced by `MessageHeaders.getHeaders()` -- rule 00100 exists but uses METHOD_CALL on `org.apache.http.HttpMessage.getAllHeaders`. PASS -- covered. | Migration recipes | -- | Covered by rule 00100. |
| `HttpGet`, `HttpPut`, `HttpDelete`, `HttpHead`, `HttpPatch`, `HttpOptions`, `HttpTrace` -- all HTTP method classes relocated from `org.apache.http.client.methods` to `org.apache.hc.client5.http.classic.methods`. Only `HttpPost` has a dedicated rule (00270). The others are only caught by the broad PACKAGE rule (00020) with generic guidance. | Migration steps (package namespace change) | high | Add IMPORT-level rules for each: `org.apache.http.client.methods.HttpGet`, `org.apache.http.client.methods.HttpPut`, etc. Message: "Re-import from `org.apache.hc.client5.http.classic.methods.<ClassName>`. The API is largely compatible." |
| `RequestBuilder` relocated from `org.apache.http.client.methods.RequestBuilder` -- only caught by PACKAGE rule. The guide shows `ClassicRequestBuilder` as the 5.x replacement. | Migration steps | medium | `java.referenced` pattern: `org.apache.http.client.methods.RequestBuilder` at IMPORT; message: "Replace `RequestBuilder` with `ClassicRequestBuilder` from `org.apache.hc.client5.http.classic.methods`." |
| `Timeout` class usage for timeouts -- the guide says to "Use `Timeout` class to define timeouts" instead of raw `int` milliseconds. Rules 00160/00170 mention `ConnectionConfig` but don't explain the `Timeout` wrapper class. | Migration steps | medium | Not detectable as a standalone rule (no old API to match). Could be added as guidance in rules 00160/00170 messages: "Use `Timeout.ofMinutes(1)` or `Timeout.ofSeconds(30)` instead of raw int milliseconds." |
| `TimeValue` class for duration values -- guide says "Use `TimeValue` class to define time values (duration)." | Migration steps | low | Same as `Timeout` -- not easily detectable standalone. Add to rule 00200 message about TTL. |
| `PoolConcurrencyPolicy` and `PoolReusePolicy` configuration options -- new in 5.x, no old API to match. | Migration steps | low | Informational only -- no old API to detect. Could be mentioned in rule 00210 message. |
| `CloseableHttpResponse` relocated from `org.apache.http.client.methods.CloseableHttpResponse` | Migration steps | high | `java.referenced` pattern: `org.apache.http.client.methods.CloseableHttpResponse` at IMPORT; message: "Re-import from `org.apache.hc.client5.http.impl.classic.CloseableHttpResponse`." Currently only caught by PACKAGE rule. |
| `HttpClientContext` relocated from `org.apache.http.client.protocol.HttpClientContext` | Migration steps | high | `java.referenced` pattern: `org.apache.http.client.protocol.HttpClientContext` at IMPORT; message: "Re-import from `org.apache.hc.client5.http.protocol.HttpClientContext`." Currently only caught by PACKAGE rule. |

### Summary counts

| Category | Count |
|----------|-------|
| **Precision issues** | 1 |
| **Coherence issues** | 4 |
| **Cross-rule issues** | 1 (the 00140/00150 overlap; the 00020 layering is by design) |
| **Gaps (actionable, high/medium)** | 5 (HttpGet/Put/Delete/Head/Patch/Options/Trace batch counts as 1; RequestBuilder; CloseableHttpResponse; HttpClientContext; HttpContext.getAttribute) |

### Detailed verdict

- **27 of 33 rules passed** eval judge review
- **1 precision issue**: HttpPost import rule (00270) is broader than the constructor-specific guidance, but acceptable
- **4 coherence issues**: SSLConnectionSocketFactory message incomplete (00220); HttpResponse.getEntity message misleading (00130); CloseableHttpClient async-only advice fires on all projects (00330); timeout rules (00160/00170) may be too narrow
- **1 cross-rule issue**: getStatusLine/getStatusCode rules (00140/00150) overlap with complementary but redundant messages
- **5 gaps**: Missing dedicated IMPORT rules for HttpGet/Put/Delete/Head/Patch/Options/Trace, RequestBuilder, CloseableHttpResponse, HttpClientContext; missing HttpContext.getAttribute recipe rule

### Notes

- No language-specific reference file was available at `references/languages/java.md`. Review was conducted using generic guidance from the eval skill.
- Ground truth last reviewed 2026-05-27 -- within 90-day validity window.
- The ground truth is japicmp-derived (380 entries), almost entirely `package_change` type. The broad PACKAGE-level rule (00020) provides generic coverage for all 380 entries. The 32 specific rules cover the guide's explicit migration recipes and API changes. The 5 gaps identified are high-value classes explicitly mentioned in the guide or commonly used in real applications that would benefit from dedicated import-level rules with specific guidance rather than just the generic package-change message.
- Links coverage is very poor (2/33). This is a quality issue but not a judge finding -- the deterministic eval already flags this.
