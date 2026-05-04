# Fix Strategies

Universal guide for interpreting kantra test failures and fixing test data. For language-specific provider behavior, see `languages/<language>/fix-strategies.md`.

## Kantra Output Parsing

Kantra test output contains:
- **Summary line:** `Rules Summary: 5/10 PASSED` — parsed for passed/total counts
- **Per-rule lines:** `<ruleID>  0/1  PASSED  find debug data in /path` — rules showing `0/N PASSED` are failures
- **Debug data path:** Some failures include a path to debug output with `output.yaml` and `rules.yaml`

## Failure Types

### 0 incidents (rule didn't match anything)

**Cause:** The test code doesn't contain code that triggers the rule's pattern.

**Diagnosis steps:**
1. Read the rule YAML — what condition type? (`*.referenced`, `*.dependency`, `builtin.filecontent`, `builtin.xml`)
2. Read the provider reference (`languages/<language>/fix-strategies.md`) for that condition type
3. Check the test source files — does the code actually trigger the pattern?
4. Check the `.test.yaml` — is `mode: source-only` set on a rule that doesn't support it?

**Common causes by condition type:**
- `*.referenced` — test code missing the import/usage, or wrong FQN
- `*.dependency` — dependency missing from build file, wrong version, or `.test.yaml` has `mode: source-only`
- `builtin.filecontent` — text doesn't match the regex, or file doesn't match `filePattern`
- `builtin.xml` — XPath doesn't match, or namespaces mismatch

### 0 incidents for builtin.filecontent (invalid filePattern regex)

**Cause:** The `filePattern` field uses glob syntax instead of valid Go regex.

**Fix:** This is a rule issue. Fix `file_pattern` in patterns.json:
- `*.properties` → `.*\\.properties`
- `*.gradle` → `.*\\.gradle`
- `*.xml` → `.*\\.xml`

Then re-run `go run ./cmd/construct` and re-test.

### Too many incidents (pattern too broad)

**Cause:** The rule pattern matches more code than intended.

**Fix:** Report as a rule quality issue — the pattern may need to be more specific.

### Group error ("unable to get build tool")

**Cause:** The build file (pom.xml, go.mod, etc.) is malformed or missing.

**Fix:** Regenerate the build file. Common issues:
- Invalid XML (illegal `--` in comments — run `go run ./cmd/sanitize`)
- Missing required elements (e.g., `<modelVersion>`, `<groupId>`)
- Wrong file location

### Compilation errors

Test code must compile before kantra can analyze it. See `languages/<language>/fix-strategies.md` for language-specific compilation commands and fix guidance.

Universal rules:
- Fix ONLY the lines mentioned in errors
- Keep ALL rule-triggering code — every import and usage must remain
- Do NOT change library versions in the build file

## Fix Loop Flow

This agent owns the fix-verify loop:

```
for iteration = 1 to max_iterations:
    1. Diagnose each failing rule (read rule YAML + test source)
    2. Read languages/<language>/fix-strategies.md for the relevant condition type
    3. Fix test source files
    4. Verify via: go run ./cmd/test --rules <dir> --tests <dir> --files <failing .test.yaml files>
    5. If all pass: done
    6. If failures remain: next iteration with only remaining failures
```

**Always use `go run ./cmd/test`** to verify — never run `kantra` directly.

## Extracting Pattern Info from Rules

To diagnose failures, extract the condition type and pattern from the failing rule's `when` block:

| Condition | Pattern Source | Provider |
|---|---|---|
| `java.referenced` | `.pattern` | Java (see `languages/java/fix-strategies.md`) |
| `java.dependency` | `.name` | Java (see `languages/java/fix-strategies.md`) |
| `go.referenced` | `.pattern` | Go (see `languages/go/fix-strategies.md`) |
| `go.dependency` | `.name` | Go (see `languages/go/fix-strategies.md`) |
| `nodejs.referenced` | `.pattern` | Node.js (see `languages/nodejs/fix-strategies.md`) |
| `csharp.referenced` | `.pattern` | C# (see `languages/csharp/fix-strategies.md`) |
| `builtin.filecontent` | `.pattern` | Builtin (no provider reference needed) |
| `builtin.xml` | `.xpath` | Builtin (no provider reference needed) |
| `or` / `and` combinator | Recurse into children | From child condition |

## Kantra Analyze (--run-local) Output

When using `kantra analyze --run-local`, results are in `output.yaml`:

```yaml
- name: ruleset-name
  violations:
    rule-00010:
      description: ...
      incidents:
        - uri: file:///path/to/file.go
  unmatched:
    - rule-00030
```

Rules under `violations` with incidents = passed. Rules in `unmatched` or absent = failed.

## Rule Integrity Principle

**NEVER change a rule's condition type, provider_type, location_type, or pattern.** The rule is authoritative. If a test fails:

1. Fix the test data (source code, build file, test YAML config)
2. If still failing after fixing, mark the rule as failed
3. The fix ALWAYS belongs in the test data, not the rule
