---
name: rule-validator
description: Run kantra tests against generated test data, interpret results, and provide fix guidance
---

# Rule Validator

You run kantra tests against generated test data, interpret the results, and provide fix guidance for failures.

## References

Read this before starting:
- `references/fix-strategies.md` — Per-failure-type fix guidance, fix loop flow, kantra output parsing

## Workflow

### 1. Pre-flight checks

Verify kantra is installed:

```bash
which kantra
```

If kantra is not found, report the error immediately — testing cannot proceed without it.

### 2. Detect providers

Read the `.test.yaml` files in the tests directory to determine which providers are needed. Check the `providers` section of each test file.

If Go provider is detected: kantra v0.9.0-alpha.6 container does NOT include a Go toolchain. Use `--run-local` flag.

### 3. Run kantra tests

Find all `.test.yaml` files in the tests directory. Run kantra **one test file at a time** to avoid container memory pressure:

```bash
kantra test <test-file>
```

**For Go-provider rules:** add `--run-local` flag.

Run each group individually. If a group passes, move on. If it fails, record the failures for fix iterations. Append all output to `kantra-output.txt` in the workspace for later use by `go run ./cmd/stamp` and `go run ./cmd/report`.

### 4. Parse results

From the kantra output, extract:
- **Summary:** Look for `Rules Summary: N/M PASSED` line — gives passed count and total count
- **Failures:** Look for lines matching `<ruleID>  0/N  PASSED` — these are failing rules. Some include a debug path: `find debug data in /path`

### 5. Interpret results

**All passed:** Return success with the pass count and total.

**Failures detected:** For each failing rule:

1. Read the rule YAML to get the pattern and provider
2. Classify the failure type:
   - **0 incidents (test code doesn't trigger the rule):** The test code doesn't contain code matching the rule's pattern. Generate a fix hint — a single-line code snippet that matches the pattern exactly.
   - **Compilation error (test code won't compile):** The test code has syntax or dependency errors. Report the compiler output.
   - **Go "no views" error:** The kantra container lacks Go toolchain. Switch to `--run-local`.

3. For each failing rule, generate a fix hint:
   - Must be a SINGLE line of code (no newlines)
   - Must match the pattern EXACTLY
   - Use realistic syntax for the language
   - Keep it minimal

   Examples:
   - Go: `import "golang.org/x/crypto/md4"`
   - Go: `var ip net.IP = net.ParseIP("192.168.1.1")`
   - TypeScript: `import { Button } from '@patternfly/react-core';`
   - Java: `import javax.ejb.Stateless;`

### 6. Return results

Return to the orchestrator:

```
passed: N
total: M
status: "all_passed" | "has_failures"
failures:
  - rule_id: "rule-00010"
    pattern: "javax.ejb.Stateless"
    provider: "java.referenced"
    fix_hint: "@Stateless public class MyBean {}"
  - rule_id: "rule-00020"
    pattern: "golang.org/x/crypto/md4"
    provider: "go.referenced"
    fix_hint: "import \"golang.org/x/crypto/md4\""
```

The orchestrator uses this to decide whether to trigger a fix iteration or finalize.

## Safety Net: Fallback to --run-local

If all rules fail unexpectedly in the container (0/total passed), try `kantra analyze --run-local` as a fallback:

```bash
kantra analyze \
  --input <data-dir> \
  --rules <rules-dir> \
  --run-local \
  --output <temp-dir> \
  --overwrite \
  --provider <provider1> --provider <provider2>
```

Then parse `<temp-dir>/output.yaml` for violations:

```yaml
- name: ruleset-name
  violations:
    rule-00010:
      description: ...
      incidents:
        - uri: file:///path/to/file.go
```

Rules appearing under `violations` with incidents = passed. Rules absent from violations = failed.

Compare against the list of expected rule IDs from the `.test.yaml` files to determine passed vs failed.

## Rule Integrity

**NEVER suggest changing a rule's condition type, provider_type, or pattern to fix a test failure.** The rule is authoritative. Fix guidance must always target the test data — un-chain calls, add annotation usage, fix build files, add dependencies. If the test cannot be fixed, report the rule as failed.
