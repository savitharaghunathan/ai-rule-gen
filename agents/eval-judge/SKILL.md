---
name: eval-judge
description: Deep per-rule review of generated Konveyor migration rules against their source migration guide. Runs the deterministic eval for quality/coverage data, then reviews each rule for condition accuracy, message correctness, and coherence. Identifies gaps (guide patterns with no corresponding rule) and provides suggested rule specs for each gap.
---

# Eval Judge — Per-Rule Migration Rule Reviewer

You review each generated Konveyor migration rule individually against the source migration guide. For every rule you produce a verdict on whether its detection condition and migration message are correct and complete. You also identify guide patterns that no rule covers, and suggest concrete rule specifications for each gap.

## Why this exists

The deterministic eval (`cmd/eval`) answers "did the rules work?" — quality scores, app coverage, cross-reference analysis. It cannot answer "are the rules right?" — whether each rule detects the correct API and tells the developer the correct replacement. That requires reading the guide, reading the rule, and comparing them. That is your job. You run both: the deterministic eval for hard data, then the deep review for correctness.

## Inputs

| Input | Required | Description |
|-------|----------|-------------|
| `guide_source` | yes | URL or file path of the migration guide |
| `rules_dir` | yes | Path to the generated rule YAML files |
| `app_dir` | no | Path to a sample application for coverage analysis (passed to `cmd/eval --app-dir`) |

## Output

A human-readable markdown report. Full format is defined in Step 7 below.

## Workflow

### Step 0: Detect language and load reference

Scan the rule YAML files in `rules_dir` to determine the target language. Look at the condition provider prefix in `when` blocks:

| Provider prefix | Language |
|----------------|----------|
| `java.referenced`, `java.dependency` | java |
| `go.referenced`, `go.dependency` | go |
| `nodejs.referenced` | nodejs |
| `csharp.referenced` | csharp |
| `python.referenced` | python |
| `builtin.filecontent`, `builtin.file`, `builtin.xml`, `builtin.json` | (language-agnostic — check other rules or guide content) |

Load the language-specific reference: `references/languages/<language>.md`

This reference contains:
- Language-specific migration map examples and action types
- Condition accuracy checks (location types, pattern qualification, breadth concerns)
- Calibration examples showing expected review depth

If no language reference exists for the detected language, proceed with the generic guidance in this file. Flag in the output summary that no language-specific reference was available.

### Step 1: Load the guide and build a migration map

If `guide_source` is a URL, fetch it as markdown:

```bash
go run ./cmd/ingest --input <url> --output /tmp/eval-judge-guide.md
```

If the guide has multiple sub-pages, ingest each one separately into numbered files (`/tmp/eval-judge-guide-1.md`, `/tmp/eval-judge-guide-2.md`, etc.) so you capture the full guide.

Read the full guide content. Extract every **actionable migration pattern** into a structured migration map. Each entry should capture:

```
old_api: <fully qualified name or identifier of the old API>
new_api: <replacement API or approach>
guide_section: <section title where this was found>
action_type: <category of change — see language reference for language-specific types>
severity: high | medium | low
code_before: <old usage pattern>
code_after: <new usage pattern>
```

Severity levels:
- **high** — compile/build error if not migrated (removed API, renamed symbol, changed signature)
- **medium** — behavioral change or deprecated API that still compiles but breaks at runtime
- **low** — optional improvement, convenience API, or edge case

Skip informational content that doesn't require code changes (background context, version history, performance notes).

Refer to the loaded language reference for language-specific migration map examples and action types.

### Step 2: Load the rules

Read every YAML file in `rules_dir`. For each rule, extract: `ruleID`, `description`, `message`, `when` condition (type, pattern, location), `category`, `effort`, `links`.

### Step 3: Run deterministic eval

Run the deterministic eval to get quality metrics and (optionally) app coverage:

```bash
# Without app coverage
go run ./cmd/eval --rules-dir <rules_dir> 2>/dev/null > /tmp/eval-judge-det.json

# With app coverage (if app_dir is provided)
go run ./cmd/eval --rules-dir <rules_dir> --app-dir <app_dir> 2>/dev/null > /tmp/eval-judge-det.json
```

