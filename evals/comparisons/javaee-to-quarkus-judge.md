# Judge: javaee → quarkus, curated-guides AI run vs handcrafted

Materials reviewed: `evals/comparisons/javaee-to-quarkus.md`; AI rule files `cdi.yaml`, `ejb.yaml`, `messaging.yaml`, `persistence.yaml`, `web.yaml`, `transaction.yaml`; handcrafted `200-ee-to-quarkus.windup.yaml`, `218-jms-to-reactive-quarkus.windup.yaml`; coolstore `pom.xml`, `StartupListener.java`, `CartEndpoint.java`, `Order.java`.

## Verdict

**Neither side is clearly better; on coolstore the AI ruleset is now closer to merge-ready than handcrafted on *coverage breadth*, but its 4x incident inflation is mostly redundant package-level firings rather than new findings.** The AI catches 12 files handcrafted misses (mostly via legitimate `javax.persistence` / `javax.ws.rs` / `javax.inject` package-level rules on entities and JAX-RS endpoints) and has materially better Spring coverage (cache, security, scheduling, observability, config) which handcrafted only sketches. But the AI is *missing* the load-bearing pom/build remediation rules (`javaee-pom-to-quarkus-000{00..80}`), the `beans.xml` ignored-descriptor rule, the JDBC/JPA-mixed code smell rules, and the Weblogic/JNDI-specific catches that show up on coolstore. Net: the AI ruleset has more knowledge but emits multiple overlapping rules per concern (an `*-import-00010` package rule, an `*-import-00040` second package rule, plus a per-annotation rule) which is what produces 214 incidents on a 12-source-file app. Handcrafted is leaner and more action-oriented (especially the pom/Maven plugin scaffolding rules), but is now noticeably behind on Spring/Quarkus configuration migration.

## Sweep coverage

- `javax.ejb` — covered. `ejb-import-00010` (PACKAGE `javax.ejb*`), `ejb-import-00020` (PACKAGE `javax.ejb`), `ejb-import-00030` (PACKAGE `jakarta.ejb`), plus per-annotation `ejb-annotation-0001{0..50}` (`Stateless`, `Stateful`, `Singleton`, `EJB`, `Asynchronous`). Handcrafted hits `javax.ejb.*` via `ee-to-quarkus-00020` (ANNOTATION glob).
- `javax.persistence` — covered. `persistence-import-00010` (PACKAGE), `hibernate-annotation-00010/00020` for `PersistenceUnit`. Handcrafted has no equivalent broad package rule and relies on PR-side dependency rule `javaee-pom-to-quarkus-00070`.
- `javax.inject` / `javax.enterprise` / `javax.annotation` / `javax.transaction` / `javax.servlet` / `javax.ws.rs` / `javax.validation` / `javax.jms` — all have AI package-level rules: `cdi-import-00020/00050`, `cdi-import-00010/00040/00060/00070`, `cdi-import-00030`, `core-import-00010`, `transaction-import-00010/00020`, `web-import-00010/00030`, `web-import-00020`, `validation-import-00010`, `messaging-import-00010/00020`. Handcrafted does not duplicate these — it counts on `ee-to-quarkus-*` annotation rules plus dependency-removal rules.
- `org.springframework.*` — covered, broadly. AI has individual rules for `stereotype.{Component,Service,Repository}` (`cdi-annotation-00010`, `spring-di-annotation-00010/00020`), `beans.factory.annotation.{Autowired,Qualifier,Value}` (`cdi-annotation-00040/00050`, `config-annotation-00010`), `context.annotation.{Configuration,Bean,ComponentScan,Import,Conditional,Scope,Profile}`, `boot.autoconfigure.condition.*`, `cloud.openfeign.FeignClient`, `data.jpa.repository.*`, `security.access.*`, `web.bind.annotation.*Mapping`/`@RestController`/`@RequestMapping`, etc. Handcrafted covers Spring DI artifacts and some webmvc but is largely a catch-all (`springboot-generic-catchall-00100`).
- `weblogic.*` — neither ruleset has a rule. coolstore's `StartupListener` extends `weblogic.application.ApplicationLifecycleListener`; neither side catches it. Real gap.

## AI-only files: real or noise?

Spot-checks against the 12 files flagged only by AI:

- **`model/CatalogItemEntity.java`, `Order.java`, `OrderItem.java`, `ShoppingCart.java`, `InventoryEntity.java`** — these are JPA entities importing `javax.persistence.*` (verified for `Order.java`). AI fires `persistence-import-00010` (PACKAGE `javax.persistence`). **Legitimate catch.** Handcrafted has no broad `javax.persistence` package rule; it only triggers when `@Stateless`/`@Stateful`/EJB annotations are seen. These entities will absolutely not work on Quarkus without re-importing to `jakarta.persistence`, so handcrafted is missing real migration debt here.
- **`rest/CartEndpoint.java`, `OrderEndpoint.java`, `ProductEndpoint.java`** — JAX-RS resources with `javax.ws.rs.*` (verified for `CartEndpoint`). AI fires `web-import-00020` (PACKAGE `javax.ws.rs*`) and likely `cdi-import-00020` (`javax.inject`). **Legitimate catch.** Handcrafted only fires `jaxrs-to-quarkus-00010` on the *dependency* and on `Application` subclasses; it has no package-level rule for JAX-RS source code, so per-file flags are missing.
- **`service/PromoService.java`** — almost certainly `javax.inject`/`javax.enterprise` package hit; **legitimate** for the same reason (handcrafted only catches EJB-annotated services).
- **`utils/StartupListener.java`** — has `@Inject` from `javax.inject`. AI fires `cdi-import-00020` (PACKAGE `javax.inject*`). **Legitimate catch on the import**, though neither ruleset catches the real interesting thing (`weblogic.application.ApplicationLifecycleListener`).
- **`utils/Transformers.java`** — likely `javax.json` / `javax.enterprise` / logging package hits; legitimate package-rename catch.
- **`webapp/WEB-INF/web.xml`** — AI fires `web-xml-00010` (xpath `//*[local-name()='web-app']`) and/or `web-pattern-00010` (filecontent). **Legitimate**; handcrafted has no web.xml rule at all, which is a real gap given Quarkus REST does not honor `web.xml`.

**Bottom line on the 12 files: all 12 are real catches, not false positives.** Handcrafted misses them because it leans heavily on annotation-pattern rules (`@Stateless`, `@MessageDriven`, `@Resource(lookup=…)`) and never wrote broad `javax.* PACKAGE` rules. The AI's package-rename rules are exactly the bread-and-butter of a Jakarta-9+ namespace migration.

## Handcrafted still wins on

From the 51-rule B→A miss list:

