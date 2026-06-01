# Benchmark Results — Run 1

Comparison of rule generation quality across agent runtimes and LLM models.

## Methodology

- **Pipeline**: `/generate-rules` skill invoked with the same migration guide URL
- **Evaluation**: Deterministic eval (`cmd/eval`) + LLM judge (`agents/eval/SKILL.md`)
- **Quality Score**: Completeness metric (max 6 pts: message presence + links + effort + before/after guidance). Measures documentation completeness, not rule correctness.
- **Overlaps**: Count of rule pairs that fire on the same code. Overlaps can indicate specificity layering (a broad package-level rule + a specific method-level rule covering the same API), which improves developer experience. High overlap count is not inherently bad — it may mean better coverage through layered detection.
- **Eval Judge**: LLM-based review checking precision (false positive risk), coherence (detection/guidance alignment), cross-rule conflicts, and coverage gaps vs. the migration guide.
- **Timing**: Wall-clock from pipeline start to report completion
- **Date**: May–June 2026

## Runtime × Model Matrix

| Runtime | Model | Model ID | Provider |
|---------|-------|----------|----------|
| Claude Code | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Claude Code | Opus | claude-opus-4-6 | Anthropic |
| Claude Code | Haiku | claude-haiku-4-5-20251001 | Anthropic |
| OpenCode | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| OpenCode | Opus | claude-opus-4-6 | Anthropic |
| OpenCode | Gemini Pro | google-vertex/gemini-3.1-pro-preview | Google |
| Goose | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Goose | Opus | claude-opus-4-6 | Anthropic |
| Goose | Gemini Pro | google-vertex/gemini-3.1-pro-preview | Google |
| Scribe (MCP) | Sonnet | claude-sonnet-4-20250514 | Anthropic |
| Scribe (MCP) | Opus | claude-opus-4-6 | Anthropic |

## httpclient4-to-httpclient5

| Runtime | Model | Rules | Pass Rate | Quality Avg | Overlaps | Time (min) | Precision | Coherence | Cross-Rule | Gaps |
|---------|-------|-------|-----------|-------------|----------|------------|-----------|-----------|------------|------|
| claude-code | sonnet | 43 | 42/43 | 5.95 | 3 | 20.7 | 4 | 4 (1 fail) | 3 | 1 |
| claude-code | opus | 29 | 29/29 | 5.93 | 15 | 18.2 | 9 | 6 (2 fail) | 2 | 3 |
| claude-code | haiku | 26 | 26/26 | 3.0 | 14 | 4.0 | 3 | 26 (all fail) | 5 | 9 |
| opencode | sonnet | 35 | 35/35 | 5.91 | 20 | 49.2 | 12 | 5 | 3 | 9 |
| opencode | opus | 29 | 29/29 | 5.90 | 17 | 21.5 | 8 | 3 | 2 | 8 |
| opencode | gemini-pro | 33 | 33/33 | 4.45 | — | 11.1 | 1 | 4 | 1 | 5 |
| goose | sonnet | 28 | 28/28 | 6.0 | 11 | 21.2 | 7 | 3 | 3 | 8 |
| goose | opus | — | — | — | — | — | — | — | — | — |
| goose | gemini-pro | — | — | — | — | — | — | — | — | — |
| **scribe** | **sonnet** | **14** | **n/a** | **6.0** | **1** | **n/a** | **2** | **3** | **1** | **29** |
| **scribe** | **opus** | **30** | **n/a** | **5.7** | **2** | **n/a** | **2** | **2** | **3** | **13** |

### Eval Details (httpclient4→5)

#### Claude Code / Sonnet — 43 rules, 42/43 pass

- **31 of 43 rules passed** eval judge review
- **4 precision issues** (all `warn`): unqualified METHOD_CALL rules (`00050`, `00060`, `00070`, `00080`) match 5.x replacement APIs — duplicates of more qualified rules (`00200`, `00210`, `00180`)
- **4 coherence issues** (1 `fail`, 3 `warn`): **00350** fires on classic `CloseableHttpClient` import but tells all users to switch to async client — wrong for the majority following the classic migration path. `00230` fires on any `URIUtils` import but only advises about `normalizeSyntax()`. `00340` fires on `PoolingHttpClientConnectionManager` but only covers async replacement. `00420` has implementation note in message referencing internal rule mechanics
- **3 cross-rule duplicates**: `00050`+`00200`, `00060`+`00210`, `00080`+`00180` — unqualified rules should be removed
- **1 gap**: no rule for `HttpContext.getAttribute(HTTP_TARGET_HOST)` → `HttpClientContext.getHttpRoute().getTargetHost()`

#### Claude Code / Opus — 29 rules, 29/29 pass

