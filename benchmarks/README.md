# Benchmark Results — Run 1

Comparison of rule generation quality across agent runtimes and LLM models.

## Methodology

- **Pipeline**: Skill-based runs (Claude Code, OpenCode, Goose) invoke the `/generate-rules` skill with the same migration guide URL. Scribe uses a separate MCP-based pipeline with no kantra testing.
- **Evaluation**: Deterministic eval (`cmd/eval`) + LLM judge (`agents/eval/SKILL.md`)
- **Quality Score**: Completeness metric (max 6 pts: message presence + links + effort + before/after guidance). Measures documentation completeness, not rule correctness.
- **Overlaps**: Count of rule pairs that fire on the same code. Overlaps can indicate specificity layering (a broad package-level rule + a specific method-level rule covering the same API), which improves developer experience. High overlap count is not inherently bad — it may mean better coverage through layered detection.
- **Eval Judge**: LLM-based review checking precision (false positive risk), coherence (detection/guidance alignment), cross-rule conflicts, and coverage gaps vs. the migration guide.
- **Timing**: Wall-clock from pipeline start to report completion. "Rule Gen" = ingest through construct stages. "Test" = scaffold through kantra validation. Scribe has no test phase.
- **Gaps**: Migration patterns from the guide with no corresponding rule. "Adjusted Gaps" accounts for broad package-level rules (e.g., `org.apache.http*` at PACKAGE) that provide generic coverage for all sub-packages — these are not counted as gaps even though guidance is not package-specific.
- **Date**: May–June 2026

## Runtime × Model Matrix

| Runtime | Model | Model ID | Provider |
|---------|-------|----------|----------|
| Claude Code | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Claude Code | Opus | claude-opus-4-6 | Anthropic |
| Claude Code | Haiku | claude-haiku-4-5-20251001 | Anthropic |
| OpenCode | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| OpenCode | Opus | claude-opus-4-6 | Anthropic |
| OpenCode | Gemini Pro | google-vertex/gemini-3.1-pro-preview¹ | Google |
| Goose | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Goose | Opus | claude-opus-4-6 | Anthropic |
| Goose | Gemini Pro | gemini-2.5-pro | Google |
| OpenCode | DeepSeek V3.2 | google-vertex/deepseek-ai/deepseek-v3.2-maas | Google (Vertex) |
| Scribe (MCP) | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Scribe (MCP) | Opus | claude-opus-4-6 | Anthropic |

¹ OpenCode's Gemini Pro model ID (`gemini-3.1-pro-preview`) does not match a known public Gemini model. Likely the same `gemini-2.5-pro` used by Goose, accessed via a different Vertex AI endpoint. Unverified.

## httpclient4-to-httpclient5

| Runtime | Model | Rules | Pass Rate | Quality | Rule Gen (min) | Total (min) | Precision | Coherence | Cross-Rule | Gaps | Adj. Gaps |
|---------|-------|-------|-----------|---------|----------------|-------------|-----------|-----------|------------|------|-----------|
| claude-code | sonnet | 43 | 42/43 | 5.95 | 13.2 | 34.6 | 4 | 4 (1 fail) | 3 | 1 | 0 |
| claude-code | opus | 29 | 29/29 | 5.93 | 12.3 | 28.5 | 9 | 6 (2 fail) | 2 | 3 | 0 |
| claude-code | haiku | 26 | 26/26 | 3.0 | — | 4.0 | 3 | 26 (all fail) | 5 | 9 | — |
| opencode | sonnet | 35 | 35/35 | 5.91 | 8.4 | 49.2 | 12 | 5 | 3 | 9 | 0 |
| opencode | opus | 29 | 29/29 | 5.90 | 7.4 | 21.5 | 8 | 3 | 2 | 8 | 8 |
| opencode | gemini-pro | 33 | 33/33 | 4.45 | — | 11.1 | 1 | 4 | 1 | 5 | — |
| opencode | deepseek-v3 | 35 | 34/35 | 4.3 | — | 61.7 | 9 | 7 | 4 | 13 | — |
| goose | sonnet | 28 | 28/28 | 6.0 | 11.3 | 21.2 | 7 | 3 | 3 | 8 | 0 |
| goose | opus | 28 | 28/28 | 5.86 | 12.6 | 24.3 | 13 | 4 | 3 | 13 | 13 |
| goose | gemini-pro | 45 | 44/45 | 3.51 | — | — | 6 | 4 | 2 | 8 | — |
| scribe | sonnet | 14 | n/a | 6.0 | ~2 | ~2 | 2 | 3 | 1 | 29 | ~16 |
| scribe | opus | 30 | n/a | 5.7 | ~3 | ~3 | 2 | 2 | 3 | 13 | 13 |

### Eval Details (httpclient4→5)

#### Claude Code / Sonnet — 43 rules, 42/43 pass

- **31 of 43 rules passed** eval judge review
- **4 precision issues** (all `warn`): unqualified METHOD_CALL rules (`00050`, `00060`, `00070`, `00080`) match 5.x replacement APIs — duplicates of more qualified rules (`00200`, `00210`, `00180`)
- **4 coherence issues** (1 `fail`, 3 `warn`): **00350** fires on classic `CloseableHttpClient` import but tells all users to switch to async client — wrong for the majority following the classic migration path. `00230` fires on any `URIUtils` import but only advises about `normalizeSyntax()`. `00340` fires on `PoolingHttpClientConnectionManager` but only covers async replacement. `00420` has implementation note in message referencing internal rule mechanics
- **3 cross-rule duplicates**: `00050`+`00200`, `00060`+`00210`, `00080`+`00180` — unqualified rules should be removed
- **1 gap**: no rule for `HttpContext.getAttribute(HTTP_TARGET_HOST)` → `HttpClientContext.getHttpRoute().getTargetHost()`

#### Claude Code / Opus — 29 rules, 29/29 pass

- **9 precision issues** (all `warn`): every unqualified METHOD_CALL rule (`00050`–`00220`) — highest false-positive risk from `setConnectTimeout` and `setSocketTimeout` which are common across many Java frameworks
- **6 coherence issues** (2 `fail`, 4 `warn`): **00280** and **00290** fire on ALL HC4 `CloseableHttpClient`/`HttpClients` imports and tell developers to migrate to async classes — actively wrong for the majority following the classic-first migration path
- **2 cross-rule issues**: `00130` duplicates `00010` with wrong package namespace; `00280`+`00290` contradict classic migration guidance in `00010`
- **3 gaps**: no rule for `client.start()` async lifecycle requirement (high-severity silent runtime failure); no classic-path IMPORT rules for `CloseableHttpClient` and `HttpClients`