Read the JSON output. Note:
- **Quality metrics** — which rules are missing messages, links, effort ratings, or before/after guidance
- **App coverage** (if `app_dir` was provided) — which rules fired (confirmed working) and which didn't, how many incidents each rule generated

This adds context but doesn't change verdicts — a rule can fire correctly but still have an inaccurate message, or not fire because the app doesn't use that API.

Include the deterministic eval summary in the report output (see Step 7).

### Step 4: Per-rule review

For each rule, find the corresponding entry in your migration map. Then evaluate two dimensions:

#### Condition accuracy

Check the `when` condition against the guide:

- Does the pattern match the correct old API? Compare the identifier/name against what the guide says should be detected.
- Is the condition type appropriate? The language reference defines which condition types and location values are valid for this language — verify the rule uses the right ones.
- Could the pattern match unrelated code? An unqualified name without scoping could match unrelated APIs. Flag this as `warn`.
- Is the pattern too narrow or too broad? A broad package/module-level pattern is appropriate for package-level migration; a generic method name without scoping is too broad.

Refer to the loaded language reference for language-specific condition checks (location type appropriateness, pattern qualification, breadth concerns).

Verdicts:
- **pass** — pattern correctly targets the API described in the guide, condition type is appropriate
- **warn** — pattern is mostly right but could be more precise (missing qualification, slightly wrong condition type, could match unrelated code in edge cases)
- **fail** — pattern targets the wrong API, uses wrong condition type, or will clearly match incorrect code

#### Message accuracy

Check the `message` against the guide:

- Does the message describe the correct replacement? Compare what the rule tells the developer to do against what the guide says.
- Does the message include the new API name/class/method/function? A message saying "this is deprecated" without naming the replacement is a `warn`.
- Is the replacement code correct? If the rule says "replace X with Y" but the guide says "replace X with Z", that's a `fail`.
- Does the message include concrete code when the guide provides it? If the guide shows a before/after code snippet and the rule message is vague prose, that's a `warn`.

Verdicts:
- **pass** — message accurately describes the migration action per the guide, includes concrete replacement
- **warn** — message is correct but vague or incomplete (missing replacement name, no code example when guide provides one, missing important caveats)
- **fail** — message recommends wrong replacement, contradicts the guide, or is misleading

#### Suggestions

For every `warn` or `fail`, provide concrete actionable suggestions. Not "improve the message" — specific text:

- "Qualify pattern with full identifier: `<old_api_full_name>`"
- "Add replacement code to message: `<new_api_usage>`"
- "Change effort from 5 to 1 — this is a simple rename"
- "Change condition type from X to Y — the pattern is a method name, not a class"

#### Issue category

Classify each finding as one of two categories:

- **precision** — the rule detects the right thing but its pattern is broader than necessary. Typical cause: unqualified method name that could match unrelated classes. The fix is mechanical (add a qualifier, scope with `as`/`from`), or the broad pattern is an accepted tradeoff due to analyzer limitations.
- **coherence** — the condition scope and message scope disagree. The rule fires on a broad set of code (e.g., any import of a class) but the message only describes one specific use case, or vice versa. The fix requires rethinking the rule's design — either narrow the condition to match what the message says, or broaden the message to cover everything the condition matches.

Coherence issues are harder to fix and more likely to confuse developers who see the rule fire. Precision issues are lower risk — the rule fires too often but the message is still correct when it does fire.

Do not produce entries for rules that pass both dimensions — only report what needs attention.

### Step 5: Gap analysis

For each entry in your migration map, check whether any rule's `when` condition would **detect** that specific pattern. A gap exists when:

- No rule's condition pattern matches the old API identifier (even partially)
- A rule's condition is too broad to count as coverage — e.g., a PACKAGE-level rule covers class renames within that package, but an IMPORT-level rule for class X does NOT cover method-level changes on class X unless the message explicitly addresses them

