---
name: generate-rules
description: Generate Konveyor analyzer migration rules from a migration guide (URL, file path, or pasted text)
---

# Generate Konveyor Migration Rules

Generate Konveyor analyzer migration rules from a migration guide.

**Input:** $ARGUMENTS (URL, file path, or pasted text of a migration guide)

If no argument is provided, ask the user for the migration guide source.

## UX Principles

The user should see a **flowing stream of short status lines** — never silence. Every step prints one line when it starts and one line when it finishes. No walls of text, no tables mid-pipeline, no unnecessary questions.

**Checkpoint after extraction.** After extraction completes, ask the user: "Continue with test generation and validation?" This is the only question in the pipeline. If they say no or say to skip testing, finalize with untested rules.

**Don't ask beyond the checkpoint.** Once testing starts, run the full pipeline (test → validate → fix → finalize) without further questions. If something fails after 3 fix attempts, report it and move on — don't block.

**Only fix what's broken.** When the fix loop runs, only re-generate and re-validate the failing rules, not the entire suite.

## Output Format

Use this format for every status line:

```
[step-name] message
```

Example full run:

```
[ingest] Fetching guide from https://...
[ingest] Done — 3876 lines, 77 sections

[extract] Extracting migration patterns...
[extract] Done — 52 patterns → 52 rules in output/rules/

[test-gen] Generating test data for 52 rules...
[test-gen] Done — 12 groups, 39 files

[validate] Running tests on 12 groups...
[validate] 49/52 passed — fixing 3 failures
[validate] Fix 1/3 — 51/52 passed
[validate] Fix 2/3 — 52/52 passed

[done] 52 rules generated, 52/52 passed — output/rules/
```

That's the entire user-visible output. Everything else happens silently in sub-agents.

## Sub-agent Protocol

This orchestrator uses the **Agent tool** to run sub-agents for heavy LLM work. Each sub-agent:

- Receives a **self-contained prompt** — explicit inputs, context, instructions, and expected return format
- Reads reference files itself (e.g., `agents/rule-writer/references/`) — but the orchestrator tells it which files and why
- Returns **structured JSON** — the orchestrator parses it and prints a status line

**Prompt discipline:** Never say "read the SKILL.md and follow it." Instead, give each sub-agent:
1. What it's doing and why (context)
2. What files to read for reference (with purpose of each)
3. Specific inputs (line ranges, file paths, section headings)
4. Exact return format (JSON schema)

**Single agent for extraction:** Extraction is a well-defined task with clear instructions. One agent processing the full guide is faster than batching across multiple agents (avoids duplicated reference reads and merge overhead).

## Pipeline

### 1. Ingest

```
[ingest] Fetching guide from <source>...
```

```bash
mkdir -p output
```

- **URL:** `go run ./cmd/ingest --input <url> --output output/guide.md`
- **File (not markdown):** `go run ./cmd/ingest --input <path> --output output/guide.md`
- **Pasted text or already markdown:** Write directly to `output/guide.md`

Count lines (`GUIDE_LINES`) and section headings. Print:

```
[ingest] Done — <GUIDE_LINES> lines, <N> sections
```

### 2. Extract

```
[extract] Extracting migration patterns...
```

Spawn a single sub-agent with this self-contained prompt:

