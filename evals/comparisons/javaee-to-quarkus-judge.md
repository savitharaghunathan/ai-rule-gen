# Judge verdict: javaee-to-quarkus (ai-generated vs handcrafted)

Materials reviewed:
- `evals/comparisons/javaee-to-quarkus.md` (coverage matrix + kantra diff against coolstore)
- `output/javaee-to-quarkus-benchmark/guide.md` (the OpenRewrite recipe pages used as AI input; 2,939 lines, four nested recipes)
- All 5 AI ruleset files in `evals/javaee-to-quarkus/rules/`
- Handcrafted spot-checks: `200-ee-to-quarkus`, `201-persistence-to-quarkus`, `202-remote-ejb-to-quarkus`, `210-cdi-to-quarkus`, `216-javaee-pom-to-quarkus`, `218-jms-to-reactive-quarkus`, `238-jndi-to-quarkus`

## Verdict

**Handcrafted wins decisively for real-world migration use.** It fires 25 rules / 55 incidents on coolstore vs. the AI ruleset's 8 rules / 12 incidents, and exclusively flags 6 files that the AI ruleset misses entirely (`RestApplication.java`, two MDB files, `Producers.java`, `persistence.xml`, `beans.xml`). The AI ruleset is technically accurate within its scope and writes noticeably more actionable messages per rule, but its scope is far too narrow to be a usable javaee→quarkus ruleset on its own. The bottleneck was primarily the source guide — OpenRewrite's javaee→Quarkus recipes simply do not describe JMS, JNDI, transactions, persistence.xml, beans.xml, JAX-RS activation, or Spring-Boot at all — but the AI generator also missed easy wins that were in the guide (e.g. `@EJB`/JNDI references mentioned in the EJB removal narrative, `@Resource` mentioned for JMS examples).

## Accuracy

Every AI rule I read is technically correct and well-targeted. No false claims, no obviously over-broad selectors.

- `ejb-annotation-00010` / `-00020` / `-00030` (`@Stateless`, `@Stateful`, `@Singleton`) — accurate replacement targets. The recommendation to use `@Dependent` for `@Stateless` is conservative/safe; handcrafted `ee-to-quarkus-00000` suggests `@ApplicationScoped`, which is what most coolstore-style EJBs actually want. Both are defensible; the AI's note about `@Dependent` lifecycle is correct.
- `ejb-annotation-00040` (`@Local` must be removed) — correct and well-explained, including the consequence on `@Remote`.
- `ejb-annotation-00050` (`@EJB` → `@Inject`) — correct; the call-out about dropping `lookup=`/`mappedName=` is more rigorous than the handcrafted set's catch-all `ee-to-quarkus-00020` (`javax.ejb.*`).
- `persistence-annotation-00010` (`@PersistenceContext` → `@Inject`) — accurate, with a useful note about `@io.quarkus.hibernate.orm.PersistenceUnit` qualifiers for multi-PU setups that the handcrafted `persistence-to-quarkus-00010` lacks.
- `build-xml-00010` … `build-xml-00050`, `build-pattern-00010` — all correct on their narrow XPath assertions. Two issues:
  - `build-xml-00020` and `build-xml-00050` are duplicates (same XPath, same condition). Both fire on `<packaging>war</packaging>`; in the coolstore run both fired on the same line, inflating incident counts cosmetically.
  - `build-pattern-00010` overlaps `build-xml-00030`/`-00040` (the filecontent regex covers what the two XPaths already cover). Not wrong, but redundant — 3 rules where 1 would do.
- `dependencies-dependency-00010/20/30` — accurate selectors for `dep:javax.javaee-api`, `dep:javax.annotation.javax.annotation-api`, `dep:javax.enterprise.cdi-api`. These are roughly equivalent to handcrafted `cdi-to-quarkus-00000` and parts of `211-dependency-removal-for-quarkus`. Comparable.

No accuracy problems worth flagging beyond the duplication.

## Completeness

The AI covered ~4 of ~22 "themes" the handcrafted ruleset addresses. The gaps fall into two buckets:

**Missing because the guide doesn't mention them at all** (guide bottleneck, not AI fault):
- JMS → SmallRye Reactive Messaging (handcrafted `jms-to-reactive-quarkus-00000`..`-00050`, 6 rules; hit `ShoppingCartOrderProcessor`, the MDB classes, on coolstore).
- JNDI `InitialContext` / `Context.lookup()` (handcrafted `jndi-to-quarkus-00001`/`-00002`; hit `ShoppingCartService` on coolstore).
- `@Transactional` on `EntityManager.persist/merge/remove` (handcrafted `transaction-to-quarkus-00001`..`-00003`; hit `CatalogService`, `OrderService` on coolstore). The AI guide does mention `@Transactional` prose-wise in the `@Stateless` rule, but not as a callable pattern.
- `persistence.xml` / `*-ds.xml` migration (handcrafted `persistence-to-quarkus-00000`; hit on coolstore).
- `beans.xml` ignored (handcrafted `cdi-to-quarkus-00030`; hit on coolstore).
- JAX-RS `@ApplicationPath`/`Application` activation cleanup (handcrafted `jaxrs-to-quarkus-00020`, `jakarta-jaxrs-to-quarkus-00020`; hit `RestApplication.java` on coolstore).
- JDBC/JPA mixed usage warnings (`jdbc-jpa-mixed-to-quarkus`).
- All of Spring-Boot → Quarkus (handcrafted has 30+ rules; irrelevant for coolstore but huge real-world surface).
- Quarkus BOM adoption / Quarkus Maven plugin / Surefire LogManager / Failsafe native profile (handcrafted `javaee-pom-to-quarkus-00010`..`-00060`). The AI guide actually shows the BOM and quarkus-maven-plugin in the "After" pom example; the AI generator chose only to flag what to remove, not what to add.
- Jakarta-namespace variants of every rule (handcrafted has parallel `jakarta-*` rules for CDI, faces, JAX-RS).