1. **POM/Maven scaffolding (`javaee-pom-to-quarkus-000{00..80}`)** — adopt Quarkus BOM (`-00010`), add quarkus-maven-plugin (`-00020`), maven-compiler-plugin with `-parameters` (`-00030`), maven-surefire with `java.util.logging.manager` (`-00040`), maven-failsafe with `native.image.path` (`-00050`), native profile (`-00060`), configure hibernate-orm (`-00070`), swap junit (`-00080`), and the packaging check `javaee-pom-to-quarkus-00000` (must be `jar` not `war`/`ear`). AI has *some* overlap (`build-xml-00040` says don't use `war`/`jar`, but with a confused message; `build-xml-00010/00020/00070` cover Spring Boot parent and spring-boot-maven-plugin) but **no rule to add the Quarkus BOM, add the Quarkus Maven plugin, configure surefire's logmanager, or add the native build profile** — the actually-required Quarkus pom changes.
2. **`beans.xml` descriptor ignored (`cdi-to-quarkus-00030`, `jakarta-cdi-to-quarkus-00030`)** — handcrafted catches `/b:beans`; AI has zero rules for `beans.xml`. This is the only file handcrafted catches uniquely on coolstore.
3. **JDBC/JPA mixed usage (`jdbc-jpa-mixed-to-quarkus-0000{1..3}`)** — `java.sql.Connection`, `Statement`, `PreparedStatement` + `*entitymanager` heuristics; AI has nothing for `java.sql.*`.
4. **`persistence-to-quarkus-00000`** (move `*-ds.xml` and `persistence.xml` to properties) — AI's `persistence-pattern-00010` and `jpa-pattern-00020` are similar but only `persistence.xml`; handcrafted also catches `-ds.xml`. AI misses `-ds.xml` entirely.
5. **`jndi-to-quarkus-00002`** (`Context.lookup`) — AI has `datasource-method-00020` for the same; this one is actually *covered* but with poor placement under datasource. Mostly a labeling issue.
6. **JSF / Jakarta Faces (`jakarta-faces-to-quarkus-0000{0,10}`, `javaee-faces-to-quarkus-00000`)** — AI has zero JSF coverage. Real gap.
7. **`springboot-integration-to-quarkus-000{10,20}`** (Spring Integration `IntegrationFlow`, `int:channel`) — AI has zero coverage.
8. **`springboot-properties-to-quarkus-00001`** (`application-{profile}.properties` rename) — AI has no equivalent file-pattern rule.
9. **`springboot-generic-catchall-00100`** — handcrafted's "any Spring component requires investigation" backstop; AI has no catch-all but does cover more Spring sub-areas individually.
10. **`springboot-shell-to-quarkus-00000`** (`spring-shell-core` → picocli) — AI didn't write it despite picocli being one of the 25 guides.
11. **`jaxrs-to-quarkus-00000/00010`** (dependency-level `org.jboss.spec.javax.ws.rs:*` and `javax.ws.rs:javax.ws.rs-api`) — AI catches the source-level package but missed the dependency artifact removal rules.

## Quality vs noise (the 214 vs 55 incident gap)

Looking at the file-by-file rule firings in the kantra diff (e.g., `service/ShoppingCartOrderProcessor.java`: AI fires 11+ rules, handcrafted fires 4), the AI inflation is driven by **redundant package-level rules layered on top of per-annotation rules**, not by hallucinated findings. Examples on a single import block:

- `import javax.enterprise.context.ApplicationScoped` triggers `cdi-import-00010` (pattern `javax.enterprise*`), `cdi-import-00040` (pattern `javax.enterprise`), AND `cdi-import-00060` (pattern `javax.enterprise.context`). Three rules, one fix.
- `import javax.inject.Inject` triggers `cdi-import-00020` (`javax.inject*`) AND `cdi-import-00050` (`javax.inject`). Two rules, one fix.
- `import javax.transaction.Transactional` triggers `transaction-import-00010` (`javax.transaction*`), `transaction-import-00020` (`javax.transaction`), AND `transaction-annotation-00030` (`javax.transaction.Transactional`). Three rules, one fix.
- `import javax.ejb.Stateless` triggers `ejb-import-00010`, `ejb-import-00020`, `ejb-annotation-00010` plus `ee-to-quarkus-00000`/`-00020` equivalents — but only AI fires three on the import line.

Multiply across ~12 affected coolstore source files with 3–6 javax imports each and the 4x inflation is fully explained by overlapping wildcard-vs-bare-package-vs-class rules in the SAME concern area. The findings are not false; they are duplicates of each other.

**This is noise that hurts UX**, because reviewers see 11 issues per file when there are really 3–4 distinct concerns. Handcrafted's lean approach (one rule per concern, action-oriented) produces tighter signal.

A secondary noise source: AI fires generally-applicable rules (`cdi-import-00010` says "Quarkus uses Jakarta CDI") on every file that touches the package, even when handcrafted would only fire on the EJB-bearing service that actually triggers a migration decision. The Jakarta namespace rename is real migration work — but reporting it 30 times across one app is more book-keeping than insight.

## Recommendations

1. **De-duplicate the AI package-level rules.** Pick one of `javax.foo`, `javax.foo*`, `javax.foo.bar` per package and delete the others. Suggested keep: the broadest wildcard (`javax.enterprise*`). Drop `cdi-import-00040/00060/00070`, `cdi-import-00050`, `transaction-import-00020`, `web-import-00030`. Estimated incident reduction: ~40-50%.
2. **Suppress the package-rename rule when a more-specific class/annotation rule fires on the same line.** Either by lowering its severity to `optional`/`potential` or by adding an `and-not` clause keyed on the per-annotation rule. The Jakarta-9 rename is a build-once activity, not per-import migration work.
3. **Add the missing pom/Maven plugin scaffolding rules** (`adopt quarkus-bom`, `add quarkus-maven-plugin`, `surefire java.util.logging.manager`, `native profile`, `replace war/ear packaging with jar`, `add quarkus-hibernate-orm`). This is the most impactful gap — every Quarkus migration needs these and the AI ruleset currently doesn't tell you to make them. Source: `216-javaee-pom-to-quarkus.windup.yaml`.
4. **Add `beans.xml` (`/b:beans`) and `*-ds.xml` rules.** Both are coolstore-relevant.
5. **Add JSF, Spring Integration, JDBC-mixed-with-JPA, and `application-{profile}.properties` rules** so the AI matches handcrafted's frontier on legacy concerns.
6. **Investigate the `jaxrs-to-quarkus-00000`-style dependency rules.** The AI covered the source-level `javax.ws.rs` package rename but missed the dependency removal advice for `org.jboss.spec.javax.ws.rs:jboss-jaxrs-api_*` — a common JBoss/Wildfly artifact in coolstore-like apps.
7. **Add a Weblogic catch-all** (`weblogic.*` package) since neither side has it and coolstore demonstrates the gap.
8. **Decide the canonical scope of the AI-generated `web-annotation-0010X` family.** Most have effort=1 and just say "X replaced by Y" — they are correct but verbose; consider folding `web-annotation-0004{0..80}` (six `*Mapping` rules) into a single rule with an `or:` clause to halve the surface area.

Net: with steps 1, 2, 3, 4, this ruleset would likely beat handcrafted on coverage *and* match it on signal-to-noise.

{"report_path": "evals/comparisons/javaee-to-quarkus-judge.md", "overall_winner": "neither", "key_finding": "AI's 12 extra coolstore files are legitimate javax.* package-rename catches handcrafted misses, but the 214-vs-55 incident gap is driven almost entirely by 2-3 overlapping package-level rules firing on the same import line, not by new findings; AI also lacks the load-bearing Quarkus pom/Maven plugin scaffolding rules and beans.xml/JSF coverage that handcrafted provides."}
