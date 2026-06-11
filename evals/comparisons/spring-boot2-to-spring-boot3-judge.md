# Judge: ai vs handcrafted (spring-boot2 → spring-boot3)

App under test: spring-petclinic @ 276880ed (Spring Boot 2.7.3).

## Verdict

**AI wins on the load-bearing javax → jakarta sweep on this app**, and that single class of finding is the one that will hit virtually every real Spring Boot 2 → 3 migration. Handcrafted wins on **breadth and specificity of the long tail** (159 of 170 rules have no AI equivalent), particularly removed APIs, Spring Cloud version floors, and getter/setter‑level property method removals.

Per app, the AI ruleset produced more actionable signal (47 incidents on 13 files, all in legitimate Jakarta migration sites, plus a Spring Boot parent and Spring 5 core warning on `build.gradle`). Handcrafted produced only 10 incidents on 5 files, and missed every single one of the petclinic entity/controller files that import `javax.persistence` or `javax.validation`. Across a broader corpus of apps, however, handcrafted will pick up many things AI never wrote (Spring Cloud version floors, Hibernate naming strategy, `LocalServerPort`, the entire `Mustache*`/`Flyway*`/`Dynatrace*` property removal set).

Overall on this app: **ai-generated**. Across the migration domain: **complementary**, with handcrafted having a larger surface area but missing the most impactful sweep on the most common case.

## Sweep rule coverage

The Jakarta EE 9+ namespace move is the single highest-volume change in Spring Boot 3.

**AI (`evals/spring-boot2-to-spring-boot3/rules/jakarta.yaml`)** has 10 dedicated rules at `location: PACKAGE`:
- `jakarta-import-00010` `javax.servlet`
- `jakarta-import-00020` `javax.persistence`
- `jakarta-import-00030` `javax.annotation`
- `jakarta-import-00040` `javax.validation`
- `jakarta-import-00050` `javax.transaction`
- `jakarta-import-00060` `javax.ejb`
- `jakarta-import-00070` `javax.jms`
- `jakarta-import-00080` `javax.mail`
- `jakarta-import-00090` `javax.ws.rs`
- `jakarta-import-00100` `javax.xml.bind`

Plus `jakarta-dependency-00010` on the `javax.servlet:javax.servlet-api` artifact.

**Handcrafted** has exactly one Jakarta rule: `spring-boot-2.x-to-3.0-core-changes-00060`. Its `when` is `or` of `java.dependency: javax.servlet.javax.servlet-api` and `java.referenced: pattern: javax.servlet* location: IMPORT`.

Three real gaps in the handcrafted rule:
1. **Only `javax.servlet*`** — not `javax.persistence`, `javax.validation`, `javax.annotation`, `javax.transaction`, `javax.jms`, `javax.mail`, `javax.ws.rs`, `javax.xml.bind`. Petclinic does not import `javax.servlet` directly anywhere; it imports `javax.persistence.*` and `javax.validation.constraints.*`. The result: handcrafted catches **zero** Jakarta sites on petclinic, AI catches every one.
2. **`location: IMPORT` vs `location: PACKAGE`** — `PACKAGE` matches qualified-name references too, not just import statements. For an app that already uses fully qualified names or static references, AI catches more.
3. **No artifact rules for the non-servlet Jakarta artifacts** (e.g., `jakarta.persistence:jakarta.persistence-api`, `jakarta.validation:jakarta.validation-api`).

Effort numbers are comparable (3-5 across both rulesets). Message bodies are roughly equivalent in quality; AI's messages are slightly more migration-oriented per package.

## What handcrafted catches that AI misses

The 159 handcrafted-only rules are not all noise — there are real classes of finding AI omitted.

**Removed APIs (the bulk of `spring-boot-2.x-to-3.0-removals.yaml`, IDs 00010 → 01200).** These are individual getters/setters/constructors on `MustacheProperties`, `FlywayProperties`, `DynatraceProperties`, `WebMvcProperties`, `GangliaProperties`, the Spring Boot Loader `Archive`/`Packager`/`Repackager`, `IncludeExcludeEndpointFilter`, `HealthEndpoint`, etc. The AI emitted nothing at this level — its `removals` lane stops at class‑level imports for a handful of types. Examples handcrafted catches and AI doesn't:
- `spring-boot-2.x-to-3.0-removals-00010` `AbstractDataSourceInitializer` (and `00020` inheritance variant)
- `spring-boot-2.x-to-3.0-removals-00040` `ConfigFileApplicationListener`
- `spring-boot-2.x-to-3.0-removals-00090` `SpringPhysicalNamingStrategy` (the Hibernate naming change — very common)
- `spring-boot-2.x-to-3.0-removals-00170` `LocalServerPort` (annotation, common in tests)
- `spring-boot-2.x-to-3.0-removals-00580`-00640 the Flyway `setIgnore*`/`isIgnore*` property setter family
- `spring-boot-2.x-to-3.0-removals-00530` → 00850 the Mustache property accessor family (~25 rules)
- `spring-boot-2.x-to-3.0-removals-00390`/00490/00990 Dynatrace `getDeviceId`/`getTechnologyType`/`setTechnologyType`