#### Claude Code / Haiku — 26 rules, 0/26 pass eval judge

- **All 26 rules fail**: every message is empty (`': '`), all use `builtin.filecontent` instead of `java.referenced`
- **3 precision issues**: 2 rules target the new 5.x API (`ClassicRequestBuilder`), 1 matches JDK `SSLContext` (not HttpClient-specific)
- **5 duplicate groups** (11 rules): `00010`+`00090`, `00040`+`00050`, `00060`+`00070`, `00140`+`00200`, `00100`+`00210`+`00230`+`00240`+`00260`
- **Massive coverage gap**: only ~14 unique old APIs detected out of 380 ground truth entries (3.7% coverage)

#### OpenCode / Opus — 29 rules, 29/29 pass

- **20 of 29 rules passed** eval judge review
- **8 precision issues**: unqualified METHOD_CALL rules — `setConnectTimeout`, `setSocketTimeout`, `setConnectionTimeToLive`, `addInterceptorLast`, `setRetryHandler`, `setSSLSocketFactory`, `closeExpiredConnections`, `closeIdleConnections`
- **3 coherence issues**: **00250** fires on classic `PoolingHttpClientConnectionManager` import but gives async-only guidance. **00260** fires on `CloseableHttpClient` import but says replace with `CloseableHttpAsyncClient` — wrong for classic users. **00270** same issue for `HttpClients`
- **2 cross-rule issues**: `00020`+`00280` SSL guidance overlap, `00250`+`00260`+`00270` async assumption cluster
- **8 gaps**: `CookieSpecs` replacement, `HttpGet/Put/Delete/Patch` constructor migration (only `HttpPost` covered), `ResponseHandler` pattern, `BasicCookieStore`/`BasicCredentialsProvider` package changes, classic `PoolingHttpClientConnectionManagerBuilder`, async `SimpleHttpRequest`, `IOReactorConfig`

#### OpenCode / Gemini Pro — 33 rules, 33/33 pass

- **27 of 33 rules passed** eval judge review
- **1 precision issue**: `HttpPost` import rule (00270) broader than constructor-specific guidance — acceptable tradeoff
- **4 coherence issues**: SSLConnectionSocketFactory message omits TLS-on-connection-manager requirement (00220). `HttpResponse.getEntity()` message misleading — real issue is package change not method change (00130). `CloseableHttpClient` async-only advice fires on all projects (00330). Timeout rules (00160/00170) may be too narrow
- **1 cross-rule issue**: `00140`+`00150` getStatusLine/getStatusCode overlap with complementary but redundant messages
- **5 gaps**: Missing dedicated IMPORT rules for HttpGet/Put/Delete/Head/Patch/Options/Trace, RequestBuilder, CloseableHttpResponse, HttpClientContext; missing HttpContext.getAttribute recipe rule
- **Quality note**: Very poor links coverage (2/33 = 6%) — lowest of any passing run. Quality avg 4.45 is significantly below Sonnet/Opus runs (~5.9)

#### OpenCode / Sonnet — 35 rules, 35/35 pass

- **22 of 35 rules passed** eval judge review
- **12 precision issues**: every unqualified METHOD_CALL rule — common method names like `setConnectTimeout`, `setSoTimeout`, `getAllHeaders`, `getRequestLine` match across many Java frameworks. Second-most precision issues after Goose/Opus (13).
- **5 coherence issues**: **00280** and **00290** push classic `CloseableHttpClient`/`HttpClients` users to async — wrong for classic migration path. **00300** and **00310** jump to async `SimpleRequestBuilder` for `HttpPost`/`HttpGet` instead of classic 5.x replacements. **00010**+**00120** duplicate package rules with wrong namespace (`org.apache.hc.httpclient5` instead of `org.apache.hc.client5`)
- **3 cross-rule issues**: `00010`+`00120` duplicate detection, `00280`+`00290` contradictory async push (fail), `00270`+`00320` overlapping async connection manager guidance
- **9 gaps**: `client.execute()` return type change, `SSLConnectionSocketFactory` removal, `StatusLine` removal, `PoolingHttpClientConnectionManager` classic migration, `HttpRequestRetryHandler` removal — 5 high-impact patterns missing

#### Goose / Sonnet — 28 rules, 28/28 pass

- **14 of 24 rules passed** eval judge review (excludes ruleset + dependency rules)
- **7 precision issues**: unqualified METHOD_CALL rules — `getAllHeaders` (high false-positive risk), `setConnectTimeout` (high), `setSocketTimeout` (high), `closeExpiredConnections`, `closeIdleConnections`, `addInterceptorLast`, `setRetryHandler`
- **3 coherence issues**: **00260** fires on `CloseableHttpClient` import but gives async-only guidance. **00270** same for `HttpClients`. **00250** same for `PoolingHttpClientConnectionManager` — all three wrong for classic migration path
- **3 cross-rule issues**: `00010`+`00260`+`00270` contradictory guidance (package rule vs async rules), `00010` overlaps with all IMPORT rules, `00280`+`00260` both push async
- **8 gaps**: `HttpResponse` interface split (high), `ResponseHandler` pattern (high), `CookieSpecs`, `TimeUnit`→`Timeout`/`TimeValue`, `HttpGet/Put/Delete/Patch` constructors (only `HttpPost` covered), `EntityUtils`, `BasicNameValuePair`, async streaming consumers

#### OpenCode / DeepSeek V3.2 — 35 rules, 34/35 pass

- **15 of 35 rules passed** eval judge review
- **9 precision issues** (3 error, 6 warn): 13 of 16 METHOD_CALL patterns use bare names — `setConnectTimeout`, `getStatusCode`, `getAllHeaders` match across Spring, OkHttp, JDBC. Rule `00010` (`org.apache.http*` at PACKAGE) is a noise-generating superset of all specific import rules
- **7 coherence issues** (3 error, 4 warn): **00300** async/classic confusion — detects classic `PoolingHttpClientConnectionManager` but recommends async replacement. **00060** detects wrong import path (`org.apache.http.protocol.HttpClientContext` instead of `org.apache.http.client.protocol.HttpClientContext`). **00080** failed kantra — CookieSpecs.STANDARD detection vs StandardCookieSpec.STRICT message mismatch. Most messages are boilerplate (`"<ClassName>: migration required for HttpClient 5.x"`) with no actionable detail
- **4 cross-rule issues**: `00010` PACKAGE rule subsumes all specific import rules (error), duplicate timeout rules, overlapping connection manager guidance
- **13 gaps**: Same 13 uncovered packages as other runs
- **Third-lowest quality** after Haiku (3.0) and Goose/Gemini (3.51): 25/35 missing before/after code, 3 missing links. DeepSeek generates correct rule structure but produces near-empty messages. 61.7 min — slowest of any httpclient run
- **Model tier**: DeepSeek V3.2 sits between Haiku (non-functional) and Gemini Pro (functional but weak). It follows pipeline structure and passes kantra (97%) but cannot produce meaningful migration guidance

