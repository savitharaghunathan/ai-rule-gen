# Fix Strategies

Guide for interpreting kantra test failures and providing fix guidance to the test-generator.

## Kantra Output Parsing

Kantra test output contains:
- **Summary line:** `Rules Summary: 5/10 PASSED` — parsed for passed/total counts
- **Per-rule lines:** `<ruleID>  0/1  PASSED  find debug data in /path` — rules showing `0/N PASSED` are failures
- **Debug data path:** Some failures include a path to debug output with `output.yaml` and `rules.yaml`

## Failure Types and Fixes

### 0 incidents (rule didn't match anything)

**Cause:** The test code doesn't contain code that triggers the rule's pattern.

**Fix guidance to test-generator:** Generate a single-line code snippet that matches the rule pattern. The snippet must:
1. Be a SINGLE line of code (no newlines)
2. Match the pattern EXACTLY
3. Use realistic syntax for the language
4. Be minimal and simple

**Example snippets by language:**
- Go: `import "golang.org/x/crypto/md4"`
- Go: `var ip net.IP = net.ParseIP("192.168.1.1")`
- TypeScript: `import { Button } from '@patternfly/react-core';`
- Java: `import javax.ejb.Stateless;`

### 0 incidents for ANNOTATION (import but no usage)

**Cause:** The test code imports the annotation class but doesn't actually USE it as an annotation. JDTLS matches `ANNOTATION` location only when the annotation is applied to a class, method, or field — `import` alone does not count.

**Fix guidance:** Add actual annotation usage to the test code:
```java
// FAILS: import-only
import org.springframework.boot.test.mock.mockito.MockBean;

// WORKS: annotation applied to a field
import org.springframework.boot.test.mock.mockito.MockBean;
@MockBean
private Object mockService;
```

### 0 incidents for METHOD_CALL with chained calls

**Cause:** JDTLS in source-only mode cannot resolve return types of chained method calls without the dependency JAR. For example, `Foo.get().bar()` — JDTLS doesn't know the return type of `get()`, so it can't confirm `bar()` is on type `Foo`.

**Fix guidance:** Un-chain the call by assigning to an explicitly typed variable:
```java
// FAILS: chained call
PropertyMapper map = PropertyMapper.get().alwaysApplyingWhenNonNull();

// WORKS: explicit variable
PropertyMapper mapper = PropertyMapper.get();
PropertyMapper nonNullMapper = mapper.alwaysApplyingWhenNonNull();
```

### 0 incidents for builtin.filecontent (filePattern is invalid regex)

**Cause:** The `filePattern` field in the rule uses glob syntax (e.g., `*.properties`) instead of valid Go regex (e.g., `.*\\.properties`). `*` alone is not valid regex — it means "zero or more of the preceding token" and fails when there's no preceding token.

**Fix guidance:** This is a patterns.json issue, not a test data issue. Fix the `file_pattern` field in patterns.json to be valid Go regex:
- `*.properties` → `.*\\.properties`
- `*.gradle` → `.*\\.gradle`
- `*.xml` → `.*\\.xml`

Then re-run `go run ./cmd/construct` and re-test.

### Too many incidents (pattern too broad)

**Cause:** The rule pattern is matching more code than intended — the pattern may be too generic.

**Fix guidance:** This is a rule quality issue, not a test data issue. Report back that the rule pattern may need to be more specific.

### Compilation errors in test code

Test code must compile before kantra can analyze it. Two-phase fix approach:

**Phase A — Compilation fix (up to 5 attempts per iteration):**
1. Run the language-specific compiler to check for errors:
   - Go: `go build ./...`
   - Java: `mvn compile -q -B`
   - Node.js: `npx tsc --noEmit`
   - C#: `dotnet build --no-restore`

2. If compilation fails, fix ONLY the lines mentioned in the errors:
   - Do NOT change code that already compiles
   - Keep ALL rule-triggering code — every import and usage that triggers a rule MUST remain
   - Do NOT change library versions in the build file — fix the code to match the installed version

3. For Go errors: run `go doc <package>` to get actual function signatures from the installed version. Use the EXACT signatures from `go doc`, don't guess.

4. For Java errors: check the failing symbol name and find the correct method/type in the dependency's API.

5. For Node.js errors: check TypeScript error messages for property/member name info (e.g., "Property 'foo' does not exist on type 'Bar'") and use the correct name from the package's type definitions.

6. For C# errors: check dotnet build errors for type/member info (e.g., "'Type' does not contain a definition for 'Member'") and use the correct member name.

7. After fixing, re-resolve dependencies:
   - Go: `go mod tidy` then `go mod vendor`
   - Java: `mvn dependency:resolve -q -B`
   - Node.js: `npm install`
   - C#: `dotnet restore`

**Phase B — Kantra pattern matching (after compilation succeeds):**
1. Run kantra test
2. For each failing rule, generate a fix hint: a single-line code snippet that matches the pattern
3. Inject these hints into the test-generator's context when regenerating
4. Regenerate ONLY failing test groups — preserve passing groups

## Fix Loop Flow

This agent owns the fix-verify loop:

```
for iteration = 1 to max_iterations:
    1. Diagnose each failing rule (read rule YAML + test source)
    2. Fix test source files
    3. Verify via: go run ./cmd/test --rules <dir> --tests <dir> --files <failing .test.yaml files>
    4. If all pass: done
    5. If failures remain: next iteration with only remaining failures
```

**Always use `go run ./cmd/test`** to verify — never run `kantra` directly. The `cmd/test` wrapper handles Docker, output parsing, and stamping.

## Extracting Pattern Info from Rules

To generate fix hints, extract the pattern and provider from the failing rule's `when` condition:

| Condition | Pattern Source | Provider String |
|---|---|---|
| `go.referenced` | `.pattern` | `go.referenced` |
| `java.referenced` | `.pattern` | `java.referenced` |
| `nodejs.referenced` | `.pattern` | `nodejs.referenced` |
| `csharp.referenced` | `.pattern` | `csharp.referenced` |
| `builtin.filecontent` | `.pattern` | `builtin.filecontent` |
| `or` / `and` combinator | Recurse into children, return first match | From child |

## Kantra Analyze (--run-local) Output

When using `kantra analyze --run-local`, results are in `output.yaml`:

```yaml
- name: ruleset-name
  violations:
    rule-00010:
      description: ...
      incidents:
        - uri: file:///path/to/file.go
    rule-00020:
      description: ...
      incidents:
        - uri: file:///path/to/file.go
  unmatched:
    - rule-00030
```

Rules appearing under `violations` with incidents = passed. Rules in `unmatched` or absent = failed.

## Rule Integrity Principle

**NEVER change a rule's condition type, provider_type, location_type, or source_fqn to make a test pass.** The rule definition represents the correct migration pattern. If a test fails:

1. First, fix the test data (un-chain calls, add annotation usage, add dependencies, fix build file)
2. If the test still fails after fixing test data, mark the rule as failed
3. The fix ALWAYS belongs in the test data, not the rule

Example: if a `java.referenced METHOD_CALL` rule fails because JDTLS can't resolve a chained call, un-chain the call in the test data — do NOT downgrade the rule to `builtin.filecontent`.