```
You are extracting migration patterns from a migration guide for Konveyor analyzer rules.

**Read these references first** (they define valid fields, condition types, and working examples):
- `agents/rule-writer/references/patterns-json-schema.md` — All pattern fields, what the CLI does with them, what counts as extractable, common extraction mistakes
- `agents/rule-writer/references/condition-types.md` — All 12 condition types, when to use each, critical matching rules
- `agents/rule-writer/references/examples/` — Read ALL files. These are working rule examples.

Then read the full migration guide: `output/guide.md`

**Migration context:**
- Source: <source if user specified, otherwise "auto-detect from guide content">
- Target: <target if user specified, otherwise "auto-detect from guide content">
- Language: <language if known, otherwise "auto-detect">

**Workflow:**
1. Auto-detect source/target/language from the guide if not provided
2. Scan the guide and build a section index (all ## / ### / #### headings with line ranges)
3. Process each section: extract patterns or record a skip reason
4. Deduplicate: same source_fqn or dependency_name → keep the more complete pattern
5. Write the complete `output/patterns.json` file

**For each section — extract patterns aggressively:**
1. Read the lead paragraph carefully — the biggest change is often stated first
2. Extract one or more patterns with proper fields (see patterns-json-schema.md)
3. **Do NOT skip sections labeled "informational", "deprecated", or "advisory".** These are the MOST meaningful sections — deprecated starters are detectable dependency renames, advisory sections warn about behavioral changes that affect user code, and informational tables contain concrete old→new mappings.
4. **Every section that mentions ANY of these produces patterns — no exceptions:**
   - Removed feature/library/integration → Detect via *.dependency
   - Deprecated or renamed dependency/starter → Detect old artifact via *.dependency
   - Relocated class/annotation/interface → Detect via *.referenced
   - Old→new mapping table → Each row is a pattern (dependency rename, class relocation, etc.)
   - Behavioral default change → Detect the affected artifact as a `potential` pattern
   - Advisory to add/change a dependency → Detect the old state via *.dependency
5. Only skip a section if it has **zero code artifacts** — no classes, no dependencies, no properties, no annotations, no config elements, no build file entries. "General advice" without any named artifact is the ONLY valid skip reason.

**Condition type selection** (set the right fields in each pattern):
- API/annotation/import changes → `source_fqn` + `location_type` + `provider_type` (→ *.referenced)
- Dependency removed/renamed → `dependency_name` + `upper_bound` and/or `lower_bound` (→ *.dependency). Version bounds required.
- POM/XML structure → `xpath` + `namespaces` + `xpath_filepaths` (→ builtin.xml)
- Config property renames → `source_fqn` (regex) + `file_pattern` (valid Go regex, NOT glob — `.*\\.properties` not `*.properties`) + `provider_type: builtin` (→ builtin.filecontent)

**Consolidate repetitive patterns:** When multiple properties follow the same rename pattern (e.g., `spring.data.mongodb.*` → `spring.mongodb.*`), combine them into ONE pattern with a regex alternation (e.g., `spring\.data\.mongodb\.(host|port|uri|...)`) instead of creating separate patterns for each property.

**Message:** For each pattern, set the `message` field with 2-4 sentences: what to change, why, and Before/After code examples as markdown code blocks with language highlighting.

**Key rules:**
- One pattern per distinct change (but consolidate repetitive renames into regex)
- Use specific FQNs (e.g., `org.springframework.boot.test.mock.mockito.MockBean` not `*.MockBean`)
- Detect the affected artifact, not the missing fix
- "Removed" ALWAYS means detectable — never skip
- "Deprecated" ALWAYS means detectable — detect the old artifact before it's removed
- "Advisory" sections that name specific artifacts ARE detectable — extract the artifact
- Tables with old→new mappings are patterns, not decoration — process every row
- Minimum required fields per pattern: `source_pattern`, `rationale`, `complexity`, `category`

**Output:** Write `output/patterns.json` directly:
{"source": "<source>", "target": "<target>", "language": "<language>", "patterns": [...]}

Then run:
go run ./cmd/construct --patterns output/patterns.json --output output/rules
go run ./cmd/validate --rules output/rules

If validation fails, fix patterns.json and re-run construct + validate.

**Return ONLY this structured summary (no other text):**

source: <detected-source>
target: <detected-target>
patterns_count: <N>
rules_count: <M>
rules_dir: output/rules
coverage_report:
  sections_processed: <N>
  sections_with_patterns: <M>
  sections_skipped: <K>
  skipped_sections:
    - "<heading>" -- <reason>
    ...
```

Run syntactic validation and print the result:

```bash
go run ./cmd/validate --rules output/rules
```

