# Java Provider — Fix Reference

## Fix lookup — 0 incidents by condition type

| Condition + Location | Fix |
|---|---|
| `java.dependency` | Remove `mode: source-only` from `.test.yaml`; ensure pom.xml declares the dependency with version below `upperBound` |
| `java.referenced` + `ANNOTATION` | Ensure `@Annotation` is applied to a class/field/method — import alone is not enough |
| `java.referenced` + `METHOD_CALL` | Un-chain method calls into explicit typed variables |
| `java.referenced` + `IMPORT` | Add or fix the import statement to match the rule's pattern FQN |
| `java.referenced` + `TYPE` | Declare or use the type (variable, cast, field) |
| `builtin.filecontent` | Ensure file has text matching the regex; check `filePattern` is Go regex not glob |
| `builtin.xml` | Remove `mode: source-only` from `.test.yaml`; ensure XML matches the XPath |
| Group error | Regenerate pom.xml — likely malformed XML (run `cmd/sanitize`) |

## Details

### java.dependency — non-semver version (kantra limitation — classify, do not fix)

Check this FIRST before any other `java.dependency` fix.

If the version declared in the test pom.xml is not plain semver (does not match `^\d+\.\d+\.\d+$`), this is a kantra engine limitation — kantra's dependency provider cannot compare non-semver strings against numeric bounds, producing 0 incidents regardless of whether the artifact is present.

**Examples of non-semver versions:** `2.3-groovy-4.0` (Spock), `6.4.0.Final` (Hibernate), `1.0.0.RELEASE` (old Spring), `2.13.12_1.0` (Scala).

**Action:**
- Classify as `still_failing_kantra_limitation` (see `fix-strategies.md`)
- Do NOT change the version to a synthetic plain-semver string — e.g. `2.3.0` for a Spock artifact does not exist on Maven Central and the test would validate a scenario that can't occur in production
- Do NOT change the rule condition type — `java.dependency` is semantically correct
- Do NOT enter a fix iteration for this rule

### java.dependency — `mode: source-only` is the #1 failure

`java.dependency` rules use Maven resolution, not JDTLS. `mode: source-only` skips Maven entirely → always 0 incidents.

The dependency `name` field uses dot notation: `org.springframework.boot.spring-boot-starter-undertow` → `<groupId>org.springframework.boot</groupId><artifactId>spring-boot-starter-undertow</artifactId>`

If pom.xml looks correct and `mode: source-only` is already removed, run `mvn dependency:resolve -q -B` in the test data directory.

### java.referenced ANNOTATION — must be applied, not just imported

```java
// FAILS: import-only
import org.springframework.boot.test.mock.mockito.MockBean;

// WORKS: annotation applied to a field
@MockBean
private Object mockService;
```

### java.referenced METHOD_CALL — no chained calls

JDTLS in source-only mode can't resolve return types of chained calls.

```java
// FAILS: chained
PropertyMapper map = PropertyMapper.get().alwaysApplyingWhenNonNull();

// WORKS: explicit variable
PropertyMapper mapper = PropertyMapper.get();
PropertyMapper nonNullMapper = mapper.alwaysApplyingWhenNonNull();
```

### builtin.xml — also breaks with source-only

Same as java.dependency — remove `mode: source-only` from `.test.yaml`. Ensure XML has elements matching the XPath, with correct namespaces if specified.

### Compilation fixes

1. Do NOT run `mvn compile` — Kantra parses pom.xml directly and JDTLS resolves source-only
2. Fix ONLY lines mentioned in errors — keep all rule-triggering code
3. Do NOT change library versions
4. Prefer minimal dependencies — JDTLS has limited memory in the kantra container. Use `spring-boot-autoconfigure` instead of `spring-boot-starter-web`
