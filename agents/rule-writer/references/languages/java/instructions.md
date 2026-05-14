# Java-Specific Instructions

## Permissions

| Operation | Pattern | Purpose |
|-----------|---------|---------|
| shell | `curl -s "https://search.maven.org/solrsearch/select*"` | Verify artifact versioning scheme on Maven Central |

## Package Registry Pre-Check for Dependency Patterns

Before emitting any pattern that uses `dependency_name` (which produces a `java.dependency` condition), query Maven Central to verify the artifact's versioning scheme:

```bash
curl -s "https://search.maven.org/solrsearch/select?q=g:%22<groupId>%22+AND+a:%22<artifactId>%22&core=gav&rows=10&wt=json"
```

Parse the `groupId` and `artifactId` from the rule's `dependency_name` field (dot notation — the artifactId is the last hyphen-containing segment, e.g. `org.spockframework.spock-spring` → `g:org.spockframework`, `a:spock-spring`). Parse `.response.docs[].v` for the list of published versions.

**Decision:**

| Finding | Action |
|---|---|
| Plain-semver versions exist (`^\d+\.\d+\.\d+$`) | No flag — proceed normally |
| Only non-semver versions (e.g. `2.3-groovy-4.0`, `6.4.0.Final`) | Flag: `suspected_kantra_limitation: no_plain_semver_version` |
| Artifact not found on Maven Central | Flag: `suspected_kantra_limitation: artifact_not_found` |

**In all cases, still emit `java.dependency`** — it is the correct condition type for dependency detection. Do NOT substitute `builtin.xml` or `builtin.filecontent`. The limitation is in kantra's version comparator, not in the rule design.

Collect flagged patterns in a `suspected_kantra_limitations` list and return it alongside `patterns_count` and `output_file`.

## Source Artifact Resolution

For `java.referenced` patterns, emit `source_artifact` so the deterministic verifier can confirm the FQN exists in the published JAR. This catches hallucinated FQNs before rules are constructed.

**How to determine Maven coordinates:**
1. Read the migration guide for the source framework version (e.g., "migrating from Spring Boot 3.5.x" → version `3.5.0`)
2. Map the FQN's package to the correct Maven artifact. Examples:
   - `org.springframework.boot.BootstrapRegistry` → `org.springframework.boot:spring-boot:3.5.0`
   - `org.springframework.boot.autoconfigure.jackson.Jackson2ObjectMapperBuilderCustomizer` → `org.springframework.boot:spring-boot-autoconfigure:3.5.0`
   - `org.apache.http.client.HttpClient` → `org.apache.httpcomponents:httpclient:4.5.14`
3. Use the **source** version (the version being migrated FROM), not the target version

**Format:**
```json
{
  "source_artifact": {
    "group_id": "org.springframework.boot",
    "artifact_id": "spring-boot",
    "version": "3.5.0"
  }
}
```

## Validation Notes

- Java has 14 valid location types: `TYPE`, `INHERITANCE`, `METHOD_CALL`, `CONSTRUCTOR_CALL`, `ANNOTATION`, `IMPLEMENTS_TYPE`, `ENUM`, `RETURN_TYPE`, `IMPORT`, `VARIABLE_DECLARATION`, `PACKAGE`, `FIELD`, `FIELD_DECLARATION`, `METHOD`, `CLASS`
- Invalid `location_type` is a common validation failure — check against this list
- `FIELD` and `FIELD_DECLARATION` are aliases — both match field declarations by type, NOT static field access
- `java.dependency` requires full analysis (not `source-only` mode) since it uses Maven dependency resolution

## Non-Semver Third-Party Artifacts

Some Java artifacts (Spock, Hibernate, Scala, etc.) never publish plain-semver versions. `java.dependency` is still the correct condition type — do NOT substitute `builtin.xml` or `builtin.filecontent`. Instead, always run the Maven Central pre-check above before emitting a `dependency_name` pattern. If Maven Central confirms no plain-semver version exists, flag the pattern as a suspected kantra limitation. Kantra's version comparator cannot handle non-semver strings; this is an engine limitation, not a rule design problem.
