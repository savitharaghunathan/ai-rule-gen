# Judge: ai-generated vs handcrafted (javaee → jakarta, round 2)

This verdict re-evaluates the AI-generated ruleset after the PDF-normalizer fix that
restored section 4.1 ("THE JAVAX TO JAKARTA PACKAGE NAMESPACE CHANGE") of the EAP 8
migration guide to the section index. Round 1 found that section was silently
dropped, the AI never saw it, and the load-bearing javax→jakarta package sweep rule
was missing. This round, the AI did generate the sweep rule — `namespace-import-00010`
in `evals/javaee-to-jakarta/rules/namespace.yaml`.

## Verdict

The AI is back to first-author quality on the **headline** rule of this migration.
Round 1's catastrophic gap is closed: `namespace-import-00010` fires on every coolstore
file that the handcrafted `javax-to-jakarta-import-00001` does, and the two rulesets
now flag the same 24 Java files and same `pom.xml`. The AI ruleset still trails the
handcrafted one in **breadth of artifacts covered for the same migration step** — it
catches the Java imports and `pom.xml` dependency groupId, but misses XML
namespace/schema sweeps for descriptors (`web.xml`, `beans.xml`, `persistence.xml`,
`faces-config.xml`), `META-INF/services/javax.*` bootstrapping file renames, and the
`<property name="javax.*">` rename in deployment descriptors. None of those gaps
showed up as missed coolstore incidents because coolstore's descriptors happen to be
generic enough not to carry the `xmlns.jcp.org` namespace, but they are real coverage
gaps against the section 4.1 bullet list.

Overall winner: **handcrafted**, but by a much smaller margin than round 1 — the AI
now does the same *job* on real code, just less thoroughly and with a single coarse
incident per file rather than per-line guidance.

## The sweep rule: AI vs handcrafted

`namespace-import-00010` (AI, `evals/javaee-to-jakarta/rules/namespace.yaml:1-40`):

```yaml
when:
  java.referenced:
    pattern: javax*
    location: PACKAGE
```

`javax-to-jakarta-import-00001` (handcrafted,
`evals/javaee-to-jakarta-handcrafted/rules/164-javax-to-jakarta-package.windup.yaml`):

```yaml
customVariables:
- name: renamed
  nameOfCaptureGroup: renamed
  pattern: javax.(?P<renamed>(activation|annotation|batch|decorator|ejb|el|enterprise|faces|inject|interceptor|jms|json|jws|mail|persistence|resource|security|servlet|transaction|validation|websocket|ws|xml))?.*
message: Replace the `javax.{{renamed}}` import statement with `jakarta.{{renamed}}`
when:
  java.referenced:
    location: IMPORT
    pattern: javax.(activation|annotation|batch|decorator|ejb|el|enterprise|faces|inject|interceptor|jms|json|jws|mail|persistence|resource|security|servlet|transaction|validation|websocket|ws|xml)*
```

Same *intent*, three meaningful differences:

1. **Pattern shape.** AI uses `javax*` + `location: PACKAGE`. Handcrafted enumerates
   the 21 Jakarta EE top-level packages explicitly and matches on `location: IMPORT`.
   The AI rule is broader on both axes (any package starting with `javax`, any
   reference site that resolves to a package).