Print:

```
[extract] Done — <patterns_count> patterns → <rules_count> rules in output/rules/
[extract] Validation: <result from cmd/validate>
```

Save `source`, `target`, `patterns_count`, `rules_count`, and the coverage report for the final summary.

### Checkpoint

After printing the extract summary, ask the user:

```
Continue with test generation and validation? (y/n)
```

If the user declines, skip to Step 6 (Summary) with untested rules. Otherwise continue.

### 3. Test Generation

**3a. Scaffold (orchestrator runs this directly, not an agent):**

```bash
go run ./cmd/scaffold --rules output/rules --output output/tests
```

This creates all directories, `.test.yaml` files, and `manifest.json`. No LLM needed.

**3b. Read manifest and split into batches:**

Read `output/tests/manifest.json`. It contains a `groups` array — each group has `name`, `data_dir`, `files` (paths to generate), and `rule_ids`.

Split the groups into **3 roughly equal batches** by rule count.

```
[test-gen] Generating test data for <rules_count> rules (3 parallel agents)...
```

**3c. Spawn 3 test-generator agents in parallel**, each with a self-contained prompt. Send all 3 Agent calls in a single message so they run concurrently:

```
Read `agents/test-generator/references/test-data-guide.md` for pattern matching rules and project structure.

**Context:** You are generating test source code that triggers Konveyor analyzer migration rules. The scaffold step already created all directories, `.test.yaml` files, and project files (.classpath, .project, .settings). You ONLY need to generate pom.xml and Application.java (or equivalent) for each group.

**Inputs:**
- Rules directory: `output/rules`
- Tests directory: `output/tests`

**Your groups (generate ONLY these, ignore all others):**
<for each group in this batch:>
- Group: <name>
  Data dir: <data_dir>
  Rules: <rule_ids as comma-separated list>
  Files to generate:
    - <path> (purpose: <build|source>)
    - <path> (purpose: <build|source>)

**What to do for each group:**
1. Read the rule YAML files from `output/rules/` to understand each rule's `when` condition
2. Generate the build file (pom.xml) — minimal dependencies, just enough to compile
3. Generate the source file (Application.java) — code that triggers EVERY rule in the group
4. Write both files using the Write tool
5. For `builtin.filecontent` rules: write a config file matching the filePattern regex with content matching the pattern
6. For `builtin.xml` rules: ensure pom.xml contains elements matching the XPath expression

**Critical pattern matching rules:**
- `java.referenced ANNOTATION`: annotation must be APPLIED (`@Ann private Object x;`), import alone is NOT enough
- `java.referenced METHOD_CALL`: call on explicitly typed variable, do NOT chain calls
- `java.dependency`: only needs pom.xml dependency entry, no Java source code needed
- `builtin.filecontent`: write a config file matching the filePattern regex with content matching the pattern
- `builtin.xml`: write an XML file matching the XPath expression

**Important:** Use Read/Write/Glob tools for file operations. Do NOT use Bash for file creation, counting, or directory listing.

**Return ONLY:**
groups_completed: <N>
files_written: <M>
```

**3d. Collect results and sanitize.** After all 3 agents complete:

```bash
go run ./cmd/sanitize --dir output/tests/tests/data
```

```
[test-gen] Done — <total_groups> groups, <total_files> files
```

### 4. Validate (orchestrator-driven loop)

The orchestrator runs tests directly in **batched sequential runs** to avoid OOM (one giant kantra run) and Docker contention (parallel kantra runs). The fix loop uses a sub-agent for LLM-driven repairs.

**4a. Batch and run tests:**

Split the groups from `manifest.json` into **3 roughly equal batches** by rule count (same batching as test-gen). Run each batch sequentially using `--files`:

```
[validate] Running batch 1/3: <group_names> (<N> rules)...
```

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <batch1.test.yaml>,<batch2.test.yaml>,...
```

```
[validate] Running batch 2/3: <group_names> (<N> rules)...
```

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <batch3.test.yaml>,<batch4.test.yaml>,...
```