**Spring Cloud version floors.** `spring-boot-2.x-to-3.0-dependencies-00001` → 00016 enforce minimum versions of 14 individual Spring Cloud artifacts plus `spring-cloud-dependencies`. AI has none of these.

**Other useful misses by AI:**
- `spring-boot-2.x-to-3.0-dependencies-00020` `spring-boot-starter-parent` upgrade (handcrafted fires on petclinic's `pom.xml`)
- `spring-boot-2.x-to-3.0-core-changes-00001` image banner file detection (`banner.gif`/`jpg`/`png`)
- `spring-boot-2.x-to-3.0-core-changes-00030` `spring.factories` file presence
- `spring-boot-2.x-to-3.0-core-changes-00050` logging date format change (`LOG_DATEFORMAT_PATTERN`)
- `spring-boot-2.x-to-3.0-webapp-changes-00010` `SmartLifecycle` graceful shutdown phase change
- `spring-boot-2.x-to-3.0-webapp-changes-00020` Jetty 11 floor (and `-00030` path matching change in YAML, `-00040` `ErrorController.getErrorPath`)
- `spring-boot-2.x-to-3.0-session-00010` Spring Session 3.0 floor
- `spring-boot-2.x-to-3.0-micrometer-00020`/00030/00040 Micrometer JVM metrics and 1.10 baseline
- `spring-boot-2.x-to-3.0-datasource-00000` `spring.data` properties prefix flag

There is also significant duplication inside the handcrafted ruleset: `spring-boot-2.x-to-3.0-removals-00100`/00210, 00110/00220, 00120/00230, 00150/00250, 00160/00260, 00170/00270, and 00090/00190 are clear duplicates. Real catalogue is closer to ~155 unique rules than 170.

## Why AI emits more incidents (47 vs 10)

On this app the AI's higher incident count is **not** overlapping-rule duplication (the quarkus pattern). It's a single dynamic: the `jakarta-import-*` PACKAGE rules each fire once per file per package, and petclinic touches `javax.persistence` and `javax.validation` from 11 files (entities + controllers). That alone explains roughly 22 of the 47 incidents. The remaining incidents come from `core-dependency-00010`/`-00020`/`data-dependency-00010` (Hibernate group ID) etc. all firing on `build.gradle` (where the dependency declarations exist) — but each is a legitimately distinct migration concern.

No fanout from overlapping rules was observed for petclinic.

## AI-only files: real or noise?

All 12 files are **legitimate**:

| File | Why AI flags | Verdict |
|---|---|---|
| `model/BaseEntity.java` | imports `javax.persistence.{Id,GeneratedValue,MappedSuperclass}` | real |
| `model/NamedEntity.java` | imports `javax.persistence.{Column,MappedSuperclass}` | real |
| `model/Person.java` | imports `javax.persistence.MappedSuperclass`, `javax.validation.constraints.NotEmpty` | real |
| `owner/Owner.java` | imports 8 `javax.persistence.*`, 2 `javax.validation.constraints.*` | real (heaviest hit) |
| `owner/OwnerController.java` | `javax.validation.Valid` | real |
| `owner/Pet.java` | `javax.persistence.*` | real |
| `owner/PetController.java` | `javax.validation.Valid` | real |
| `owner/PetType.java` | `javax.persistence.Entity` | real |
| `owner/Visit.java` | `javax.persistence.*` | real |
| `owner/VisitController.java` | `javax.validation.Valid` | real |
| `vet/Specialty.java` | `javax.persistence.*`, `javax.xml.bind.annotation.*` | real |
| `vet/Vet.java` | `javax.persistence.*`, `javax.xml.bind.annotation.*` | real |

Every one of these files would break compilation on Spring Boot 3 unless the imports are migrated. Handcrafted misses every single one because its only Jakarta rule is keyed on `javax.servlet*`. That is the most important quality finding in this comparison.

## Quality differences

**Where AI is sharper:**
- Jakarta sweep coverage (10 packages vs 1, PACKAGE vs IMPORT) — directly load-bearing.
- Specific API rename rules at the right scope: `metrics-import-00010` `WebMvcMetricsFilter`, `metrics-import-00020` `MetricsRestTemplateCustomizer`, `metrics-import-00030`/00040 `WebMvcTagsProvider`/`Contributor`, `metrics-import-00070`/00080 `RestTemplateExchangeTagsProvider`/`WebClientExchangeTagsProvider`. Handcrafted does **not** cover any of these — they're listed in 159-only as gaps in the matrix but actually they're gaps in handcrafted that AI fills. (These are visible in the "Rules in ai with no equivalent in handcrafted" list.)
- Property-pattern rules for Cassandra (`config-pattern-00010`), Redis (`config-pattern-00020`), Actuator metrics export (`metrics-pattern-00010`), httptrace (`actuator-pattern-00010`), keys-to-sanitize (`actuator-pattern-00020`), SAML2 identity-provider (`security-pattern-00010`) — all of which are real renames called out in the migration guide.
- Build-system patterns: Gradle `mainClassName` (`build-pattern-00010`), `isEnabled` Kotlin DSL (`build-pattern-00020`), `buildInfo` exclude (`build-pattern-00030`), `<fork>` Maven plugin (`build-xml-00010`).

**Where handcrafted is sharper:**
- Breadth of removed-API surface (the entire `removals` lane below class-level).
- Spring Cloud and Spring Session version floors.
- Spring Boot parent version floor (catches `pom.xml`, which AI's `core-dependency-00010` keyed on `org.springframework.boot:spring-boot` does not on petclinic — petclinic uses `spring-boot-starter-parent`, not direct `spring-boot`).
- File-based detection for `spring.factories`, banner images, and `application*.properties` patterns (`spring.data`, `pathmatch`, session).
- Logging date format pattern, ErrorController interface implementation, SmartLifecycle implementation, Jetty 11 floor.

**Handcrafted weaknesses:**
- Internal duplication (~10 duplicate rule pairs in `removals`).
- Jakarta sweep is one-package-deep and IMPORT-only, missing the dominant case.
- Sometimes too-broad triggers (e.g., `dep:org.springframework.batch.spring-batch-core` for the "running multiple batch jobs" rule).

**AI weaknesses:**
- Long tail of `*Properties.setX()` removals is absent.
- No Spring Cloud floors.
- No `spring-boot-starter-parent` version rule (so won't fire on Maven projects that don't declare `spring-boot` directly).
- Missing some valuable file-content rules (banner images, `spring.factories` file, `application*.properties` content).

## Recommendations

1. **Adopt AI's `jakarta-import-*` PACKAGE rules into handcrafted.** This is the single biggest quality improvement available — handcrafted currently misses the most common Spring Boot 3 migration site in any non‑servlet-style app.
2. **Adopt AI's targeted metrics/actuator rename rules** (`metrics-import-00010`/00020/00030/00040/00070/00080, `actuator-import-00010`, `actuator-pattern-00010`/00020). Handcrafted has no equivalent and these are explicit in the migration guide.
3. **Adopt AI's config-property pattern rules** (`config-pattern-00010` Cassandra, `config-pattern-00020` Redis, `metrics-pattern-00010` metrics export, `security-pattern-00010` SAML2). Cheap and accurate.
4. **For ai-rule-gen:** add a generator pass over property-bag getters/setters (the `Mustache*`/`Flyway*`/`Dynatrace*` removal patterns). These are repetitive and well-documented in the migration guide.
5. **For ai-rule-gen:** add a Spring Cloud / Spring Session / Spring Boot parent version-floor template — this category was completely missed.
6. **For ai-rule-gen:** add file-content rules for `spring.factories`, `banner.*`, and a `pathmatch`/`spring.data` sweep on `application*.properties|yaml`.
7. **De-duplicate handcrafted removals** (00100/00210, 00110/00220, 00120/00230, 00150/00250, 00160/00260, 00170/00270, 00090/00190).
8. **Replace handcrafted's single Jakarta rule with a per-package PACKAGE rule set** rather than a single `javax.servlet*` IMPORT rule.

{"report_path": "evals/comparisons/spring-boot2-to-spring-boot3-judge.md", "overall_winner": "ai-generated", "key_finding": "AI catches the load-bearing javax.persistence/javax.validation sweep across 12 petclinic files via 10 PACKAGE-level Jakarta rules, while handcrafted's single javax.servlet IMPORT rule misses every site on this app despite its 170-rule breadth elsewhere."}
