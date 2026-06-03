# Eval Details — spring-boot3-to-spring-boot4

Per-run eval judge results for each runtime × model combination. See [main benchmark results](../README.md) for summary tables and analysis.

---

## Claude Code / Sonnet — 89 rules, 85/89 pass

- **73 of 89 rules passed** eval judge review
- **3 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `org.springframework.lang*` too broad for nullability migration, CycloneDX rule fires regardless of version
- **7 coherence issues**: 5 rules with **inverted detection logic** (`00260`, `00420`, `00430`, `00450`, `00730`) — detect config properties that only exist in projects already managing the setting, producing zero incidents on SB3 codebases. `00820` fires on all `@SpringBootTest` but gives MockMVC-specific advice. `00890` duplicates `00270` (AOP starter rename)
- **2 cross-rule overlaps**: `00270`+`00890` (AOP starter duplicate), `00100`+`00110`+`00120` (security test triple-fire)
- **8 gaps**: Optional dependencies in Maven, WebClient/TestRestTemplate with @SpringBootTest, SAML starter rename, BootstrapRegistryInitializer/ConfigurableBootstrapContext moves, BigDecimal representation config, modular starter rules

## Claude Code / Opus — 74 rules, 73/74 pass

- **56 of 74 rules passed** eval judge review
- **5 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `webClientEnabled|webDriverEnabled` via filecontent too broad, MockMvc detection fires on users already using `@AutoConfigureMockMvc`, CycloneDX matches comments, `launchScript` filecontent minor risk
- **4 coherence issues**: **00010** conflates system requirements (Java 17, Jakarta EE 11), modular design, and classic starters into one overloaded rule. `00500` assumes WAR deployment for all `spring-boot-starter-tomcat` users. `00120` vague on test starter replacements. `00580` broad MongoDB detection for narrow UUID/BigDecimal issue
- **3 cross-rule issues**: `00440`+`00730` duplicate (both detect `/fonts/**` static resource change), `00290`+`00420` overlap on Spring Authorization Server, `00160`+`00010` implicit ordering conflict
- **10 gaps**: `@AutoConfigureTestRestTemplate` requirement (high), actuator `@Nullable` migration (high), Jackson annotations exception, classic starters quick path, logback charset, Jackson auto-module-registration, SAML starter rename, `@AutoConfigureRestTestClient`, package organization changes, `jackson-2-defaults` compat property

## Claude Code / Haiku — 0 rules (pipeline failed)

- Pipeline crashed at construct stage with `read_patterns_failed` and `construct_failed` errors
- Extracted 83 patterns from the guide (showing comprehension) but produced invalid regex (`*` — bare repetition operator)
- Could not read its own pattern JSON files back
- Zero rules, zero eval possible

## OpenCode / Sonnet — 95 rules, 92/95 pass

- **73 of 95 rules passed** eval judge review
- **8 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `spring.datasource.` too broad, `org.springframework.graphql*` fires on all GraphQL imports, CycloneDX matches comments, `@SpringBootTest` fires on all tests not just WebClient/TestRestTemplate users, `jackson.find-and-add-modules` inverted, logback broad, properties-migrator inverted
- **5 coherence issues**: **00530** (fail) detects wrong FQN — uses SB4 package `org.springframework.boot.http.converter.autoconfigure.HttpMessageConverters` instead of SB3 package. **00700** detects wrong Spring AMQP class instead of Spring Boot autoconfigure class. **00730** fires on `@AutoConfigureMockMvc` users who are already correct. **00920/00930/00940/00950** are noise rules that fire on unchanged MongoDB properties saying "no change required"
- **3 cross-rule issues**: `00620`+`00870-00910` MongoDB property rename duplication (broad regex + 5 individual rules), `00290`+`00830` AOP starter overlap, `00010`+`00140` system requirements vs starter rename overlap
- **8 gaps**: Optional dependencies in Maven uber jars, MongoDB SSL properties, individual MongoDB connection properties, actuator `@Nullable` parameter context, Jackson generator properties, package organization class relocations, `BootstrapRegistryInitializer` move, MongoDB gridfs.database

