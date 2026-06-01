## Eval Judge Report -- opencode / gemini-pro / spring-boot3-to-spring-boot4

**Guide:** https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide
**Language:** java
**Tool:** OpenCode with Gemini 3.1 Pro
**Date:** 2026-06-01

### How the rules performed

| Metric | Value |
|--------|-------|
| Total rules | 33 |
| Kantra pass | 32/33 (97%) |
| Quality score | avg 5.03/6 |
| Rules with links | 32/33 |
| Rules with before/after guidance | 23/33 |

**Quality gaps:** 10 rules missing before/after guidance

### Rules that need attention

**24 of 33 rules passed.** The 9 below need fixes:

> **What "detection" and "guidance" mean:**
> - **Detection** = the rule's `when` condition -- does it find the right code?
> - **Guidance** = the rule's `message` -- does it tell the developer the correct fix?
>
> **Issue types:**
> - **Precision** -- detection is too broad (e.g., matches unrelated code). The guidance is still correct when the rule fires. Fix is usually mechanical.
> - **Coherence** -- detection and guidance don't match. The rule fires on one thing but advises about something else. Needs a design rethink.

#### Precision issues (4)

These rules detect the right thing but cast too wide a net. Guidance is correct when they fire.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `00090` | too broad | ok | Fires on every `PropertyMapper.from()` call; most callers are not affected by the null-mapping behavioral change | Accepted tradeoff -- cannot distinguish callers relying on null mapping vs not. Add qualifying note to message: "Only affects calls where source may return null and you relied on the destination being called with null." |
| `00100` | too broad | ok | Fires on every project using `spring-boot-maven-plugin` regardless of whether optional dependencies exist | Accepted tradeoff -- cannot detect `<optional>true</optional>` deps via this condition alone. Message already says "if needed" |
| `00270` | too broad | ok | Fires on every `@SpringBootTest` usage; only tests also using MockMVC are actually affected | Add second condition to narrow: AND with `MockMvc` import or `@AutoConfigureMockMvc`. Alternatively, reclassify as informational |
| `00140` | too broad | ok | Fires on any CycloneDX plugin usage including versions already >= 3.0.0 | Accepted tradeoff -- `builtin.filecontent` cannot filter by version |

#### Coherence issues (5)

These rules have a mismatch between what they detect and what they advise. Developers may see confusing or irrelevant guidance.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `00330` | too broad | wrong scope | Fires on any `PropertyMapper` METHOD_CALL but message only addresses `alwaysApplyingWhenNonNull()` removal -- exact duplicate of rule 00080 with a broader, incorrect condition | **Delete this rule.** Rule 00080 already covers `alwaysApplyingWhenNonNull` with the correct pattern |
| `00030` | ok | vague | Message says "annotations changed with JSpecify addition" without naming the replacement `org.jspecify.annotations.Nullable` | Rewrite message: "Migrate to `org.jspecify.annotations.Nullable`. For actuator endpoint parameters, `org.springframework.lang.Nullable` is no longer supported for declaring optional parameters." |
| `00040` | ok | vague | Message says "annotations changed with JSpecify addition" without naming what to do | Rewrite message: "Spring Boot 4 adds JSpecify nullability annotations. If using Kotlin or a null-checker, this may cause compilation failures. Review the Spring Framework JSpecify migration guide." |
| `00020` | ok | misleading | Message says Spock removed from test starter (implying you can add it back), but guide says Spock integration is entirely removed because Spock does not support Groovy 5 | Rewrite message: "Spring Boot's Spock integration has been removed because Spock does not yet support Groovy 5. You cannot use Spock with Spring Boot 4 until Spock adds Groovy 5 support." |
| `00160` | ok | wrong API name | Description says "JsonValueDeserializer" but the guide renames `JsonObjectDeserializer` -> `ObjectValueDeserializer`. The class `JsonValueDeserializer` does not exist in the guide. The OR condition catches `JsonObjectDeserializer` (correct) but the description/message name the wrong class | Fix description and message to say "JsonObjectDeserializer renamed to ObjectValueDeserializer". Remove the `JsonValueDeserializer` pattern from the OR condition -- it is a hallucinated class name |

### Cross-rule issues (2)

| Rules | Issue | Severity | Fix |
|-------|-------|----------|-----|
| `00080` + `00330` | Both target PropertyMapper alwaysApplyingWhenNonNull() removal. Rule 00330 is a duplicate with a broader, wrong condition that fires on any PropertyMapper method call | fail | Delete rule 00330 |
| `00080` + `00090` | Both cover PropertyMapper changes from different angles. A developer using `PropertyMapper.from()` sees both warnings with potentially confusing overlap | warn | Add cross-reference in 00090's message: "See also the removal of `alwaysApplyingWhenNonNull()`" |

### Missing rules (18 gaps)

These migration patterns from the guide have no corresponding rule. A missing rule means affected code gets no warning at all.

