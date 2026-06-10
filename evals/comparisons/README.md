# Comparison runs

Side-by-side numbers for ai-rule-gen vs the hand-authored rulesets at konveyor/rulesets. Generated artifacts in this dir:

- `javaee-to-jakarta.md` / `javaee-to-quarkus.md` — `cmd/compare` output (coverage matrix + kantra-analyze diff on coolstore)
- `javaee-to-jakarta-judge.md` / `javaee-to-quarkus-judge.md` — LLM judge writeups (head-to-head, sampled rules)

## Setup

| Migration | AI source | Handcrafted source |
| --- | --- | --- |
| javaee → jakarta | Red Hat EAP 8 Migration Guide PDF | `stable/java/eap8` |
| javaee → quarkus | OpenRewrite recipe docs (4 pages concatenated) | `stable/java/quarkus` |

Sample app for the kantra diff: `konveyor-ecosystem/coolstore` @ b5752b9.

## Numbers (rerun)

|                                | jakarta AI | jakarta handcrafted | quarkus AI | quarkus handcrafted |
| ------------------------------ | ---------- | ------------------- | ---------- | ------------------- |
| rules                          | 292        | 340                 | 15         | 82                  |
| quality (cmd/eval)             | 5.3 / 6    | 5.1 / 6             | 5.6 / 6    | 5.0 / 6             |
| coolstore — rules fired        | 2          | 13                  | 8          | 25                  |
| coolstore — incidents          | 109        | 118                 | 12         | 55                  |
| coolstore — files only here    | 0          | 3                   | 0          | 6                   |
| coolstore — files in both      | 24         | 24                  | 9          | 9                   |

Coverage matrix totals (per the `.md` reports):

|                                                 | jakarta | quarkus |
| ----------------------------------------------- | ------- | ------- |
| AI rules with handcrafted equivalent            | 63 / 292 | 3 / 15 |
| handcrafted rules with AI equivalent            | 74 / 340 | 4 / 82 |

## Notes

- Round 1 of jakarta missed the `javax → jakarta` import sweep rule. Traced to a regex tolerance in the bespoke PDF-to-markdown step that dropped section 4.1 of the source guide before the section indexer ran. Round 2 (above) re-ran the pipeline with that fixed; AI emitted the sweep rule (`namespace-import-00010`) and coolstore numbers landed close to handcrafted.
- AI sweep pattern is `javax*` (PACKAGE location). Handcrafted enumerates 21 EE roots (`javax.(activation|annotation|batch|...|xml)`) and uses a capture group to template the matched name into the message. AI version will false-positive on Java SE packages and has a static message.
- AI didn't write the companion sweeps for XML namespace URIs, `META-INF/services/javax.*`, or `<property name="javax.*">` rename, all called out in the same guide section. Handcrafted has rules for each.
- quarkus AI ruleset is small because the OpenRewrite recipe docs we used as the source disclaim JMS, JSF, and other coverage outright. Fairer benchmark would feed it the per-feature quarkus.io guides the handcrafted rules cite.
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
- Sample apps other than coolstore (ejb-remote, ejb-security, tasks-qute are cloned at `~/scratch/eval-apps/` but not wired into the comparison loop).