## OpenCode / Opus — 91 rules, 83/91 pass

- **78 of 91 rules passed** eval judge review
- **7 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations`, `alwaysApplyingWhenNonNull` unqualified METHOD_CALL, CycloneDX version-blind, GraphQL package slightly broad, MongoDB property substring risk, Jackson 2 compat vague, `PathRequest` too broad
- **4 coherence issues**: **00450** (fail) detects wrong FQN — uses SB4 package for `HttpMessageConverters` instead of SB3. **00880** fires on any `PropertyMapper` import for narrow null-handling change. **00010** umbrella rule. **00870** doesn't account for `spring-boot-starter-webflux` users
- **2 cross-rule issues**: `00230`+`00880` overlapping PropertyMapper guidance, deprecated starter rules use wrong category
- **8 gaps**: Flyway starter requirement (high), Liquibase starter requirement (high), `@AutoConfigureMockMvc` attribute changes, `HttpMessageConverter` customizer, logback charset, `@AutoConfigureWebTestClient`, `spring-boot-starter-classic`, `spring.jackson.generator.*` properties

## OpenCode / Gemini Pro — 33 rules, 32/33 pass

- **24 of 33 rules passed** eval judge review
- **4 precision issues**: `PropertyMapper.from()` too broad (00090), `spring-boot-maven-plugin` matches all projects (00100), `@SpringBootTest` matches all tests not just MockMVC users (00270), CycloneDX version-agnostic (00140)
- **5 coherence issues**: Rule **00330** is broken duplicate of 00080 — fires on any `PropertyMapper` method call (fail). Rules 00030/00040 have vague JSpecify messages. Rule 00020 misleads about Spock being re-addable. Rule **00160** names hallucinated class `JsonValueDeserializer` (guide says `JsonObjectDeserializer`)
- **2 cross-rule issues**: Rules 00080+00330 are duplicates (fail); rules 00080+00090 overlap on PropertyMapper (warn)
- **18 gaps**: Missing all deprecated starter renames (6 starters), config property renames (spring.session.redis, spring.data.mongodb, spring.dao, spring.kafka, spring.jackson — 8 properties), build dependency changes (hibernate-jpamodelgen, spring-boot-starter-batch, AOP starter, Tomcat WAR — 4 entries), Jackson 2→3 package migration
- **Second-fewest rules of any spring-boot run** (33 — only DeepSeek has fewer at 31). Covers Java code-level changes (class renames, annotation changes, removed methods) but misses build file migrations and config property renames entirely

## Goose / Sonnet — 83 rules, 82/83 pass

- **70 of 82 rules passed** eval judge review
- **8 precision issues**: `alwaysApplyingWhenNonNull` unqualified METHOD_CALL, `PropertyMapper` import too broad, `@Nullable` fires on all non-actuator code, `MockMvc` import too broad. **00820** and **00830** fire on *new* Elasticsearch API imports (`ElasticsearchClient`, `ReactiveElasticsearchClient`) instead of old ones — inverted detection
- **4 coherence issues**: **00010** umbrella rule conflates Java 17, Kotlin 2.2, GraalVM 25, Jakarta EE 11, Servlet 6.1, Spring 7.x into one rule. **00500** tells all `spring-boot-starter-tomcat` users to switch to `tomcat-runtime` but only applies to WAR deployments. **00490** Jersey Jackson 3 fires on non-JSON users. **00440** `PathRequest` too broad
- **3 cross-rule issues**: `00270`+`00790`+`00800` triple-fire on AOP starter rename (dependency + `@Timed` + `@Counted`), `00290`+`00300` Spring Authorization Server overlap, `00820`+`00830`+`00510` Elasticsearch triple-fire with wrong trigger on 2 of 3
- **8 gaps**: logback charset, JSpecify `@NonNull`/`@NonNullApi`, Jackson 2 compat module, `spring.jackson.generator.*` properties, `@MockBean` in `@Configuration`, package reorganization beyond GraphQL, `spring-boot-starter-classic` path, Jackson serialization/deserialization properties

## Goose / Opus — 85 rules, 82/85 pass

- **72 of 85 rules passed** eval judge review
- **6 precision issues**: `com.fasterxml.jackson*` matches unchanged `jackson-annotations` (00320), `org.springframework.lang` PACKAGE too broad for JSpecify migration (00180), MockMvc import fires on `@WebMvcTest` tests that are unaffected (00640), `webClientEnabled|webDriverEnabled` filecontent matches variable names (00650), `launchScript` in Gradle too generic (00050), CycloneDX version-agnostic (00310)
- **3 coherence issues**: **00390** fires on ALL `@Nullable` usage but gives actuator-endpoint-specific advice — developers using `@Nullable` in non-actuator code get irrelevant guidance. **00790** fires on `MappingJackson2HttpMessageConverter` import but discusses `HttpMessageConverters` deprecation. **00630** detects direct `MockitoTestExecutionListener` import but the issue is indirect — affected users have `@Mock`/`@Captor` fields that silently stop working
- **4 cross-rule issues**: `00290`+`00760` exact duplicates for classic uber-jar loader in Maven, `00300`+`00770` near-duplicates for Gradle, `00590`+`00850` near-duplicates for Kafka StreamsBuilderFactoryBeanCustomizer (one has typo variant), `00180`+`00390` overlapping `@Nullable` detection scope
- **3 gaps**: `@SpringBootTest` no longer provides `TestRestTemplate` beans (high — need `@AutoConfigureTestRestTemplate`), Logback charset default change (low), `spring.jackson.use-jackson2-defaults` property (low)

## Goose / Gemini Pro — 81 rules, 11/81 pass

- **Lowest pass rate of any spring-boot run** (14% pass, 86% failure). Most failures from `builtin.filecontent` property rules (16 MongoDB + config rules) and `java.dependency` rules that fail kantra test scaffolding ("unable to get build tool"). Only `java.referenced` rules and a few `builtin.filecontent` rules pass.
- **5 precision issues**: `general-annotation-00010` (`@Nullable`) fires on all Spring projects (warn), `general-import-00030` (`com.fasterxml.jackson*`) too broad (warn), `dependencies-dependency-00010` fires on ANY Spring Boot project (warn), `config-pattern-00010` broad regex matches any property (fail), `testing-annotation-00010` (`@SpringBootTest`) fires on all tests not just MockMVC users (warn)
- **4 coherence issues**: **general-import-00070** (fail) detects SB4 FQN (`org.springframework.boot.http.converter.autoconfigure.HttpMessageConverters`) instead of SB3 — will never fire on SB3 code. Same wrong-FQN bug as OpenCode/Sonnet and OpenCode/Opus. **general-annotation-00020** (fail) detects `javax.annotations.NonNull` — wrong package name. **general-annotation-00010** (warn) fires on all `@Nullable` but gives actuator-specific advice. **general-import-00050** (warn) `JsonValueDeserializer` — exists in guide but verify FQN
- **2 cross-rule issues**: Maven/Gradle classic loader duplicates (00030+00040), three OAuth2 starter rename rules with near-identical messages (00130+00140+00150)
- **5 gaps**: Package reorganization rules (WebMvcAutoConfiguration, ErrorMvcAutoConfiguration), `spring.jackson.use-jackson2-defaults` removal, logback charset default change, `spring-boot-starter-classic` quick migration path
- **Unique strength**: 16 individual MongoDB property rename rules — most comprehensive MongoDB coverage of any run. Good coverage of Jackson 2→3 migration (annotations, builders, serializers). 81 rules is near the top for spring-boot runs.
- **Quality note**: Quality avg 5.20 is good — all rules have links (20/45 had no links in the httpclient run). The high failure rate is a test scaffolding issue, not a rule quality issue — many rules are structurally correct but can't be validated by kantra.

## OpenCode / DeepSeek V3.2 — 31 rules, 0/0 tested

- **Tests never ran** — pipeline scaffold stage completed but kantra tests were not executed. Pass rate is 0/0, not 0/31.
- **3 precision issues**: `core-dependency-00010` fires on ALL Spring Boot projects (too broad), `core-annotation-00010` fires on all `@Nullable` usage not just actuator (warn), `web-change-00010` detects `org.apache.tomcat.util.modeler.Registry` which is Tomcat-internal, not typically imported by applications (warn)
- **2 coherence issues**: **core-change-00010** (fail) — pattern is free text `"Spring Boot package structure changed"` instead of an FQN — will never fire on any code. **testing-annotation-00010/00020** combine `@MockBean` and `@SpyBean` guidance into identical messages — correct but duplicative
- **1 cross-rule issue**: `security-dependency-00010` + `00020` + `00030` OAuth2 starter renames with identical message structure (warn — acceptable for separate dependency rules)
- **12 gaps**: Missing ALL Jackson 2→3 migration (annotations, serializers, builders, properties — 6 rules), ALL MongoDB property renames (12+ properties), ALL session property renames, config property renames (`spring.dao`, `spring.kafka`), `@AutoConfigureMockMvc` changes, `HttpMessageConverters` deprecation, package reorganization class relocations
- **Fewest rules of any spring-boot run** (31 vs 33-95 for others). Covers dependency changes and annotation removals but misses config properties and Jackson migration entirely. Quality 4.35 from 20/31 missing before/after code. Slowest spring-boot run at 53.4 min.
- **Model tier confirmation**: DeepSeek V3.2 on spring-boot produces results consistent with httpclient — follows pipeline structure and generates valid rules, but cannot extract the long tail of migration patterns. Tier 2 (functional but weak).

## Scribe / Opus — 51 rules, not kantra-tested

- **51 rules across 6 file types**: Java (21), properties (10), Maven XML (11), Gradle (4), YAML (3), spring.factories (2). Multi-format coverage is a Scribe strength — no other run covers spring.factories or has dedicated Gradle detection rules.
- **2 precision issues**: `@Nullable` detection (`org.springframework.lang.Nullable` at ANNOTATION) fires on all Spring null-safety usage, not just migration-requiring cases (warn). Spring Retry wildcard import (`org.springframework.retry.{*}`) fires on projects intentionally keeping Spring Retry (warn — correctly marked optional).
- **1 coherence issue**: PropertyMapper `alwaysApplyingWhenNonNull` (rule 012) before/after section tells users to "remove the call" without showing the equivalent replacement pattern (warn).
- **1 cross-rule issue**: Gradle/YAML parity gap — 4 Gradle rules vs 11 Maven XML rules, 3 YAML rules vs 10 properties rules. Same migration concepts covered inconsistently across build tool formats (warn).
- **5 gaps**: Missing Gradle equivalents for 7 Maven starter renames (aop, web-services, hibernate, elasticsearch, jackson groupId, hibernate-processor, elasticsearch-rest-client). Missing YAML equivalents for 7 property renames (session.redis, session.mongodb, data.mongodb, dao, kafka, mongo health, mongo metrics). Missing autoconfigure class relocation detection. Missing starter-data-jdbc/jpa changes. Missing GraalVM native image changes.
- **Quality 5.75**: All 51 rules have structured `## Title` / `### Before` / `### After` / `### Additional Info` messages. All have links to specific migration guide sections, effort scores, and proper categories. Every `java.referenced` pattern uses fully qualified names. Zero METHOD_CALL qualification issues.
- **Unique strengths**: Only run with spring.factories detection rules (BootstrapRegistryInitializer, EnvironmentPostProcessor). Only run with dedicated Gradle build file rules. Fewest precision issues (2) and coherence issues (1) of any spring-boot run. All METHOD_CALL patterns use FQN (rule 012: `org.springframework.boot.context.properties.PropertyMapper.alwaysApplyingWhenNonNull`).
- **Note**: Scribe is an MCP server (different pipeline architecture). Rules were NOT validated with kantra tests. Pass rate is n/a.