#### Goose / Opus — 28 rules, 28/28 pass

- **12 of 28 rules passed** eval judge review
- **13 precision issues**: 13 of 16 METHOD_CALL rules use bare method names without FQN — only `00070` (`getStatusLine`) and `00110` (`execute`) are fully qualified. 4 high-risk: `setConnectTimeout`, `setSocketTimeout`, `getAllHeaders`, `getStatusLine` (bare duplicate) match across many Java frameworks
- **4 coherence issues**: **00250** fires on classic `PoolingHttpClientConnectionManager` import but advises async `PoolingAsyncClientConnectionManager`. **00260** fires on classic `CloseableHttpClient` import but advises `CloseableHttpAsyncClient`. **00150** `addInterceptorLast` message only covers logging use case. **00010** has formatting artifact (duplicate heading)
- **3 cross-rule issues**: `00070`+`00240` both detect `getStatusLine` — one FQN, one bare (error-level overlap), `00010` HttpResponse import overlaps catch-all, `00130`+`00140` timeout companion rules with overlapping messages
- **13 gaps**: Same 13 uncovered packages as other Opus runs — auth, cookie, impl.auth, impl.cookie, client.entity, conn.routing, conn.scheme, conn.socket, conn.util, conn.params, client.params, impl.conn.tsccm, impl.execchain
- **Most precision issues** of any httpclient run (13). The bare METHOD_CALL pattern problem is more severe here than in CC/Opus (9) or Goose/Sonnet (7)

#### Goose / Gemini Pro — 45 rules, 44/45 pass

- **29 of 45 rules passed** eval judge review
- **6 precision issues** (all `fail`): 6 rules target standard Java APIs unrelated to HttpClient — `java.util.concurrent.TimeUnit`, `com.fasterxml.jackson.core.JsonFactory`, `com.fasterxml.jackson.databind.ObjectMapper`, `com.fasterxml.jackson.databind.JsonNode`, `java.io.InputStream`, `java.util.concurrent.Future`. These fire on virtually every Java project. Gemini extracted patterns from the guide's code examples rather than from the migration instructions
- **4 coherence issues** (all `warn`): The 4 highest false-positive rules also have wrong messages — `TimeUnit` message describes HC5 best practice not migration, Jackson messages state obvious functionality, `InputStream` message is generic advice. None provide actual migration guidance
- **2 cross-rule issues**: `00050`+`00060` CredentialsProvider/BasicCredentialsProvider near-duplicate (warn), `00070`+`00080`+`00130` three Jackson rules all noise (fail)
- **8 gaps**: `HttpGet/Put/Delete` import rules (only `HttpPost` covered), `getRequestLine()` removal, `HttpResponse.getEntity()` type change, `setRetryHandler()→setRetryStrategy()`, `addInterceptorLast()→addExecInterceptorFirst()`, `getStatusCode()` chained call
- **Unique strength**: All 5 METHOD_CALL patterns use FQN — best of any httpclient run. Correct async/classic coherence (no classic→async confusion). 21 TYPE-level rules covering class relocations that other runs miss
- **Unique weakness**: The "standard Java API" false positive pattern is unique to Gemini — no other model generates rules for `InputStream`, `Future`, or `TimeUnit`

#### Scribe / Sonnet — 14 rules, not kantra-tested

- **9 of 14 rules passed** eval judge review
- **2 precision issues**: catch-all `org.apache.http.*` import rule too broad for actionable guidance — fires on every HC4 import but only shows 4 example mappings (warn); `closeExpiredConnections` concrete FQN may miss interface-typed calls (warn)
- **3 coherence issues**: **class-004** uses wrong FQN (`org.apache.http.client.entity.HttpEntityEnclosingRequest` instead of `org.apache.http.HttpEntityEnclosingRequest`) — rule will silently never fire (fail). **interceptor-014** omits response interceptor guidance. **method-009** targets a 5.x deprecation (`getParams()`) rather than a 4.x→5.x migration pattern
- **1 cross-rule issue**: `method-007`+`async-011` both target `PoolingHttpClientConnectionManager` with overlapping guidance
- **29 gaps**: Same 13 uncovered packages as Scribe/Opus, plus 16 additional gaps from having fewer rules — missing per-package import rules, missing `setSocketTimeout`/`getRequestLine`/`closeIdleConnections` method rules, Maven rule covers only `httpclient` (not `httpcore`/`httpmime`/`httpasyncclient`)
- **Note**: Only 14 rules vs Scribe/Opus's 30. The catch-all `org.apache.http.*` import strategy gives ~7% actionable coverage vs Opus's 12 per-package rules covering 46%. Perfect 6.0 quality score — every rule has structured Before/After/Additional Info sections.

#### Scribe / Opus — 30 rules, not kantra-tested

- **26 of 30 rules passed** eval judge review
- **2 precision issues**: `closeExpiredConnections` FQN resolution risk on concrete types (warn); `addInterceptorLast` fluent chain resolution (warn)
- **2 coherence issues**: `addInterceptorLast` message only covers logging use case but fires on all calls (warn); `HttpEntity` import rule (00010) message covers 6+ APIs far beyond what the import detects (warn)
- **3 cross-rule issues**: SSLConnectionSocketFactory duplicate between 00004/00016 (warn); retry handler overlap between 00015/00024 (warn); timeout relocation overlap between 00028/00029 (warn)
- **13 gaps**: 13 uncovered packages representing ~204/380 ground truth entries (46% coverage). Missing: `org.apache.http.auth.*`, `org.apache.http.client.entity.*`, `org.apache.http.impl.auth.*`, `org.apache.http.cookie.*`, `org.apache.http.impl.cookie.*`, `org.apache.http.impl.conn.tsccm.*`
- **Note**: Scribe is an MCP server (different pipeline architecture). Rules were NOT validated with kantra tests. Pass rate is n/a. Quality score (5.7) is high because all rules have links, before/after code, and effort ratings — but no functional validation was performed.