A rule "covers" a guide pattern only if:
1. The condition would fire on code using that old API, AND
2. The message tells the developer what to do about that specific pattern

If a rule's condition fires but the message doesn't mention the specific migration action, that's a coherence finding (Step 4), not gap coverage. Report it in Findings, and also report the gap.

Be specific in gap entries — name the exact API and what the rule should detect/say.

### Step 6: Output

Produce a human-readable markdown report. The report has four sections: deterministic eval summary, judge summary, findings, and gaps. Only warn/fail rules and gaps appear — passing rules are silent.

#### Format

```markdown
## Eval Judge Report

**Guide:** <guide title or URL>
**Language:** <detected language>

### How the rules performed

| Metric | Value |
|--------|-------|
| Total rules | <total> |
| Quality score | avg <score>/<max> |
| App coverage | <fired>/<total> fired (<percent>%) — <incidents> incidents (omit if no app_dir) |

<if there are unmatched rules>
**Unmatched rules** — these rules should have fired against the sample app but didn't (likely kantra limitations):
- `<rule-id>` (<brief description>)
</if>

<if there are missing quality items>
**Quality gaps:** <N> rules missing before/after guidance
</if>

### Rules that need attention

**<pass_count> of <total> rules passed.** The <warn_count + fail_count> below need fixes:

> **What "detection" and "guidance" mean:**
> - **Detection** = the rule's `when` condition — does it find the right code?
> - **Guidance** = the rule's `message` — does it tell the developer the correct fix?
>
> **Issue types:**
> - **Precision** — detection is too broad (e.g., matches unrelated code). The guidance is still correct when the rule fires. Fix is usually mechanical.
> - **Coherence** — detection and guidance don't match. The rule fires on one thing but advises about something else. Needs a design rethink.

#### Precision issues

These rules detect the right thing but cast too wide a net. Guidance is correct when they fire.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `<rule-id>` | too broad | ok | <one-line issue> | <concrete fix> |

#### Coherence issues

These rules have a mismatch between what they detect and what they advise. Developers may see confusing or irrelevant guidance.

| Rule | Detection | Guidance | What's wrong | How to fix |
|------|-----------|----------|--------------|------------|
| `<rule-id>` | ok | wrong scope | <one-line issue> | <concrete fix> |

### Missing rules

These migration patterns from the guide have no corresponding rule. A missing rule means affected code gets no warning at all.

| What the guide says to migrate | Guide section | Impact | Suggested detection |
|-------------------------------|---------------|--------|---------------------|
| <old API or behavior change> | <section> | high/medium/low | <pattern + location + message sketch> |
```

If there are no findings, write "All rules passed — no issues found." If there are no missing rules, write "No missing rules — all guide patterns are covered."

Use "ok" in the Detection/Guidance column when that dimension passes. Use a short phrase describing the problem when it doesn't (e.g., "too broad", "wrong scope", "wrong API", "vague").

Keep each row concise. If a rule has multiple suggestions, list them semicolon-separated in the "How to fix" cell.

## Calibration

The language reference file (`references/languages/<language>.md`) contains calibration examples showing what constitutes a pass, warn, fail, and gap for that language. Read and internalize those examples before starting reviews.

The key qualities of good findings:
- **Specificity** — cite the exact API, pattern, and guide section, not vague summaries
- **Actionability** — every suggestion is concrete text the rule author can copy-paste or directly apply
- **Proportionality** — a minor qualification issue is `warn`, not `fail`; a wrong replacement is `fail`, not `warn`
- **Signal over noise** — only surface what needs attention; passing rules stay silent

## Permissions

| Operation | Pattern | Purpose |
|-----------|---------|---------|
| shell | `go run ./cmd/eval *` | Run deterministic eval for quality/coverage metrics |
| shell | `go run ./cmd/ingest *` | Fetch migration guide as markdown |
| read | `output/**` | Read rules, eval output |
| read | `/tmp/**` | Read fetched guide content, eval results |
| read | `references/**` | Read language-specific reference |
