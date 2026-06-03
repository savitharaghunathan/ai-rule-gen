# Eval Details — httpclient4-to-httpclient5

Per-run eval judge results for each runtime × model combination. See [main benchmark results](../README.md) for summary tables and analysis.

---

## Claude Code / Sonnet — 43 rules, 42/43 pass

- **31 of 43 rules passed** eval judge review
- **4 precision issues** (all `warn`): unqualified METHOD_CALL rules (`00050`, `00060`, `00070`, `00080`) match 5.x replacement APIs — duplicates of more qualified rules (`00200`, `00210`, `00180`)
- **4 coherence issues** (1 `fail`, 3 `warn`): **00350** fires on classic `CloseableHttpClient` import but tells all users to switch to async client — wrong for the majority following the classic migration path. `00230` fires on any `URIUtils` import but only advises about `normalizeSyntax()`. `00340` fires on `PoolingHttpClientConnectionManager` but only covers async replacement. `00420` has implementation note in message referencing internal rule mechanics
- **3 cross-rule duplicates**: `00050`+`00200`, `00060`+`00210`, `00080`+`00180` — unqualified rules should be removed
- **1 gap**: no rule for `HttpContext.getAttribute(HTTP_TARGET_HOST)` → `HttpClientContext.getHttpRoute().getTargetHost()`

## Claude Code / Opus — 29 rules, 29/29 pass

- **9 precision issues** (all `warn`): every unqualified METHOD_CALL rule (`00050`–`00220`) — highest false-positive risk from `setConnectTimeout` and `setSocketTimeout` which are common across many Java frameworks
- **6 coherence issues** (2 `fail`, 4 `warn`): **00280** and **00290** fire on ALL HC4 `CloseableHttpClient`/`HttpClients` imports and tell developers to migrate to async classes — actively wrong for the majority following the classic-first migration path
- **2 cross-rule issues**: `00130` duplicates `00010` with wrong package namespace; `00280`+`00290` contradict classic migration guidance in `00010`
- **3 gaps**: no rule for `client.start()` async lifecycle requirement (high-severity silent runtime failure); no classic-path IMPORT rules for `CloseableHttpClient` and `HttpClients`

## Claude Code / Haiku — 26 rules, 0/26 pass eval judge

- **All 26 rules fail**: every message is empty (`': '`), all use `builtin.filecontent` instead of `java.referenced`
- **3 precision issues**: 2 rules target the new 5.x API (`ClassicRequestBuilder`), 1 matches JDK `SSLContext` (not HttpClient-specific)
- **5 duplicate groups** (11 rules): `00010`+`00090`, `00040`+`00050`, `00060`+`00070`, `00140`+`00200`, `00100`+`00210`+`00230`+`00240`+`00260`
- **Massive coverage gap**: only ~14 unique old APIs detected out of 380 ground truth entries (3.7% coverage)

## OpenCode / Opus — 29 rules, 29/29 pass

- **20 of 29 rules passed** eval judge review
- **8 precision issues**: unqualified METHOD_CALL rules — `setConnectTimeout`, `setSocketTimeout`, `setConnectionTimeToLive`, `addInterceptorLast`, `setRetryHandler`, `setSSLSocketFactory`, `closeExpiredConnections`, `closeIdleConnections`
- **3 coherence issues**: **00250** fires on classic `PoolingHttpClientConnectionManager` import but gives async-only guidance. **00260** fires on `CloseableHttpClient` import but says replace with `CloseableHttpAsyncClient` — wrong for classic users. **00270** same issue for `HttpClients`
- **2 cross-rule issues**: `00020`+`00280` SSL guidance overlap, `00250`+`00260`+`00270` async assumption cluster
- **8 gaps**: `CookieSpecs` replacement, `HttpGet/Put/Delete/Patch` constructor migration (only `HttpPost` covered), `ResponseHandler` pattern, `BasicCookieStore`/`BasicCredentialsProvider` package changes, classic `PoolingHttpClientConnectionManagerBuilder`, async `SimpleHttpRequest`, `IOReactorConfig`

## OpenCode / Gemini Pro — 33 rules, 33/33 pass

- **27 of 33 rules passed** eval judge review
- **1 precision issue**: `HttpPost` import rule (00270) broader than constructor-specific guidance — acceptable tradeoff
- **4 coherence issues**: SSLConnectionSocketFactory message omits TLS-on-connection-manager requirement (00220). `HttpResponse.getEntity()` message misleading — real issue is package change not method change (00130). `CloseableHttpClient` async-only advice fires on all projects (00330). Timeout rules (00160/00170) may be too narrow
- **1 cross-rule issue**: `00140`+`00150` getStatusLine/getStatusCode overlap with complementary but redundant messages
- **5 gaps**: Missing dedicated IMPORT rules for HttpGet/Put/Delete/Head/Patch/Options/Trace, RequestBuilder, CloseableHttpResponse, HttpClientContext; missing HttpContext.getAttribute recipe rule
- **Quality note**: Very poor links coverage (2/33 = 6%) — lowest of any passing run. Quality avg 4.45 is significantly below Sonnet/Opus runs (~5.9)