### Key Findings (httpclient4→5)

**Claude Code / Sonnet** generates 43 rules with quality 5.95 and the fewest precision issues (4) among Sonnet/Opus skill runs. One rule fails kantra tests. Rule gen takes 13.2 min; testing adds another 21 min.

**Claude Code / Opus** generates fewer rules (29) but is more conservative. Higher overlap count reflects specificity layering. More precision issues from unqualified METHOD_CALL conditions. Both CC runs have a broad `org.apache.http*` PACKAGE rule that gives zero adjusted gaps.

**Claude Code / Haiku** produces a non-functional ruleset: all 26 rules have empty messages, use wrong condition types, and include 5 duplicate groups.

**OpenCode / Opus** generates exactly the same rule count as Claude Code / Opus (29) with identical pass rate. 8 unqualified METHOD_CALL precision issues and the same async/classic coherence pattern. No catch-all rule — 8 real gaps.

**OpenCode / Sonnet** generates 35 rules with 12 precision issues — second-most after Goose/Opus (13). Slowest Sonnet run at 49 min total (vs CC/Sonnet 35 min, Goose/Sonnet 21 min). Same async/classic coherence issue plus a wrong namespace in a duplicate package rule.

**OpenCode / DeepSeek V3.2** generates 35 rules with third-lowest quality (4.3) after Haiku (3.0) and Goose/Gemini (3.51). Messages are boilerplate with no actionable detail — 25/35 missing before/after code. Slowest run at 61.7 min. Sits in the "follows instructions but can't reason deeply" tier.

**Goose / Sonnet** generates the fewest rules (28) but achieves the highest quality avg (6.0) — every rule has complete documentation (links, effort, before/after guidance). Has a catch-all PACKAGE rule — zero adjusted gaps.

**Goose / Opus** has the most precision issues of any httpclient run (13) — 13/16 METHOD_CALL patterns are bare names. No catch-all rule — 13 real gaps.

**Goose / Gemini Pro** generates the most rules (45) with perfect METHOD_CALL FQN qualification (5/5). However, 6 rules detect standard Java APIs (Jackson, InputStream, Future, TimeUnit) unrelated to HttpClient — a unique Gemini failure mode where it extracts patterns from code examples rather than migration instructions. Quality 3.51 (second-lowest after Haiku) driven by 42/45 missing before/after code.

**Sonnet across runtimes** (httpclient4→5):

| Runtime | Rules | Quality | Precision | Coherence | Adj. Gaps |
|---------|-------|---------|-----------|-----------|-----------|
| Claude Code | 43 | 5.95 | 4 | 4 | 0 |
| OpenCode | 35 | 5.91 | 12 | 5 | 0 |
| Goose | 28 | 6.0 | 7 | 3 | 0 |

All three Sonnet runs have package-level catch-all rules — zero adjusted gaps. Claude Code extracts significantly more rules (43) with fewest issues. Goose produces the fewest rules but highest documentation quality.

**Scribe comparison** (httpclient4→5):

| Metric | CC/Sonnet (skill) | Scribe/Opus (MCP) |
|--------|-------------------|-------------------|
| Rules | 43 | 30 |
| Adj. gaps | 0 | 13 |
| Rule gen time | 13.2 min | ~3 min |
| Total time | 34.6 min | ~3 min |
| Precision | 4 | 2 |
| Coherence | 4 (1 fail) | 2 |
| Kantra validated | 42/43 | none |

CC/Sonnet (skill) generates more rules with zero adjusted gaps but has double the precision/coherence issues. Scribe/Opus (MCP) produces cleaner individual rules but has 13 real gaps — no catch-all rule covers the missing packages. Rule generation alone is 13.2 min vs ~3 min. The skill pipeline's additional 21 min is kantra testing, which catches broken rules like Scribe/Sonnet's `class-004` (wrong FQN, never fires) before they ship.

**Shared issue**: All Sonnet and Opus runs generate rules that tell classic `CloseableHttpClient` users to switch to async, contradicting the migration guide's classic-first path. This is a consistent LLM blind spot regardless of runtime.

### Interesting Findings (cross-migration)

1. **Model capability has a hard threshold for agentic pipelines.** Haiku can follow instructions and produce syntactically valid YAML, but it cannot reason through the multi-step pipeline (ingest guide → extract patterns → construct rules with proper `java.referenced` conditions → generate test data → iterate on failures). It defaults to the simplest possible rule structure (`builtin.filecontent` with empty messages) and never self-corrects. This suggests agentic coding pipelines need a minimum model capability tier — there is no graceful degradation, just a cliff.

2. **More rules ≠ more noise (httpclient).** CC/Sonnet generates 48% more rules than CC/Opus (43 vs 29) but has proportionally *fewer* eval judge issues. The additional rules cover simple class relocations and API renames that Opus skips entirely. Broader extraction captures the long tail of migration patterns without sacrificing precision.

3. **Unqualified METHOD_CALL is the top precision pitfall.** Across both migrations, rules that match method names like `setConnectTimeout` without qualifying the parent class are the #1 source of false-positive risk. These fire on unrelated frameworks (Spring, OkHttp, JDBC) and on the replacement APIs the rule tells you to adopt.

4. **The async/classic migration path is a consistent LLM blind spot (httpclient).** Every runtime and model produces rules telling `CloseableHttpClient` users to switch to `CloseableHttpAsyncClient`. The migration guide explicitly recommends migrating to the 5.x *classic* API first, then optionally to async. This is the highest-severity issue across all httpclient rulesets.

5. **Overlaps are a feature, not a bug.** Specificity layering — a broad package-level import rule plus specific method-level rules for the same API — gives developers both a high-level "this package moved" warning and targeted "change this specific call" guidance.

6. **Cost-quality tradeoff is stark.** Haiku costs ~20x less per token than Opus and runs 7x faster (4.0 vs 28.5 min total on httpclient), but produces zero usable output on both migrations. Sonnet costs ~5x less than Opus and produces more rules with fewer issues on httpclient (43 rules, 4 precision vs 29 rules, 9 precision on Claude Code).

7. **Rule generation is only ~40% of skill pipeline time (httpclient).** Rule generation (ingest→construct) takes 7-13 min while testing (scaffold→kantra) takes 4-8 min. The remaining time is coverage analysis and reporting. Scribe skips testing entirely (~2-3 min total), but this means broken rules like Scribe/Sonnet's `class-004` (wrong FQN, never fires) ship without detection.

