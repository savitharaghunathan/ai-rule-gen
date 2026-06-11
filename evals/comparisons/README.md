# Comparison runs

Side-by-side numbers for ai-rule-gen vs the hand-authored rulesets at konveyor/rulesets. Generated artifacts in this dir:

- `javaee-to-jakarta.md` / `javaee-to-quarkus.md` — `cmd/compare` output on coolstore (coverage matrix + kantra-analyze diff)
- `javaee-to-{jakarta,quarkus}.{ejb-remote,ejb-security,tasks-qute}.md` — same diff against three other sample apps
- `javaee-to-jakarta-judge.md` / `javaee-to-quarkus-judge.md` — LLM judge writeups (head-to-head, sampled rules)

## Setup

| Migration | AI source | Handcrafted source |
| --- | --- | --- |
| javaee → jakarta | Red Hat EAP 8 Migration Guide PDF | `stable/java/eap8` |
| javaee → quarkus | 25 quarkus.io guides cited by the handcrafted ruleset, concatenated | `stable/java/quarkus` |

Sample apps cloned at `~/scratch/eval-apps/`:

- `konveyor-ecosystem/coolstore` @ b5752b9 — large javaee 7 monolith (primary)
- `konveyor-ecosystem/ejb-remote` — small EAP quickstart
- `konveyor-ecosystem/ejb-security` — small EAP quickstart
- `konveyor-ecosystem/tasks-qute` — eap tasks-jsf, in-progress quarkus migration

## Coolstore numbers

|                                | jakarta AI | jakarta handcrafted | quarkus AI | quarkus handcrafted |
| ------------------------------ | ---------- | ------------------- | ---------- | ------------------- |
| rules                          | 292        | 340                 | 275        | 82                  |
| quality (cmd/eval)             | 5.3 / 6    | 5.1 / 6             | 5.5 / 6    | 5.0 / 6             |
| coolstore — rules fired        | 2          | 13                  | 32         | 25                  |
| coolstore — incidents          | 109        | 118                 | 214        | 55                  |
| coolstore — files only here    | 0          | 3                   | 12         | 1                   |
| coolstore — files in both      | 24         | 24                  | 14         | 14                  |

Coverage matrix totals (per the `.md` reports):

|                                                 | jakarta | quarkus |
| ----------------------------------------------- | ------- | ------- |
| AI rules with handcrafted equivalent            | 63 / 292 | 24 / 275 |
| handcrafted rules with AI equivalent            | 74 / 340 | 20 / 82  |

## Other sample apps (kantra diff only)

Eleven additional apps: three konveyor-ecosystem migration samples and eight `jboss-developer/jboss-eap-quickstarts` (7.4.x). Each cell is `rules-fired / incidents`.

|                | jakarta AI | jakarta hc | quarkus AI | quarkus hc |
| -------------- | ---------- | ---------- | ---------- | ---------- |
| ejb-remote     | 1 / 1      | 0 / 0      | 1 / 1      | 7 / 20     |
| ejb-security   | 1 / 1      | 0 / 0      | 1 / 1      | 6 / 6      |
| tasks-qute     | 0 / 0      | 7 / 18     | 5 / 5      | 12 / 15    |
| kitchensink    | 9 / 69     | 19 / 86    | 23 / 104   | 21 / 32    |
| helloworld-jms | 3 / 9      | 2 / 5      | 6 / 14     | 9 / 13     |
| helloworld-mdb | 6 / 30     | 8 / 34     | 16 / 59    | 16 / 43    |
| helloworld-rs  | 3 / 8      | 5 / 12     | 5 / 12     | 12 / 13    |
| helloworld-ws  | 5 / 7      | 8 / 12     | **1 / 1**  | **11 / 11**|
| cmt            | 7 / 75     | 15 / 86    | 23 / 147   | 21 / 47    |
| hibernate      | 9 / 50     | 17 / 65    | 20 / 71    | 17 / 26    |
| bmt            | 5 / 29     | 10 / 38    | 21 / 59    | 16 / 20    |
| **totals (12 apps incl. coolstore)** | **51 / 388** | **104 / 474** | **154 / 688** | **173 / 301** |