| What the guide says to migrate | Guide section | Impact | Suggested detection |
|-------------------------------|---------------|--------|---------------------|
| `spring-boot-starter-web` -> `spring-boot-starter-webmvc` | Deprecated Starters | high | `builtin.xml` or `builtin.filecontent` matching `spring-boot-starter-web` in pom.xml/build.gradle |
| `spring-boot-starter-oauth2-authorization-server` -> `spring-boot-starter-security-oauth2-authorization-server` | Deprecated Starters | high | `builtin.xml` matching artifactId in pom.xml |
| `spring-boot-starter-oauth2-client` -> `spring-boot-starter-security-oauth2-client` | Deprecated Starters | high | `builtin.xml` matching artifactId in pom.xml |
| `spring-boot-starter-oauth2-resource-server` -> `spring-boot-starter-security-oauth2-resource-server` | Deprecated Starters | high | `builtin.xml` matching artifactId in pom.xml |
| `spring-boot-starter-web-services` -> `spring-boot-starter-webservices` | Deprecated Starters | high | `builtin.xml` matching artifactId in pom.xml |
| `spring-boot-starter-aop` -> `spring-boot-starter-aspectj` | AOP Starter POM | high | `builtin.xml` matching artifactId in pom.xml |
| `spring.session.redis.*` -> `spring.session.data.redis.*` | Spring Session | high | `builtin.filecontent` matching `spring.session.redis` in application.properties/yml |
| `spring.data.mongodb.*` -> `spring.mongodb.*` (13 properties) | MongoDB | high | `builtin.filecontent` matching `spring.data.mongodb.host` etc. in properties/yml |
| `spring.dao.exceptiontranslation.enabled` -> `spring.persistence.exceptiontranslation.enabled` | Persistence Modules | high | `builtin.filecontent` matching `spring.dao.exceptiontranslation` in properties/yml |
| `hibernate-jpamodelgen` -> `hibernate-processor` | Hibernate Dependency Management | high | `builtin.xml` matching artifactId in pom.xml |
| Spring Batch no longer stores metadata in DB by default | Spring Batch | high | `builtin.xml` or `builtin.filecontent` matching `spring-boot-starter-batch` in build files |
| `spring-boot-starter-tomcat` -> `spring-boot-starter-tomcat-runtime` for WAR deployments | Tomcat | high | `builtin.xml` matching artifactId in pom.xml with `<packaging>war</packaging>` |
| `spring.kafka.retry.topic.backoff.random` -> `spring.kafka.retry.topic.backoff.jitter` | Spring Kafka Retry | medium | `builtin.filecontent` matching `spring.kafka.retry.topic.backoff.random` |
| `spring.jackson.read.*` / `spring.jackson.write.*` -> `spring.jackson.json.read.*` / `spring.jackson.json.write.*` | Upgrading Jackson | medium | `builtin.filecontent` matching `spring.jackson.read` or `spring.jackson.write` |
| `spring.session.mongodb.*` -> `spring.session.data.mongodb.*` | Spring Session | medium | `builtin.filecontent` matching `spring.session.mongodb` |
| `management.health.mongo.enabled` -> `management.health.mongodb.enabled` etc. | MongoDB | medium | `builtin.filecontent` matching `management.health.mongo` or `management.metrics.mongo` |
| Jackson 2 `com.fasterxml.jackson` package -> Jackson 3 `tools.jackson` package | Upgrading Jackson | high | `java.referenced` matching `com.fasterxml.jackson` at IMPORT (broad but necessary) |
| `StreamBuilderFactoryBeanCustomizer` pattern check | Kafka Streams | medium | Rule 00240 detects `StreamsBuilderFactoryBeanCustomizer` (with extra 's') -- verify the old class name is correct. The guide says `StreamBuilderFactoryBeanCustomizer` |

### Summary counts

| Category | Count |
|----------|-------|
| **Precision issues** | 4 (all warn) |
| **Coherence issues** | 5 (1 fail, 4 warn) |
| **Cross-rule issues** | 2 (1 fail, 1 warn) |
| **Gaps** | 18 (12 high, 6 medium) |

### Verdict

**24 of 33 rules passed** eval judge review.

- **4 precision issues**: PropertyMapper.from() too broad; spring-boot-maven-plugin matches all projects; @SpringBootTest matches all tests not just MockMVC users; CycloneDX version-agnostic
- **5 coherence issues**: Rule 00330 is a broken duplicate of 00080 (fail); rules 00030/00040 have vague JSpecify messages (warn); rule 00020 misleads about Spock being re-addable (warn); rule 00160 names a hallucinated class JsonValueDeserializer (warn)
- **2 cross-rule issues**: Rules 00080+00330 are duplicates (fail); rules 00080+00090 overlap on PropertyMapper (warn)
- **18 gaps**: Major categories missing: deprecated starter renames (6 starters), configuration property renames (spring.session, spring.data.mongodb, spring.dao, spring.kafka, spring.jackson -- 8 properties), build dependency changes (hibernate-jpamodelgen, spring-boot-starter-batch, spring-boot-starter-aop, Tomcat WAR -- 4 entries), Jackson 2->3 package migration

The ruleset covers Java code-level changes well (class renames, annotation changes, removed methods) but has significant gaps in build file migrations (deprecated starters) and configuration property renames. The 1 fail-severity coherence issue (rule 00330) should be fixed immediately by deleting the duplicate rule.