**Missing despite being present or implied in the guide** (AI extraction gaps):
- `@Resource(lookup=...) Queue` / `Topic` field patterns — the guide doesn't dwell on JMS, fair enough.
- `@MessageDriven` — also not in the guide; fair.
- `@Remote` — the AI ruleset mentions remote EJB in prose inside `ejb-annotation-00040` but produced no detection rule for `javax.ejb.Remote` despite it being trivially within scope of the JavaEE code-migration recipe. Handcrafted `remote-ejb-to-quarkus-00000` fired on `ShippingService` on coolstore.
- Quarkus BOM / `quarkus-maven-plugin` / `maven-compiler-plugin -parameters` / `maven-surefire-plugin` + `org.jboss.logmanager.LogManager` — all shown verbatim in the AI's input guide ("After" pom.xml on lines 106-200). The generator chose to emit only "remove this" rules and not "you must add this" rules, which is a real coverage miss.
- Drop `<scope>provided</scope>` — mentioned in prose by `dependencies-dependency-00010` but no dedicated rule.

The AI ruleset is 15 rules and the handcrafted is 82 (the comparison header says 82; the on-disk file count is 35 because some files contain many rules). Even discounting Spring-Boot (~32 handcrafted rules that have no source-of-truth equivalent in the guide), the AI is short by ~25–30 javaee-relevant rules.

## Actionability

This is the one dimension where the AI ruleset is **noticeably better than the handcrafted one**, rule-for-rule.

AI messages follow a consistent structure: 1-2 sentence summary → "Before"/"After" code block → "Migration Steps" or "Additional Info" with caveats and gotchas. Examples worth calling out:
- `ejb-annotation-00050` warns to drop `lookup`, `mappedName`, `beanInterface`, `name` attributes and to use CDI qualifiers instead — a class of error the handcrafted `ee-to-quarkus-00020` catch-all does not address.
- `ejb-annotation-00030` warns about the `javax.ejb.Singleton` vs `javax.inject.Singleton` package collision — a genuine footgun.
- `persistence-annotation-00010` explains `@PersistenceUnit` qualifier for multiple PUs, which the handcrafted equivalent doesn't.

The handcrafted messages are terser, often a single sentence ("Stateless EJBs can be converted to a CDI bean by replacing the `@Stateless` annotation with a scope eg `@ApplicationScoped`") and occasionally have formatting issues (literal `{{` `}}` in `jms-to-reactive-quarkus-00020`, awkward inline `\n` escapes in `216-javaee-pom-to-quarkus`). For a developer doing the rewrite, the AI rules give better remediation guidance per incident.

That said, actionable text doesn't matter when the rule never fires. The handcrafted ruleset's terser messages still describe correct fixes.

## Coverage on coolstore

The kantra diff is unambiguous: 8 rules / 12 incidents (AI) vs 25 rules / 55 incidents (handcrafted). Drilling into the 6 files only handcrafted flags:

| File | What the AI missed | Rule needed |
|---|---|---|
| `RestApplication.java` | JAX-RS Application class is obsolete in Quarkus | `jakarta-jaxrs-to-quarkus-00020` |
| `InventoryNotificationMDB.java`, `OrderServiceMDB.java` | `@MessageDriven` MDBs | `jms-to-reactive-quarkus-00010`/`-00020` |
| `Producers.java` | `@Produces EntityManager` is illegal in Quarkus | `persistence-to-quarkus-00011` |
| `META-INF/persistence.xml` | Move to `application.properties` | `persistence-to-quarkus-00000` |
| `WEB-INF/beans.xml` | Descriptor ignored | `cdi-to-quarkus-00030` |

Even on the 9 shared files, the AI typically fires 1 rule where handcrafted fires 3-4 (e.g. on `pom.xml`: AI fires `build-xml-00010/20/50` + `build-pattern-00010` which are largely the same finding restated; handcrafted fires 7 distinct `javaee-pom-to-quarkus` rules covering BOM, plugins, surefire, failsafe, native profile). Real coolstore migration would need most of those.