8. **Model capability has four tiers for agentic pipelines.** Haiku/DeepSeek V3.2 = non-functional or near-empty output. Gemini Pro = functional but weak documentation (3.5-4.5 quality, varies by runtime). Sonnet = strong across all dimensions (5.9-6.0 quality). Opus = comparable to Sonnet but more precision issues from unqualified METHOD_CALL patterns. The jump from tier 1→2 is the "capability cliff"; the difference between tiers 3→4 is nuanced. Gemini Pro shows a unique failure mode: extracting patterns from code examples rather than migration instructions, producing rules that detect standard Java APIs (InputStream, Future, TimeUnit) unrelated to the migration.

## spring-boot3-to-spring-boot4

**Note:** The spring-boot table uses a single "Time (min)" column (total wall-clock) vs. httpclient's "Rule Gen" + "Total" breakdown. Rule gen / test split data was not captured for these runs. "Adj. Gaps" is also omitted — no spring-boot run uses a catch-all package rule.

| Runtime | Model | Rules | Pass Rate | Quality Avg | Overlaps | Time (min) | Precision | Coherence | Cross-Rule | Gaps |
|---------|-------|-------|-----------|-------------|----------|------------|-----------|-----------|------------|------|
| claude-code | sonnet | 89 | 85/89 | 5.60 | 28 | 35.1 | 3 | 7 | 2 | 8 |
| claude-code | opus | 74 | 73/74 | 5.43 | 17 | 26.7 | 5 | 4 | 3 | 10 |
| claude-code | haiku | 0 (failed) | 0/0 | — | 0 | 5.0 | — | — | — | — |
| opencode | sonnet | 95 | 92/95 | 5.52 | 31 | 39.4 | 8 | 5 | 3 | 8 |
| opencode | opus | 91 | 83/91 | 5.54 | 17 | 46.5 | 7 | 4 | 2 | 8 |
| opencode | gemini-pro | 33 | 32/33 | 5.03 | 2 | 31.7 | 4 | 5 | 2 | 18 |
| goose | sonnet | 83 | 82/83 | 5.48 | 17 | 62.3 | 8 | 4 | 3 | 8 |
| goose | opus | 85 | 82/85 | 5.48 | 19 | 54.4 | 6 | 3 | 4 | 3 |
| goose | gemini-pro | 81 | 11/81 | 5.20 | 25 | 16.7 | 5 | 4 | 2 | 5 |
| opencode | deepseek-v3 | 31 | 0/0 | 4.35 | 18 | 53.4 | 3 | 2 | 1 | 12 |
| scribe | opus | 51 | n/a | 5.75 | 3 | — | 2 | 1 | 1 | 5 |

### Eval Details (spring-boot3→4)

#### Claude Code / Sonnet — 89 rules, 85/89 pass

- **73 of 89 rules passed** eval judge review
- **3 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `org.springframework.lang*` too broad for nullability migration, CycloneDX rule fires regardless of version
- **7 coherence issues**: 5 rules with **inverted detection logic** (`00260`, `00420`, `00430`, `00450`, `00730`) — detect config properties that only exist in projects already managing the setting, producing zero incidents on SB3 codebases. `00820` fires on all `@SpringBootTest` but gives MockMVC-specific advice. `00890` duplicates `00270` (AOP starter rename)
- **2 cross-rule overlaps**: `00270`+`00890` (AOP starter duplicate), `00100`+`00110`+`00120` (security test triple-fire)
- **8 gaps**: Optional dependencies in Maven, WebClient/TestRestTemplate with @SpringBootTest, SAML starter rename, BootstrapRegistryInitializer/ConfigurableBootstrapContext moves, BigDecimal representation config, modular starter rules

#### Claude Code / Opus — 74 rules, 73/74 pass

- **56 of 74 rules passed** eval judge review
- **5 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `webClientEnabled|webDriverEnabled` via filecontent too broad, MockMvc detection fires on users already using `@AutoConfigureMockMvc`, CycloneDX matches comments, `launchScript` filecontent minor risk
- **4 coherence issues**: **00010** conflates system requirements (Java 17, Jakarta EE 11), modular design, and classic starters into one overloaded rule. `00500` assumes WAR deployment for all `spring-boot-starter-tomcat` users. `00120` vague on test starter replacements. `00580` broad MongoDB detection for narrow UUID/BigDecimal issue
- **3 cross-rule issues**: `00440`+`00730` duplicate (both detect `/fonts/**` static resource change), `00290`+`00420` overlap on Spring Authorization Server, `00160`+`00010` implicit ordering conflict
- **10 gaps**: `@AutoConfigureTestRestTemplate` requirement (high), actuator `@Nullable` migration (high), Jackson annotations exception, classic starters quick path, logback charset, Jackson auto-module-registration, SAML starter rename, `@AutoConfigureRestTestClient`, package organization changes, `jackson-2-defaults` compat property

#### Claude Code / Haiku — 0 rules (pipeline failed)

- Pipeline crashed at construct stage with `read_patterns_failed` and `construct_failed` errors
- Extracted 83 patterns from the guide (showing comprehension) but produced invalid regex (`*` — bare repetition operator)
- Could not read its own pattern JSON files back
- Zero rules, zero eval possible

#### OpenCode / Sonnet — 95 rules, 92/95 pass

- **73 of 95 rules passed** eval judge review
- **8 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `spring.datasource.` too broad, `org.springframework.graphql*` fires on all GraphQL imports, CycloneDX matches comments, `@SpringBootTest` fires on all tests not just WebClient/TestRestTemplate users, `jackson.find-and-add-modules` inverted, logback broad, properties-migrator inverted
- **5 coherence issues**: **00530** (fail) detects wrong FQN — uses SB4 package `org.springframework.boot.http.converter.autoconfigure.HttpMessageConverters` instead of SB3 package. **00700** detects wrong Spring AMQP class instead of Spring Boot autoconfigure class. **00730** fires on `@AutoConfigureMockMvc` users who are already correct. **00920/00930/00940/00950** are noise rules that fire on unchanged MongoDB properties saying "no change required"
- **3 cross-rule issues**: `00620`+`00870-00910` MongoDB property rename duplication (broad regex + 5 individual rules), `00290`+`00830` AOP starter overlap, `00010`+`00140` system requirements vs starter rename overlap
- **8 gaps**: Optional dependencies in Maven uber jars, MongoDB SSL properties, individual MongoDB connection properties, actuator `@Nullable` parameter context, Jackson generator properties, package organization class relocations, `BootstrapRegistryInitializer` move, MongoDB gridfs.database