- **9 precision issues** (all `warn`): every unqualified METHOD_CALL rule (`00050`–`00220`) — worst are `setConnectTimeout` and `setSocketTimeout` which are common across many Java frameworks
- **6 coherence issues** (2 `fail`, 4 `warn`): **00280** and **00290** fire on ALL HC4 `CloseableHttpClient`/`HttpClients` imports and tell developers to migrate to async classes — actively wrong for the majority following the classic-first migration path
- **2 cross-rule issues**: `00130` duplicates `00010` with wrong package namespace; `00280`+`00290` contradict classic migration guidance in `00010`
- **3 gaps**: no rule for `client.start()` async lifecycle requirement (high-severity silent runtime failure); no classic-path IMPORT rules for `CloseableHttpClient` and `HttpClients`

#### Claude Code / Haiku — 26 rules, 0/26 pass eval judge

- **All 26 rules fail**: every message is empty (`': '`), all use `builtin.filecontent` instead of `java.referenced`
- **3 precision issues**: 2 rules target the new 5.x API (`ClassicRequestBuilder`), 1 matches JDK `SSLContext` (not HttpClient-specific)
- **5 duplicate groups** (11 rules): `00010`+`00090`, `00040`+`00050`, `00060`+`00070`, `00140`+`00200`, `00100`+`00210`+`00230`+`00240`+`00260`
- **Massive coverage gap**: only ~14 unique old APIs detected out of 380 ground truth entries (3.7% coverage)

#### OpenCode / Opus — 29 rules, 29/29 pass

- **20 of 28 rules passed** eval judge review
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



- **22 of 35 rules passed** eval judge review
- **12 precision issues**: every unqualified METHOD_CALL rule — common method names like `setConnectTimeout`, `setSoTimeout`, `getAllHeaders`, `getRequestLine` match across many Java frameworks. This is the worst precision score across all runs.
- **5 coherence issues**: **00280** and **00290** push classic `CloseableHttpClient`/`HttpClients` users to async — wrong for classic migration path. **00300** and **00310** jump to async `SimpleRequestBuilder` for `HttpPost`/`HttpGet` instead of classic 5.x replacements. **00010**+**00120** duplicate package rules with wrong namespace (`org.apache.hc.httpclient5` instead of `org.apache.hc.client5`)
- **3 cross-rule issues**: `00010`+`00120` duplicate detection, `00280`+`00290` contradictory async push (fail), `00270`+`00320` overlapping async connection manager guidance
- **9 gaps**: `client.execute()` return type change, `SSLConnectionSocketFactory` removal, `StatusLine` removal, `PoolingHttpClientConnectionManager` classic migration, `HttpRequestRetryHandler` removal — 5 high-impact patterns missing

#### Goose / Sonnet — 28 rules, 28/28 pass

- **14 of 24 rules passed** eval judge review (excludes ruleset + dependency rules)
- **7 precision issues**: unqualified METHOD_CALL rules — `getAllHeaders` (high false-positive risk), `setConnectTimeout` (high), `setSocketTimeout` (high), `closeExpiredConnections`, `closeIdleConnections`, `addInterceptorLast`, `setRetryHandler`
- **3 coherence issues**: **00260** fires on `CloseableHttpClient` import but gives async-only guidance. **00270** same for `HttpClients`. **00250** same for `PoolingHttpClientConnectionManager` — all three wrong for classic migration path
- **3 cross-rule issues**: `00010`+`00260`+`00270` contradictory guidance (package rule vs async rules), `00010` overlaps with all IMPORT rules, `00280`+`00260` both push async
- **8 gaps**: `HttpResponse` interface split (high), `ResponseHandler` pattern (high), `CookieSpecs`, `TimeUnit`→`Timeout`/`TimeValue`, `HttpGet/Put/Delete/Patch` constructors (only `HttpPost` covered), `EntityUtils`, `BasicNameValuePair`, async streaming consumers

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

**Claude Code / Sonnet** produces the most comprehensive ruleset (43 rules) with highest quality scores and fewest eval judge issues. One rule fails kantra tests, but overall coverage and guidance are strong.

**Claude Code / Opus** generates fewer rules (29) but is more conservative. Higher overlap count reflects specificity layering. More precision issues from unqualified METHOD_CALL conditions.

**Claude Code / Haiku** produces a non-functional ruleset: all 26 rules have empty messages, use wrong condition types, and include 5 duplicate groups.

**OpenCode / Opus** generates exactly the same rule count as Claude Code / Opus (29) with identical pass rate. 8 unqualified METHOD_CALL precision issues and the same async/classic coherence pattern. 8 gaps — notably only covers `HttpPost` constructor migration, missing `HttpGet/Put/Delete/Patch`.