2. **Message quality.** Handcrafted captures the second segment with a regex named
   group and renders a dynamic per-incident message ("Replace the `javax.ejb` import
   with `jakarta.ejb`"). The AI rule emits a single static markdown blurb that
   references three example imports and does not name the actual offending package.
   That is the rule-author craft the AI is closest to but still misses: even with
   correct shape, the message is generic where the handcrafted one is specific.

3. **Effort.** AI: `5`. Handcrafted: `1`. Per-incident the handcrafted value is
   correct — this is a mechanical rename, and the migration toolkit / Eclipse
   Transformer will do it automatically per section 4.1. The AI's `effort: 5` will
   inflate the project-wide story-point estimate considerably (109 incidents × 5 vs.
   the same 109 × 1 = a ~440-point delta on coolstore alone).

## What the AI still misses

The handcrafted ruleset treats "section 4.1" as a small *family* of rules. The
guide enumerates four migration actions (imports/source code, system properties,
configuration properties, `META-INF/services` resources). Handcrafted has a
dedicated rule per artifact type; the AI only has the import one.

Concretely, the AI has no equivalent of:

- `javaee-to-jakarta-namespaces-00001..00056` (handcrafted file 161) — sweep of
  XML namespace URIs (`http://xmlns.jcp.org/xml/ns/javaee` → `https://jakarta.ee/xml/ns/jakartaee`),
  every Java EE XSD filename, and every schema `version=` attribute in 56 descriptor
  variants. The AI ruleset's `jaxb-pattern-00010` covers the JAXB binding namespace
  only — it does not touch the JCP javaee/persistence/validation namespaces.
- `javax-to-jakarta-dependencies-00001..00008` (handcrafted file 163) — pom.xml
  `groupId>javax.*<` and `artifactId>javax.*-api<` sweeps. The AI's
  `dependencies-dependency-*` rules in `dependencies.yaml` enumerate ~26 specific
  Maven coordinates one-by-one via `java.dependency`. That works for known artifacts
  but does not catch arbitrary `<groupId>javax.foo</groupId>` text in a `pom.xml`.
- `javax-to-jakarta-bootstrapping-files-00001` (handcrafted file 162) — `builtin.file`
  match on `javax.enterprise.*` filenames for the `META-INF/services` rename
  explicitly called out in section 4.1.
- `javax-to-jakarta-properties-00001` (handcrafted file 165) — `<property
  name="javax."` sweep in any XML, for the EE-specified system/configuration
  property rename also called out in section 4.1.

The coverage matrix (`evals/comparisons/javaee-to-jakarta.md` lines 10-13) reports
that 196 handcrafted rules have no AI equivalent at all and another 70 are only
partial. The bulk of the 196 is the hibernate-search property explosion (which is
arguably overkill on the handcrafted side, see lines 504-624) and a long tail of
EAP-8 migration guide topics the AI guide either didn't cover or covered with a
single broader rule. The four bullets above are the ones that matter for the
*section 4.1 fidelity* question.

## Coverage on coolstore (revisited)

`evals/comparisons/javaee-to-jakarta.md` lines 861-902:

| | ai-generated | handcrafted |
|---|---|---|
| Rules fired | 2 | 13 |
| Incidents | 109 | 118 |
| Files only here | 0 | 3 |
| Files in common | 24 | 24 |

On the 24 shared files the AI fires `namespace-import-00010` once per file and
`dependencies-dependency-00110` on `pom.xml`. The handcrafted ruleset fires the
sweep import rule plus dependency, faces, hibernate, namespace, and PicketLink
rules — 13 distinct rule IDs covering the same files plus three more
(`persistence.xml`, `beans.xml`, `keycloak.json`) that the AI misses entirely.

What this means for a developer:

- The AI ruleset says **"these 24 files need work; here is one paragraph of generic
  advice."** Each file gets a single incident, the message is the same prose every
  time, and a developer triaging in Kantra has to read source to figure out which
  `javax.*` symbol triggered it. Effort total: 109 × 5 = 545 story points.
- The handcrafted ruleset says **"these 27 files need work; here are 13 specific
  changes."** Each incident names the offending package or descriptor element and
  proposes the exact replacement. The three extra files surface `persistence.xml`
  (`hibernate-00005` empty `beans.xml` discovery mode change),
  `keycloak.json` (Keycloak OIDC rename), and `beans.xml`. These are real CDI 4.0 /
  Elytron OIDC migration steps in section 4.2 of the same guide. The AI has rules
  for CDI 4.0 empty `beans.xml` (`cdi-xml-00010`, `cdi-xml-00020`) and Keycloak
  (`security-xml-00010`) — they exist but did not fire because their `xpath`
  predicates are stricter than the handcrafted ones (e.g. requiring `@version`
  attribute or `text()='KEYCLOAK'`). That is a separate AI-side bug class:
  over-specified XPath conditions that miss real-world configs.

Effort total handcrafted: 118 × 1 = 118 points, plus the qualitative win of
per-incident specificity. Net: same files caught, but the handcrafted output is
~5× cheaper to estimate and substantially more actionable.

## False positive risk

`namespace-import-00010` uses `pattern: javax*` with `location: PACKAGE`. This will
match Java SE `javax.*` packages — `javax.crypto`, `javax.sql`, `javax.naming`,
`javax.xml.parsers`, `javax.management`, `javax.net`, `javax.swing`, `javax.print`,
`javax.imageio`, `javax.script`, `javax.tools` — any of which can legitimately
appear in an EAP application and must NOT be renamed. The rule's message
acknowledges this risk explicitly ("Java SE packages ... are NOT renamed") and asks
the developer to filter visually.

The handcrafted rule sidesteps this by enumerating the 21 Jakarta EE package roots
in both the `customVariables` regex and the `pattern`. It will never match
`javax.sql.DataSource` or `javax.crypto.Cipher`.

This is the single biggest quality gap between the two sweep rules. On coolstore it
happens not to matter — coolstore does not import `javax.sql` or `javax.crypto` —
but on a more realistic EAP application this AI rule will produce false positives
that a developer has to manually triage. Recommendation: either narrow the AI
pattern to the same 21-package enumeration, or wrap `javax*` with a `not` clause
excluding the SE roots.

A secondary risk: `location: PACKAGE` matches *any* code-level reference to a
package, not just `import` statements. Fully-qualified usages (`new
javax.ejb.EJBException(...)`) are caught, which is good; but so are package-info
references and `package javax.foo;` declarations in user code, which is rare but
possible.

## Recommendations

1. **Narrow the AI sweep rule's `pattern` to the explicit list** of 21 Jakarta EE
   package roots, matching the handcrafted regex. This eliminates the SE-package
   false-positive risk and aligns with the migration tooling's actual scope.
2. **Lower `effort` from 5 to 1.** Per section 4.1, this is a mechanical rename
   handled by the Red Hat Migration Toolkit / Eclipse Transformer. The handcrafted
   value of 1 is calibrated correctly; 5 inflates Kantra story-point estimates ~5×.
3. **Add a capture group + dynamic message.** The handcrafted
   `{{renamed}}` interpolation is the difference between "you have a generic javax
   import problem" and "replace `javax.ejb` with `jakarta.ejb` on line N." This is
   purely a templating change to the existing rule.
4. **Add three companion sweep rules** that the AI's guide-extraction pipeline
   appears to have missed even with the normalizer fix:
   - XML namespace URI: `http://xmlns.jcp.org/xml/ns/javaee` →
     `https://jakarta.ee/xml/ns/jakartaee` (and the `persistence`, `validation`
     siblings). The AI has these for JAXB only.
   - `META-INF/services/javax.*` file rename (a `builtin.file` rule).
   - `<property name="javax.*">` rename in descriptors (a `builtin.filecontent`
     rule).
   All three are explicit bullets in the section 4.1 prose
   (`output/javaee-to-jakarta-benchmark/guide.md` lines 419-428) but are not
   represented in any AI rule. Worth investigating why the pipeline still drops
   them — possibly the action-bullet extractor isn't promoting bullets to rules
   when the parent heading already produced one.
5. **Investigate the over-specified XPath problem** that kept `cdi-xml-00010`,
   `cdi-xml-00020`, and `security-xml-00010` from firing on coolstore's
   `beans.xml`, `persistence.xml`, and `keycloak.json` even though equivalent
   handcrafted rules did. This is a separate failure mode from the round-1 bug:
   the AI is generating *correct* rules but with predicates that are too tight for
   real-world files.

## Comparison to the round-1 verdict

Round 1 (operating on a guide where section 4.1 had been dropped) reportedly
concluded that the AI ruleset was unfit for the headline migration step — it
caught small, peripheral changes but missed the one rule every developer actually
needs. That verdict was correct given the inputs but was diagnosing a *pipeline
data-loss bug*, not a rule-authoring deficit.

Round 2, with the normalizer fix, demonstrates the AI **can** author the sweep
rule when it sees the source material — `namespace-import-00010` is structurally
the right rule, fires on the right files, and ships in the right ruleset
(`namespace.yaml`). The remaining quality gap is editorial (message templating,
effort calibration, false-positive narrowing) and a thinner-than-ideal expansion
of the section's four bullets into four rules instead of one. Those are normal
ruleset-author-tuning concerns, not the existential round-1 problem of "the rule
isn't there."

In short: the round-1 verdict overstated the AI's authoring deficiency by
attributing a data bug to the model. With section 4.1 in hand, the AI delivers
a real, fires-on-the-right-files sweep rule — just one with rough edges a human
reviewer would smooth in a PR review.

{"report_path": "evals/comparisons/javaee-to-jakarta-judge.md", "overall_winner": "handcrafted", "key_finding": "AI sweep rule namespace-import-00010 is structurally correct and catches the same coolstore files as handcrafted javax-to-jakarta-import-00001, but lags on message specificity (no capture-group templating), effort calibration (5 vs 1), false-positive risk (javax* matches Java SE packages), and the surrounding XML/services/properties companion sweeps explicitly called out in guide section 4.1."}