## OpenCode / Sonnet — 35 rules, 35/35 pass

- **22 of 35 rules passed** eval judge review
- **12 precision issues**: every unqualified METHOD_CALL rule — common method names like `setConnectTimeout`, `setSoTimeout`, `getAllHeaders`, `getRequestLine` match across many Java frameworks. Second-most precision issues after Goose/Opus (13).
- **5 coherence issues**: **00280** and **00290** push classic `CloseableHttpClient`/`HttpClients` users to async — wrong for classic migration path. **00300** and **00310** jump to async `SimpleRequestBuilder` for `HttpPost`/`HttpGet` instead of classic 5.x replacements. **00010**+**00120** duplicate package rules with wrong namespace (`org.apache.hc.httpclient5` instead of `org.apache.hc.client5`)
- **3 cross-rule issues**: `00010`+`00120` duplicate detection, `00280`+`00290` contradictory async push (fail), `00270`+`00320` overlapping async connection manager guidance
- **9 gaps**: `client.execute()` return type change, `SSLConnectionSocketFactory` removal, `StatusLine` removal, `PoolingHttpClientConnectionManager` classic migration, `HttpRequestRetryHandler` removal — 5 high-impact patterns missing

## Goose / Sonnet — 28 rules, 28/28 pass

- **14 of 24 rules passed** eval judge review (excludes ruleset + dependency rules)
- **7 precision issues**: unqualified METHOD_CALL rules — `getAllHeaders` (high false-positive risk), `setConnectTimeout` (high), `setSocketTimeout` (high), `closeExpiredConnections`, `closeIdleConnections`, `addInterceptorLast`, `setRetryHandler`
- **3 coherence issues**: **00260** fires on `CloseableHttpClient` import but gives async-only guidance. **00270** same for `HttpClients`. **00250** same for `PoolingHttpClientConnectionManager` — all three wrong for classic migration path
- **3 cross-rule issues**: `00010`+`00260`+`00270` contradictory guidance (package rule vs async rules), `00010` overlaps with all IMPORT rules, `00280`+`00260` both push async
- **8 gaps**: `HttpResponse` interface split (high), `ResponseHandler` pattern (high), `CookieSpecs`, `TimeUnit`→`Timeout`/`TimeValue`, `HttpGet/Put/Delete/Patch` constructors (only `HttpPost` covered), `EntityUtils`, `BasicNameValuePair`, async streaming consumers

## OpenCode / DeepSeek V3.2 — 35 rules, 34/35 pass

- **15 of 35 rules passed** eval judge review
- **9 precision issues** (3 error, 6 warn): 13 of 16 METHOD_CALL patterns use bare names — `setConnectTimeout`, `getStatusCode`, `getAllHeaders` match across Spring, OkHttp, JDBC. Rule `00010` (`org.apache.http*` at PACKAGE) is a noise-generating superset of all specific import rules
- **7 coherence issues** (3 error, 4 warn): **00300** async/classic confusion — detects classic `PoolingHttpClientConnectionManager` but recommends async replacement. **00060** detects wrong import path (`org.apache.http.protocol.HttpClientContext` instead of `org.apache.http.client.protocol.HttpClientContext`). **00080** failed kantra — CookieSpecs.STANDARD detection vs StandardCookieSpec.STRICT message mismatch. Most messages are boilerplate (`"<ClassName>: migration required for HttpClient 5.x"`) with no actionable detail
- **4 cross-rule issues**: `00010` PACKAGE rule subsumes all specific import rules (error), duplicate timeout rules, overlapping connection manager guidance
- **13 gaps**: Same 13 uncovered packages as other runs
- **Third-lowest quality** after Haiku (3.0) and Goose/Gemini (3.51): 25/35 missing before/after code, 3 missing links. DeepSeek generates correct rule structure but produces near-empty messages. 61.7 min — slowest of any httpclient run
- **Model tier**: DeepSeek V3.2 sits between Haiku (non-functional) and Gemini Pro (functional but weak). It follows pipeline structure and passes kantra (97%) but cannot produce meaningful migration guidance

## Goose / Opus — 28 rules, 28/28 pass

- **12 of 28 rules passed** eval judge review
- **13 precision issues**: 13 of 16 METHOD_CALL rules use bare method names without FQN — only `00070` (`getStatusLine`) and `00110` (`execute`) are fully qualified. 4 high-risk: `setConnectTimeout`, `setSocketTimeout`, `getAllHeaders`, `getStatusLine` (bare duplicate) match across many Java frameworks
- **4 coherence issues**: **00250** fires on classic `PoolingHttpClientConnectionManager` import but advises async `PoolingAsyncClientConnectionManager`. **00260** fires on classic `CloseableHttpClient` import but advises `CloseableHttpAsyncClient`. **00150** `addInterceptorLast` message only covers logging use case. **00010** has formatting artifact (duplicate heading)
- **3 cross-rule issues**: `00070`+`00240` both detect `getStatusLine` — one FQN, one bare (error-level overlap), `00010` HttpResponse import overlaps catch-all, `00130`+`00140` timeout companion rules with overlapping messages
- **13 gaps**: Same 13 uncovered packages as other Opus runs — auth, cookie, impl.auth, impl.cookie, client.entity, conn.routing, conn.scheme, conn.socket, conn.util, conn.params, client.params, impl.conn.tsccm, impl.execchain
- **Most precision issues** of any httpclient run (13). The bare METHOD_CALL pattern problem is more severe here than in CC/Opus (9) or Goose/Sonnet (7)