#### OpenCode / Opus — 91 rules, 83/91 pass

- **78 of 91 rules passed** eval judge review
- **7 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `alwaysApplyingWhenNonNull` unqualified METHOD_CALL, CycloneDX version-blind, GraphQL package slightly broad, MongoDB property substring risk, Jackson 2 compat vague, `PathRequest` too broad
- **4 coherence issues**: **00450** (fail) detects wrong FQN — uses SB4 package for `HttpMessageConverters` instead of SB3. **00880** fires on any `PropertyMapper` import for narrow null-handling change. **00010** umbrella rule. **00870** doesn't account for `spring-boot-starter-webflux` users
- **2 cross-rule issues**: `00230`+`00880` overlapping PropertyMapper guidance, deprecated starter rules use wrong category
- **8 gaps**: Flyway starter requirement (high), Liquibase starter requirement (high), `@AutoConfigureMockMvc` attribute changes, `HttpMessageConverter` customizer, logback charset, `@AutoConfigureWebTestClient`, `spring-boot-starter-classic`, `spring.jackson.generator.*` properties

#### OpenCode / Gemini Pro — 33 rules, 32/33 pass

- **24 of 33 rules passed** eval judge review
- **4 precision issues**: `PropertyMapper.from()` too broad (00090), `spring-boot-maven-plugin` matches all projects (00100), `@SpringBootTest` matches all tests not just MockMVC users (00270), CycloneDX version-agnostic (00140)
- **5 coherence issues**: Rule **00330** is broken duplicate of 00080 — fires on any `PropertyMapper` method call (fail). Rules 00030/00040 have vague JSpecify messages. Rule 00020 misleads about Spock being re-addable. Rule **00160** names hallucinated class `JsonValueDeserializer` (guide says `JsonObjectDeserializer`)
- **2 cross-rule issues**: Rules 00080+00330 are duplicates (fail); rules 00080+00090 overlap on PropertyMapper (warn)
- **18 gaps**: Missing all deprecated starter renames (6 starters), config property renames (spring.session.redis, spring.data.mongodb, spring.dao, spring.kafka, spring.jackson — 8 properties), build dependency changes (hibernate-jpamodelgen, spring-boot-starter-batch, AOP starter, Tomcat WAR — 4 entries), Jackson 2→3 package migration
- **Second-fewest rules of any spring-boot run** (33 — only DeepSeek has fewer at 31). Covers Java code-level changes (class renames, annotation changes, removed methods) but misses build file migrations and config property renames entirely

#### Goose / Sonnet — 83 rules, 82/83 pass

- **70 of 82 rules passed** eval judge review
- **8 precision issues**: `alwaysApplyingWhenNonNull` unqualified METHOD_CALL, `PropertyMapper` import too broad, `@Nullable` fires on all non-actuator code, `MockMvc` import too broad. **00820** and **00830** fire on *new* Elasticsearch API imports (`ElasticsearchClient`, `ReactiveElasticsearchClient`) instead of old ones — inverted detection
- **4 coherence issues**: **00010** umbrella rule conflates Java 17, Kotlin 2.2, GraalVM 25, Jakarta EE 11, Servlet 6.1, Spring 7.x into one rule. **00500** tells all `spring-boot-starter-tomcat` users to switch to `tomcat-runtime` but only applies to WAR deployments. **00490** Jersey Jackson 3 fires on non-JSON users. **00440** `PathRequest` too broad
- **3 cross-rule issues**: `00270`+`00790`+`00800` triple-fire on AOP starter rename (dependency + `@Timed` + `@Counted`), `00290`+`00300` Spring Authorization Server overlap, `00820`+`00830`+`00510` Elasticsearch triple-fire with wrong trigger on 2 of 3
- **8 gaps**: logback charset, JSpecify `@NonNull`/`@NonNullApi`, Jackson 2 compat module, `spring.jackson.generator.*` properties, `@MockBean` in `@Configuration`, package reorganization beyond GraphQL, `spring-boot-starter-classic` path, Jackson serialization/deserialization properties

#### Goose / Opus — 85 rules, 82/85 pass