**OpenCode / Sonnet** generates 35 rules (fewer than Claude Code / Sonnet's 43) but has the worst precision of any run — 12 unqualified METHOD_CALL rules. Takes 2.4x longer (49 min vs 21 min) than Claude Code / Sonnet. Same async/classic coherence issue plus a wrong namespace in a duplicate package rule.

**Goose / Sonnet** generates the fewest rules (28) but achieves a perfect 6.0 quality avg — every rule has complete documentation (links, effort, before/after guidance). Fewest overlaps (11) of any run. Same async/classic coherence issue and missing HTTP method constructors as other runs.

**Sonnet across runtimes** (httpclient4→5):

| Runtime | Rules | Quality | Precision | Coherence | Gaps |
|---------|-------|---------|-----------|-----------|------|
| Claude Code | 43 | 5.95 | 4 | 4 | 1 |
| OpenCode | 35 | 5.91 | 12 | 5 | 9 |
| Goose | 28 | **6.0** | 7 | 3 | 8 |

Claude Code extracts significantly more rules (43) with fewest issues. Goose produces the fewest rules but highest documentation quality. OpenCode sits in the middle on rule count but has the worst precision and gaps.

**Shared issue**: All Sonnet and Opus runs generate rules that tell classic `CloseableHttpClient` users to switch to async, contradicting the migration guide's classic-first path. This is a consistent LLM blind spot regardless of runtime.

### Interesting Findings

1. **Model capability has a hard threshold for agentic pipelines.** Haiku can follow instructions and produce syntactically valid YAML, but it cannot reason through the multi-step pipeline (ingest guide → extract patterns → construct rules with proper `java.referenced` conditions → generate test data → iterate on failures). It defaults to the simplest possible rule structure (`builtin.filecontent` with empty messages) and never self-corrects. This suggests agentic coding pipelines need a minimum model capability tier — there is no graceful degradation, just a cliff.

2. **More rules ≠ more noise.** Sonnet generates 48% more rules than Opus (43 vs 29) but has proportionally *fewer* eval judge issues. The additional rules cover simple class relocations and API renames that Opus skips entirely. Sonnet's broader extraction captures the long tail of migration patterns without sacrificing precision.

3. **Unqualified METHOD_CALL is the top precision pitfall.** Both Sonnet (4 rules) and Opus (9 rules) generate rules that match method names like `setConnectTimeout` without qualifying the parent class. These fire on the 5.x replacement APIs too — the very code the rule tells you to write. Opus is worse here because it generates more unqualified rules and fewer qualified alternatives.

4. **The async/classic migration path is a consistent LLM blind spot.** Both Sonnet and Opus produce rules telling `CloseableHttpClient` users to switch to `CloseableHttpAsyncClient`. The migration guide explicitly recommends migrating to the 5.x *classic* API first, then optionally to async. This is the highest-severity issue across both rulesets — it would actively mislead the majority of users following the guide.

5. **Overlaps are a feature, not a bug.** Opus's higher overlap count (15 vs Sonnet's 3) reflects more specificity layering — a broad package-level import rule plus specific method-level rules for the same API. This gives developers both a high-level "this package moved" warning and targeted "change this specific call" guidance. The eval initially flagged these as conflicts, but they represent deliberate detection depth.

6. **Cost-quality tradeoff is stark.** Haiku costs ~20x less per token than Opus and runs 4.5x faster, but produces zero usable output. Sonnet costs ~5x less than Opus, runs slightly slower (20.7 vs 18.2 min), and produces better results across every dimension. For this pipeline, Sonnet is the clear cost-performance winner.

## spring-boot3-to-spring-boot4

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
| goose | gemini-pro | — | — | — | — | — | — | — | — | — |

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
- **Fewest rules of any spring-boot run** (33 vs 74-95 for other runtimes). Covers Java code-level changes (class renames, annotation changes, removed methods) but misses build file migrations and config property renames entirely

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

### Key Findings (spring-boot3→4)

**Claude Code / Sonnet** generates the most rules of any Claude Code run (89) with fewest precision issues (3) but the most coherence issues (7), driven by inverted-logic config property detection.

**Claude Code / Opus** generates 74 rules with fewest coherence issues (4) but the most gaps (10). Avoids the inverted-logic trap but conflates three migration actions into one overloaded rule.

**OpenCode / Sonnet** produces the most rules overall (95) but introduces noise rules (`00920-00950`) firing on unchanged MongoDB properties, and a fail-severity coherence bug where rule `00530` detects the *new* SB4 FQN instead of the old SB3 one.

**Goose / Sonnet** generates 83 rules — between Claude Code and OpenCode. Introduces a unique failure mode: 2 Elasticsearch rules (`00820`, `00830`) fire on the *new* API imports instead of the old ones. Shares the umbrella rule and WAR deployment coherence issues with other runs.

**Goose / Opus** generates the most rules of any spring-boot run (85) with fewest gaps (3) and fewest coherence issues (3). Produces 4 duplicate/near-duplicate rule pairs that should be deduplicated. The `@Nullable`/actuator coherence mismatch (00390) is the most impactful finding. Takes 54.4 min — slower than Claude Code / Opus (26.7 min) but faster than OpenCode / Opus (46.5 min).

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

Goose/Opus achieves the fewest gaps (3) and fewest coherence issues (3) of any spring-boot run. Claude Code/Opus has the fewest precision issues but the most gaps (10). OpenCode/Opus extracts the most rules but also has the most failures (8 of 91 didn't pass).

The same model produces meaningfully different results across runtimes. Claude Code extracts the most rules per runtime-minute and has the fewest precision issues. OpenCode extracts the most total rules but with the most issues. Goose sits in the middle. The runtime's prompt routing, tool orchestration, and context management affect output quality — not just the model.

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