## Goose / Gemini Pro — 45 rules, 44/45 pass

- **29 of 45 rules passed** eval judge review
- **6 precision issues** (all `fail`): 6 rules target standard Java APIs unrelated to HttpClient — `java.util.concurrent.TimeUnit`, `com.fasterxml.jackson.core.JsonFactory`, `com.fasterxml.jackson.databind.ObjectMapper`, `com.fasterxml.jackson.databind.JsonNode`, `java.io.InputStream`, `java.util.concurrent.Future`. These fire on virtually every Java project. Gemini extracted patterns from the guide's code examples rather than from the migration instructions
- **4 coherence issues** (all `warn`): The 4 highest false-positive rules also have wrong messages — `TimeUnit` message describes HC5 best practice not migration, Jackson messages state obvious functionality, `InputStream` message is generic advice. None provide actual migration guidance
- **2 cross-rule issues**: `00050`+`00060` CredentialsProvider/BasicCredentialsProvider near-duplicate (warn), `00070`+`00080`+`00130` three Jackson rules all noise (fail)
- **8 gaps**: `HttpGet/Put/Delete` import rules (only `HttpPost` covered), `getRequestLine()` removal, `HttpResponse.getEntity()` type change, `setRetryHandler()→setRetryStrategy()`, `addInterceptorLast()→addExecInterceptorFirst()`, `getStatusCode()` chained call
- **Unique strength**: All 5 METHOD_CALL patterns use FQN — best of any httpclient run. Correct async/classic coherence (no classic→async confusion). 21 TYPE-level rules covering class relocations that other runs miss
- **Unique weakness**: The "standard Java API" false positive pattern is unique to Gemini — no other model generates rules for `InputStream`, `Future`, or `TimeUnit`

## Scribe / Sonnet — 14 rules, not kantra-tested

- **9 of 14 rules passed** eval judge review
- **2 precision issues**: catch-all `org.apache.http.*` import rule too broad for actionable guidance — fires on every HC4 import but only shows 4 example mappings (warn); `closeExpiredConnections` concrete FQN may miss interface-typed calls (warn)
- **3 coherence issues**: **class-004** uses wrong FQN (`org.apache.http.client.entity.HttpEntityEnclosingRequest` instead of `org.apache.http.HttpEntityEnclosingRequest`) — rule will silently never fire (fail). **interceptor-014** omits response interceptor guidance. **method-009** targets a 5.x deprecation (`getParams()`) rather than a 4.x→5.x migration pattern
- **1 cross-rule issue**: `method-007`+`async-011` both target `PoolingHttpClientConnectionManager` with overlapping guidance
- **29 gaps**: Same 13 uncovered packages as Scribe/Opus, plus 16 additional gaps from having fewer rules — missing per-package import rules, missing `setSocketTimeout`/`getRequestLine`/`closeIdleConnections` method rules, Maven rule covers only `httpclient` (not `httpcore`/`httpmime`/`httpasyncclient`)
- **Note**: Only 14 rules vs Scribe/Opus's 30. The catch-all `org.apache.http.*` import strategy gives ~7% actionable coverage vs Opus's 12 per-package rules covering 46%. Perfect 6.0 quality score — every rule has structured Before/After/Additional Info sections.

## Scribe / Opus — 30 rules, not kantra-tested

- **26 of 30 rules passed** eval judge review
- **2 precision issues**: `closeExpiredConnections` FQN resolution risk on concrete types (warn); `addInterceptorLast` fluent chain resolution (warn)
- **2 coherence issues**: `addInterceptorLast` message only covers logging use case but fires on all calls (warn); `HttpEntity` import rule (00010) message covers 6+ APIs far beyond what the import detects (warn)
- **3 cross-rule issues**: SSLConnectionSocketFactory duplicate between 00004/00016 (warn); retry handler overlap between 00015/00024 (warn); timeout relocation overlap between 00028/00029 (warn)
- **13 gaps**: 13 uncovered packages representing ~204/380 ground truth entries (46% coverage). Missing: `org.apache.http.auth.*`, `org.apache.http.client.entity.*`, `org.apache.http.impl.auth.*`, `org.apache.http.cookie.*`, `org.apache.http.impl.cookie.*`, `org.apache.http.impl.conn.tsccm.*`
- **Note**: Scribe is an MCP server (different pipeline architecture). Rules were NOT validated with kantra tests. Pass rate is n/a. Quality score (5.7) is high because all rules have links, before/after code, and effort ratings — but no functional validation was performed.