```
[validate] Running batch 3/3: <group_names> (<N> rules)...
```

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <batch5.test.yaml>,<batch6.test.yaml>,...
```

**Why batched sequential, not parallel agents:** `kantra test` runs Docker containers. Multiple kantra instances running simultaneously cause Docker contention and hangs. Sequential batches keep each kantra run small (avoids OOM) while avoiding contention.

**`--files` takes bare filenames** (e.g., `data-1.test.yaml`), resolved relative to `--tests` dir. The runner scopes results to only the rules referenced by those test files.

Collate results across all batches. Print the combined result:

```
[validate] <total_passed>/<total_rules> passed
```

Or if failures:

```
[validate] <total_passed>/<total_rules> passed — <F> failures: <rule_id_1>, <rule_id_2>, ...
```

If all passed, skip to step 5.

**4b. Fix loop (max 3 iterations, only if failures):**

For each iteration:

Print what's being fixed:

```
[fix <I>/3] Fixing: <rule_id_1>, <rule_id_2> ...
```

Spawn a fix agent with a self-contained prompt listing ONLY the failing rules:

```
Read `agents/rule-validator/SKILL.md` and `agents/rule-validator/references/fix-strategies.md`.

**Context:** These rules failed kantra validation. Fix the test data so the rules pass.

**Inputs:**
- Rules directory: output/rules
- Tests directory: output/tests/tests

**Failing rules to fix:**
<for each failure:>
- Rule ID: <rule_id>
  Test file: <test_file_path>
  Error: <error summary from kantra output>

**What to do:**
1. Read each failing rule YAML to understand its `when` condition
2. Read the test source files for that group
3. Fix the source/build files so the rule triggers correctly
4. Do NOT touch passing test groups

**Return ONLY:**
fixed_rules: [<list>]
fix_details:
  - rule_id: "<id>"
    diagnosis: "<what was wrong>"
    fix: "<what was changed>"
```

After the fix agent returns, re-run ONLY the failing test groups (not the full suite):

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests --files <comma-separated failing .test.yaml filenames>
```

Parse and print:

```
[fix <I>/3] <passed>/<total> passed
```

If all passed, stop. If still failing, next iteration. After 3 iterations, move on with remaining failures.

### 5. Stamp + Report

After all batches and fix loops are done, run a **full test** (no `--files`) to generate the combined `kantra-output.txt` needed for stamping:

```bash
go run ./cmd/test --rules output/rules --tests output/tests/tests
```

Then stamp and report:

```bash
go run ./cmd/stamp --rules output/rules --kantra-output "$(cat output/tests/tests/kantra-output.txt)"
go run ./cmd/report --source <source> --target <target> --output output/report.yaml --rules-total <N> --passed <P> --failed <F> --failed-rules <comma-separated>
```

### 6. Summary

Print a formatted summary table using GitHub-flavored markdown:

```markdown
## Summary

| | |
|---|---|
| **Input** | <guide title or URL as a markdown link> |
| **Migration** | <source> → <target> (<language>) |
| **Guide** | <GUIDE_LINES> lines, <N> sections → <M> produced patterns, <K> skipped |
| **Rules** | <rules_count> generated, **<P>/<N> passed (<percent>%)** |
| **Fix iterations** | <iterations used, 0 if none> |
| **Output** | `output/rules/` (rules), `output/tests/` (tests), `output/report.yaml` (report) |

### Rule Categories

| Group | Rules | Status |
|---|---|---|
| <group_name> (<brief description of what rules cover>) | <rule_count> | <passed>/<total> passed |
| ... | ... | ... |
```

If there are failures, add a row to the top-level table:

```
| **Failed** | <rule_id_1>, <rule_id_2>, ... |
```

If coverage was low (< 30% of sections produced patterns), add a row:

```
| **Warning** | Low extraction coverage (<M>/<K> sections) |
```