- **72 of 85 rules passed** eval judge review
- **6 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations` (00320), `org.springframework.lang` PACKAGE too broad for JSpecify migration (00180), MockMvc import fires on `@WebMvcTest` tests that are unaffected (00640), `webClientEnabled|webDriverEnabled` filecontent matches variable names (00650), `launchScript` in Gradle too generic (00050), CycloneDX version-agnostic (00310)
- **3 coherence issues**: **00390** fires on ALL `@Nullable` usage but gives actuator-endpoint-specific advice — developers using `@Nullable` in non-actuator code get irrelevant guidance. **00790** fires on `MappingJackson2HttpMessageConverter` import but discusses `HttpMessageConverters` deprecation. **00630** detects direct `MockitoTestExecutionListener` import but the issue is indirect — affected users have `@Mock`/`@Captor` fields that silently stop working
- **4 cross-rule issues**: `00290`+`00760` exact duplicates for classic uber-jar loader in Maven, `00300`+`00770` near-duplicates for Gradle, `00590`+`00850` near-duplicates for Kafka StreamsBuilderFactoryBeanCustomizer (one has typo variant), `00180`+`00390` overlapping `@Nullable` detection scope
- **3 gaps**: `@SpringBootTest` no longer provides `TestRestTemplate` beans (high — need `@AutoConfigureTestRestTemplate`), Logback charset default change (low), `spring.jackson.use-jackson2-defaults` property (low)

#### Goose / Gemini Pro — 81 rules, 11/81 pass

- **Lowest pass rate of any spring-boot run** (14% pass, 86% failure). Most failures from `builtin.filecontent` property rules (16 MongoDB + config rules) and `java.dependency` rules that fail kantra test scaffolding ("unable to get build tool"). Only `java.referenced` rules and a few `builtin.filecontent` rules pass.
- **5 precision issues**: `general-annotation-00010` (`@Nullable`) fires on all Spring projects (warn), `general-import-00030` (`com.fasterxml.jackson*`) too broad (warn), `dependencies-dependency-00010` fires on ANY Spring Boot project (warn), `config-pattern-00010` broad regex matches any property (fail), `testing-annotation-00010` (`@SpringBootTest`) fires on all tests not just MockMVC users (warn)
- **4 coherence issues**: **general-import-00070** (fail) detects SB4 FQN (`org.springframework.boot.http.converter.autoconfigure.HttpMessageConverters`) instead of SB3 — will never fire on SB3 code. Same wrong-FQN bug as OpenCode/Sonnet and OpenCode/Opus. **general-annotation-00020** (fail) detects `javax.annotations.NonNull` — wrong package name. **general-annotation-00010** (warn) fires on all `@Nullable` but gives actuator-specific advice. **general-import-00050** (warn) `JsonValueDeserializer` — exists in guide but verify FQN
- **2 cross-rule issues**: Maven/Gradle classic loader duplicates (00030+00040), three OAuth2 starter rename rules with near-identical messages (00130+00140+00150)
- **5 gaps**: Package reorganization rules (WebMvcAutoConfiguration, ErrorMvcAutoConfiguration), `spring.jackson.use-jackson2-defaults` removal, logback charset default change, `spring-boot-starter-classic` quick migration path
- **Unique strength**: 16 individual MongoDB property rename rules — most comprehensive MongoDB coverage of any run. Good coverage of Jackson 2→3 migration (annotations, builders, serializers). 81 rules is near the top for spring-boot runs.
- **Quality note**: Quality avg 5.20 is good — all rules have links (20/45 had no links in the httpclient run). The high failure rate is a test scaffolding issue, not a rule quality issue — many rules are structurally correct but can't be validated by kantra.

#### OpenCode / DeepSeek V3.2 — 31 rules, 0/0 tested

- **Tests never ran** — pipeline scaffold stage completed but kantra tests were not executed. Pass rate is 0/0, not 0/31.
- **3 precision issues**: `core-dependency-00010` fires on ALL Spring Boot projects (too broad), `core-annotation-00010` fires on all `@Nullable` usage not just actuator (warn), `web-change-00010` detects `org.apache.tomcat.util.modeler.Registry` which is Tomcat-internal, not typically imported by applications (warn)
- **2 coherence issues**: **core-change-00010** (fail) — pattern is free text `"Spring Boot package structure changed"` instead of an FQN — will never fire on any code. **testing-annotation-00010/00020** combine `@MockBean` and `@SpyBean` guidance into identical messages — correct but duplicative
- **1 cross-rule issue**: `security-dependency-00010` + `00020` + `00030` OAuth2 starter renames with identical message structure (warn — acceptable for separate dependency rules)
- **12 gaps**: Missing ALL Jackson 2→3 migration (annotations, serializers, builders, properties — 6 rules), ALL MongoDB property renames (12+ properties), ALL session property renames, config property renames (`spring.dao`, `spring.kafka`), `@AutoConfigureMockMvc` changes, `HttpMessageConverters` deprecation, package reorganization class relocations
- **Fewest rules of any spring-boot run** (31 vs 33-95 for others). Covers dependency changes and annotation removals but misses config properties and Jackson migration entirely. Quality 4.35 from 20/31 missing before/after code. Slowest spring-boot run at 53.4 min.
- **Model tier confirmation**: DeepSeek V3.2 on spring-boot produces results consistent with httpclient — follows pipeline structure and generates valid rules, but cannot extract the long tail of migration patterns. Tier 2 (functional but weak).

#### Scribe / Opus — 51 rules, not kantra-tested

- **51 rules across 6 file types**: Java (21), properties (10), Maven XML (11), Gradle (4), YAML (3), spring.factories (2). Multi-format coverage is a Scribe strength — no other run covers spring.factories or has dedicated Gradle detection rules.
- **2 precision issues**: `@Nullable` detection (`org.springframework.lang.Nullable` at ANNOTATION) fires on all Spring null-safety usage, not just migration-requiring cases (warn). Spring Retry wildcard import (`org.springframework.retry.{*}`) fires on projects intentionally keeping Spring Retry (warn — correctly marked optional).
- **1 coherence issue**: PropertyMapper `alwaysApplyingWhenNonNull` (rule 012) before/after section tells users to "remove the call" without showing the equivalent replacement pattern (warn).
- **1 cross-rule issue**: Gradle/YAML parity gap — 4 Gradle rules vs 11 Maven XML rules, 3 YAML rules vs 10 properties rules. Same migration concepts covered inconsistently across build tool formats (warn).
- **5 gaps**: Missing Gradle equivalents for 7 Maven starter renames (aop, web-services, hibernate, elasticsearch, jackson groupId, hibernate-processor, elasticsearch-rest-client). Missing YAML equivalents for 7 property renames (session.redis, session.mongodb, data.mongodb, dao, kafka, mongo health, mongo metrics). Missing autoconfigure class relocation detection. Missing starter-data-jdbc/jpa changes. Missing GraalVM native image changes.
- **Quality 5.75**: All 51 rules have structured `## Title` / `### Before` / `### After` / `### Additional Info` messages. All have links to specific migration guide sections, effort scores, and proper categories. Every `java.referenced` pattern uses fully qualified names. Zero METHOD_CALL qualification issues.
- **Unique strengths**: Only run with spring.factories detection rules (BootstrapRegistryInitializer, EnvironmentPostProcessor). Only run with dedicated Gradle build file rules. Fewest precision issues (2) and coherence issues (1) of any spring-boot run. All METHOD_CALL patterns use FQN (rule 012: `org.springframework.boot.context.properties.PropertyMapper.alwaysApplyingWhenNonNull`).
- **Note**: Scribe is an MCP server (different pipeline architecture). Rules were NOT validated with kantra tests. Pass rate is n/a.

### Key Findings (spring-boot3→4)

**Claude Code / Sonnet** generates the most rules of any Claude Code run (89) with fewest precision issues (3) but the most coherence issues (7), driven by inverted-logic config property detection.

**Claude Code / Opus** generates 74 rules with 4 coherence issues and the most gaps (10). Avoids the inverted-logic trap but conflates three migration actions into one overloaded rule.

**OpenCode / Sonnet** produces the most rules overall (95) but introduces noise rules (`00920-00950`) firing on unchanged MongoDB properties, and a fail-severity coherence bug where rule `00530` detects the *new* SB4 FQN instead of the old SB3 one.

**Goose / Sonnet** generates 83 rules — between Claude Code and OpenCode. Introduces a unique failure mode: 2 Elasticsearch rules (`00820`, `00830`) fire on the *new* API imports instead of the old ones. Shares the umbrella rule and WAR deployment coherence issues with other runs.

**Goose / Opus** generates 85 rules with fewest gaps (3) and fewest coherence issues (3) among Sonnet/Opus skill runs. Produces 4 duplicate/near-duplicate rule pairs that should be deduplicated. The `@Nullable`/actuator coherence mismatch (00390) is the most impactful finding. Takes 54.4 min — slower than Claude Code / Opus (26.7 min) but faster than OpenCode / Opus (46.5 min).