Practical interpretation: a developer pointed at the AI ruleset's output would think "I just need to change `@Stateless` annotations, bump Java, and remove some deps." They would then face a non-functional Quarkus app because nothing told them to handle MDBs, persistence.xml, JNDI lookups, or to add the Quarkus BOM/plugin.

## Was the guide the problem?

**Mostly yes, but not entirely.**

What the guide does cover (and the AI extracted competently):
- Remove `javax:javaee-api`, `javax.annotation`, `javax.enterprise:cdi-api`
- Bump `maven.compiler.source/target` to 11
- Drop `maven-war-plugin` / `war` packaging
- `@Stateless` → `@Dependent`, `@Stateful` → `@SessionScoped`, `@Singleton` → `@ApplicationScoped`, `@Local` removed, `@EJB` → `@Inject`
- `@PersistenceContext` → `@Inject EntityManager`

What the guide doesn't cover at all:
- JMS, JNDI, transactions, JAX-RS Application/`@ApplicationPath`, `@Remote` EJB, `@MessageDriven`, JSF/Faces, persistence.xml/`beans.xml`, JDBC/JPA mixing, Spring-Boot.

That covers nearly every gap. The guide is genuinely narrow because OpenRewrite's `JavaEEtoQuarkus2Migration` is itself a narrow automation focused on what they can mechanically rewrite, not a comprehensive migration assessment guide. The disclaimer at the top of the guide even says: *"Additional transformations like JSF, JMS, Quarkus Tests may be necessary"* — i.e., the source admits up front that JMS/JSF are out of scope.

What is *not* the guide's fault:
- The AI emitted 5 separate pom-removal rules but 0 pom-addition rules (BOM, quarkus-maven-plugin, surefire LogManager, native profile), even though the guide's "After" example shows all of those literally. This was a generator extraction choice, not a guide deficiency.
- The AI did not emit a `javax.ejb.Remote` rule even though the EJB code-migration recipe is the natural home for it and the AI even mentions it in prose elsewhere.
- The AI did not produce Jakarta-namespace parallels of any rule. The guide is javax-focused, but if the generator understands "javax→jakarta" as a separate concern it could have doubled coverage cheaply.
- Three rules cover the Java-version bump and two rules cover war-packaging removal — the generator emitted redundant detectors for the same finding from different XPath/regex angles.

**Bottom line:** a different guide (the official Quarkus migration guide on quarkus.io, or the Red Hat "Migrating to Quarkus" docs, both of which the handcrafted ruleset clearly draws from) would have closed ~70% of the gap. The remaining ~30% is generator behavior — bias toward "remove" rules over "add" rules, missed redundancy consolidation, and no Jakarta-namespace expansion.

## Recommendations

For the AI rule generator:
1. **Use a broader source.** Concatenate the OpenRewrite recipe pages *plus* quarkus.io/guides (cdi-reference, hibernate-orm, maven-tooling, transaction, getting-started) *plus* the Red Hat "Migration Toolkit for Runtimes" Quarkus guide. The OpenRewrite recipe alone is structurally insufficient because its scope is mechanical-rewrite-only.
2. **Generate "add" rules, not just "remove" rules.** When the guide shows a target-state pom.xml, emit rules detecting the *absence* of required elements (BOM, quarkus-maven-plugin, native profile, `-parameters`, jboss LogManager). The handcrafted `javaee-pom-to-quarkus-000{10,20,30,40,50,60}` rules are entirely "absence" detectors and they fire on coolstore.
3. **Auto-emit Jakarta-namespace parallels.** Every `javax.*` Java rule should have a `jakarta.*` `or:` branch, and every `javax.*` Maven coord rule should have a `jakarta.*` sibling. Cheap doubling of coverage.
4. **Deduplicate overlapping detectors.** `build-xml-00020` and `build-xml-00050` have identical XPath. `build-pattern-00010` is subsumed by `build-xml-00030`+`-00040`. Detect this during generation.
5. **Extract entity names from prose, not just code blocks.** The EJB rule prose explicitly mentions `@Remote` and JNDI lookup strings; the generator missed both as detection targets.

For the eval harness:
1. The 8 / 25 rule-fired and 12 / 55 incident counts are the headline metrics. Surface them in the comparison file's TL;DR.
2. Flag duplicate-XPath rules in the comparison output so we can see when a high incident count is real coverage vs. the same finding restated.
3. Consider scoring "actionability" of messages — the AI ruleset wins clearly on this axis and it's invisible in the current rubric.

For users wanting to migrate javaee→quarkus today: use the handcrafted ruleset. The AI ruleset's messages are richer where it overlaps, but it leaves too many critical findings on the table.

{"report_path": "evals/comparisons/javaee-to-quarkus-judge.md", "overall_winner": "handcrafted", "key_finding": "AI ruleset is accurate and writes better remediation messages but covers only ~4 of ~22 migration themes — primarily because the OpenRewrite recipe guide omits JMS/JNDI/persistence.xml/transactions/JAX-RS-Application/Spring entirely, and secondarily because the generator emitted only removal rules and skipped the BOM/plugin addition rules its own source guide showed."}