Patterns across apps:

- **Jakarta**: AI fires roughly half as many distinct rules as handcrafted but catches ~82% of the incidents — its broad sweep rules carry coverage with few rules.
- **Quarkus**: AI fires 2.3× more incidents in aggregate. Per the quarkus judge, ~50% of that is overlapping rules (broad package + specific class + bare-name firing on the same import line), not new findings.
- **Real coverage gaps**: AI has essentially zero JAX-WS → Quarkus coverage (`helloworld-ws`: 1/1 vs 11/11). None of the 25 quarkus.io guides we ingested cover JAX-WS migration directly. JAX-RS coverage is thin too (`helloworld-rs`: 5/12 on quarkus).
- **AI wins on**: hibernate quarkus, cmt/bmt quarkus, helloworld-jms jakarta, helloworld-mdb jakarta. Mostly cases where its sweep rules apply cleanly to focused codebases.

## Notes

- Round 1 of jakarta missed the `javax → jakarta` import sweep rule. Traced to a regex tolerance in the bespoke PDF-to-markdown step that dropped section 4.1 of the source guide before the section indexer ran. Round 2 (above) re-ran the pipeline with that fixed; AI emitted the sweep rule (`namespace-import-00010`) and coolstore numbers landed close to handcrafted.
- AI sweep pattern is `javax*` (PACKAGE location). Handcrafted enumerates 21 EE roots (`javax.(activation|annotation|batch|...|xml)`) and uses a capture group to template the matched name into the message. AI version will false-positive on Java SE packages and has a static message.
- AI didn't write the companion sweeps for XML namespace URIs, `META-INF/services/javax.*`, or `<property name="javax.*">` rename, all called out in the same guide section. Handcrafted has rules for each.
- Round 1 of quarkus used the OpenRewrite "Migrate JavaEE to Quarkus 2" recipe pages (~15 AI rules). Round 2 (above) used the 25 quarkus.io guides cited by the handcrafted ruleset (~275 AI rules, 32 firing on coolstore). The judge attributes the 214-vs-55 incident inflation to overlapping rules — e.g. a single `import javax.enterprise.context.ApplicationScoped` triggers three different AI rules. Real findings, but inflated counts. Handcrafted still wins on Quarkus pom scaffolding (BOM, quarkus-maven-plugin, native profile), `beans.xml`, JSF, JDBC/JPA-mixed, and `application-{profile}.properties` rename.
- `cmd/eval` couldn't load the handcrafted rulesets before this branch; the strict YAML types rejected `'{{xmlfiles1.filepaths}}'` template chains and the `nameregex` field alias. Both are now accepted (`internal/rules/lenient.go`), so the quality column above is apples-to-apples.

## Re-running

The handcrafted rule dirs are gitignored — populate them from konveyor/rulesets first:

```
git clone https://github.com/konveyor/rulesets.git ../rulesets
cp -R ../rulesets/stable/java/eap8     evals/javaee-to-jakarta-handcrafted/rules
cp -R ../rulesets/stable/java/quarkus  evals/javaee-to-quarkus-handcrafted/rules
```

Then:

```
go run ./cmd/compare \
  --a evals/javaee-to-jakarta/rules \
  --b evals/javaee-to-jakarta-handcrafted/rules \
  --name-a ai --name-b handcrafted \
  --app-dir ~/scratch/eval-apps/coolstore \
  --out evals/comparisons/javaee-to-jakarta.md
```

Same shape for quarkus. Both run kantra locally; needs `kantra` on PATH and `~/.kantra/java-external-provider` installed.

## Skipped

- Per-rule unit tests (`cmd/test` invokes `kantra rules test`; that subcommand was removed from kantra v0.10.0-alpha.2 — it's `kantra test` now). Test data exists under `output/<migration>-benchmark/tests/` but wasn't run.
- AI rule dedup. The judge flagged the quarkus AI ruleset has 2-3 overlapping rules per import (broad package + specific class + bare-name) — would cut incidents on coolstore from 214 to ~100 without losing coverage.