**Goose / Gemini Pro** generates 81 rules but only 11 pass kantra (14% — lowest of any spring-boot run). The failures are mostly test scaffolding issues with `builtin.filecontent` and `java.dependency` rules, not rule correctness problems. Quality 5.20 is reasonable. Same wrong-FQN bug (HttpMessageConverters) as OpenCode/Sonnet and Opus. Best MongoDB property coverage (16 individual rename rules).

**OpenCode / DeepSeek V3.2** generates the fewest rules (31) with the most gaps (12). Tests never ran. Missing all Jackson 2→3, MongoDB property, and config property migrations. Confirms tier 2 (functional but weak) classification from httpclient benchmarks.

**Haiku** failed to complete the pipeline on both migrations — confirming the hard capability cliff.

**Sonnet across runtimes** (spring-boot3→4):

| Runtime | Rules | Pass Rate | Precision | Coherence | Gaps |
|---------|-------|-----------|-----------|-----------|------|
| Claude Code | 89 | 85/89 | 3 | 7 | 8 |
| OpenCode | 95 | 92/95 | 8 | 5 | 8 |
| Goose | 83 | 82/83 | 8 | 4 | 8 |

**Opus across runtimes** (spring-boot3→4):

| Runtime | Rules | Pass Rate | Precision | Coherence | Gaps |
|---------|-------|-----------|-----------|-----------|------|
| Claude Code | 74 | 73/74 | 5 | 4 | 10 |
| OpenCode | 91 | 83/91 | 7 | 4 | 8 |
| Goose | 85 | 82/85 | 6 | 3 | 3 |

Goose/Opus achieves the fewest gaps (3) and fewest coherence issues (3) among Opus runs. Claude Code/Opus has the fewest precision issues (5) but the most gaps (10). OpenCode/Opus extracts the most rules but also has the most failures (8 of 91 didn't pass).

**Gemini Pro across runtimes** (spring-boot3→4):

| Runtime | Rules | Pass Rate | Quality | Precision | Coherence | Gaps |
|---------|-------|-----------|---------|-----------|-----------|------|
| OpenCode | 33 | 32/33 | 5.03 | 4 | 5 | 18 |
| Goose | 81 | 11/81 | 5.20 | 5 | 4 | 5 |

Goose/Gemini produces 2.5x more rules than OpenCode/Gemini (81 vs 33) with far fewer gaps (5 vs 18), but has the lowest pass rate of any run (14%). OpenCode/Gemini has better pass rate (97%) but the fewest rules and most gaps. The high Goose failure rate is largely a test scaffolding issue — `builtin.filecontent` and `java.dependency` rules fail kantra validation at higher rates.

The same model produces meaningfully different results across runtimes. Rules per minute: Claude Code/Opus 2.8, Claude Code/Sonnet 2.5, OpenCode/Sonnet 2.4, OpenCode/Opus 2.0, Goose/Opus 1.6, Goose/Sonnet 1.3. Precision issues: Claude Code/Sonnet 3 (best), Goose/Opus 6, Claude Code/Opus 5, OpenCode/Opus 7, OpenCode/Sonnet 8, Goose/Sonnet 8. The runtime's prompt routing, tool orchestration, and context management affect output quality — not just the model.

**Scribe / Opus** generates 51 rules across 6 file types (Java, properties, Maven XML, Gradle, YAML, spring.factories) — the broadest format coverage of any spring-boot run. Fewest precision issues (2) and coherence issues (1) of any run. All METHOD_CALL patterns use FQN. No kantra testing. Only run with spring.factories and dedicated Gradle detection rules.

**Scribe comparison** (spring-boot3→4):

| Metric | CC/Sonnet (skill) | Scribe/Opus (MCP) |
|--------|-------------------|-------------------|
| Rules | 89 | 51 |
| Gaps | 8 | 5 |
| Total time | 35.1 min | — |
| Precision | 3 | 2 |
| Coherence | 7 | 1 |
| Kantra validated | 85/89 | none |
| File types | 5 (Java, XML, properties, YAML, Gradle) | 6 (Java, properties, XML, Gradle, YAML, spring.factories) |

CC/Sonnet (skill) generates 74% more rules with kantra validation but has 7x the coherence issues (7 vs 1), driven by inverted-logic config property detection. Both cover similar file types (5 vs 6) — the key difference is Scribe's spring.factories detection, which no skill-based run covers. Neither approach dominates: the skill pipeline has volume and test validation; the MCP pipeline has per-rule quality and fewer coherence traps.

**Cross-migration pattern**: All runtimes share the Jackson `com.fasterxml.jackson*` precision issue — a common false-positive trap when a migration renames most but not all packages under a namespace. Config-heavy migrations (spring-boot) produce lower quality scores than API-focused migrations (httpclient) because config property renames yield shallower guidance.

## How to Reproduce

### Prerequisites

- Go 1.25+
- `jq` command-line JSON processor
- One or more agent runtimes installed: Claude Code, OpenCode, Goose

### Running a Benchmark

```bash
./scripts/benchmark-collect.sh <runtime> <model> <migration>
```

The script will:
1. Invoke the agent runtime with the specified model
2. Run the full `/generate-rules` pipeline
3. Time the entire run
4. Run full eval (deterministic + LLM judge)
5. Collect rules, eval snapshot, and metrics into `benchmarks/`
6. Regenerate the comparison table

### Examples

```bash
# Claude Code with Sonnet
./scripts/benchmark-collect.sh claude-code sonnet httpclient4-to-httpclient5

# Claude Code with Haiku
./scripts/benchmark-collect.sh claude-code haiku httpclient4-to-httpclient5

# OpenCode with Gemini Pro
GOOGLE_API_KEY=... ./scripts/benchmark-collect.sh opencode gemini-pro spring-boot3-to-spring-boot4
```

### Runtime Setup

#### Claude Code
- Models: `sonnet` → claude-sonnet-4-6, `opus` → claude-opus-4-6, `haiku` → claude-haiku-4-5-20251001
- No extra env vars needed (uses your Claude Code auth)

#### OpenCode
- Set `ANTHROPIC_API_KEY` for sonnet/opus
- Set `GOOGLE_API_KEY` for gemini-pro

#### Goose
- Set API keys same as OpenCode, or use `goose configure`

### Migration Guide URLs

| Migration | Guide URL |
|-----------|-----------|
| httpclient4-to-httpclient5 | https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide |
| spring-boot3-to-spring-boot4 | https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide |
