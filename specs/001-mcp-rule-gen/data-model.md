# Data Model: Phase 1 MCP Server for AI-Powered Rule Generation

**Date**: 2026-03-19 | **Plan**: [plan.md](plan.md)

## Core Entities

### Rule

A single Konveyor analyzer rule. YAML-serializable, parseable by analyzer-lsp.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ruleID` | string | Yes | Unique identifier (no newlines or semicolons) |
| `description` | string | No | Short problem description |
| `category` | enum | No | `mandatory`, `optional`, `potential` |
| `effort` | int | No | Migration effort (1-10) |
| `labels` | []string | No | Labels (`konveyor.io/source=X`, `konveyor.io/target=Y`) |
| `message` | string | Conditional | Violation message with Before/After examples (required if no `tag`) |
| `links` | []Link | No | Reference documentation links |
| `when` | Condition | Yes | Provider condition or combinator |
| `customVariables` | []CustomVariable | No | Variables extracted from matched code |
| `tag` | []string | Conditional | Tags to create (required if no `message`) |

**Validation rules**: Either `message` or `tag` must be set. `ruleID` must not contain newlines or semicolons. `category` must be one of the enum values. Effort typically 1-10. Labels should follow `konveyor.io/source=` and `konveyor.io/target=` format. Regex patterns in conditions must be syntactically valid.

### Ruleset

Metadata for a collection of rules.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Ruleset name |
| `description` | string | No | Ruleset description |
| `labels` | []string | No | Shared labels for the collection |
| `tags` | []string | No | Shared tags |

### Condition

Provider-specific condition for the `when` field. Represented as `map[string]interface{}` for YAML flexibility.

#### java.referenced

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | Fully qualified name or regex |
| `location` | string | No | One of: TYPE, INHERITANCE, METHOD_CALL, CONSTRUCTOR_CALL, ANNOTATION, IMPLEMENTS_TYPE, ENUM, RETURN_TYPE, IMPORT, VARIABLE_DECLARATION, PACKAGE, FIELD, METHOD, CLASS |
| `annotated` | object | No | Annotation filter: `pattern` (string) + `elements` ([]{ name, value }) |
| `filepaths` | []string | No | Scope to specific files |

#### java.dependency / go.dependency

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Dependency name |
| `name_regex` | string | No | Regex pattern for name matching |
| `upperbound` | string | No | Max version (inclusive) |
| `lowerbound` | string | No | Min version (inclusive) |

#### go.referenced / nodejs.referenced

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | Symbol pattern to match |

#### csharp.referenced

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | FQDN or regex pattern |
| `location` | string | No | One of: ALL (default), METHOD, FIELD, CLASS |

#### builtin.filecontent

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | Regex to match in file content |
| `filePattern` | string | No | File name regex filter |
| `filepaths` | []string | No | Scope to specific files |

#### builtin.file

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | File name pattern |

#### builtin.xml

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `xpath` | string | Yes | XPath expression |
| `namespaces` | map[string]string | No | Namespace bindings |
| `filepaths` | []string | No | Scope to specific files |

#### builtin.json

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `xpath` | string | Yes | XPath expression |
| `filepaths` | []string | No | Scope to specific files |

#### builtin.hasTags

Type: `[]string` — list of tag patterns to check.

#### builtin.xmlPublicID

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `regex` | string | Yes | DOCTYPE regex |
| `namespaces` | map[string]string | No | Namespace bindings |
| `filepaths` | []string | No | Scope to specific files |

#### Combinators (and / or)

Array of conditions. Each entry is a condition with optional chaining fields.

#### Chaining Fields (all conditions)

| Field | Type | Description |
|-------|------|-------------|
| `from` | string | Reference output from a named condition |
| `as` | string | Name this condition's output for later use |
| `ignore` | bool | Exclude from rule match determination |
| `not` | bool | Negate the condition result |

### Link

| Field | Type | Required |
|-------|------|----------|
| `url` | string | Yes |
| `title` | string | Yes |

### CustomVariable

| Field | Type | Required |
|-------|------|----------|
| `pattern` | string (regex) | Yes |
| `name` | string | Yes |
| `defaultValue` | string | No |
| `nameOfCaptureGroup` | string | No |

### MigrationPattern

Intermediate type from LLM extraction. Bridges ingested content and rule generation.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source_pattern` | string | Yes | What to detect (e.g., `@Stateless`) |
| `target_pattern` | string | No | Replacement (null for removals) |
| `source_fqn` | string | No | Fully qualified name for `when` condition |
| `location_type` | string | No | Provider location type |
| `alternative_fqns` | []string | No | Alternative FQNs for `or` conditions |
| `rationale` | string | Yes | Why this change is needed |
| `complexity` | string | Yes | trivial, low, medium, high, expert |
| `category` | string | Yes | mandatory, optional, potential |
| `concern` | string | No | Grouping key (e.g., "security", "web") |
| `provider_type` | string | No | java, go, nodejs, csharp, builtin, combo |
| `file_pattern` | string | No | For builtin.filecontent |
| `example_before` | string | No | Code before migration |
| `example_after` | string | No | Code after migration |
| `documentation_url` | string | No | Reference link |

### ConfidenceResult

Per-rule scoring output from LLM-as-judge.

| Field | Type | Description |
|-------|------|-------------|
| `ruleID` | string | Rule being scored |
| `scores` | map[string]int | Criterion scores (1-5): pattern_correctness, message_quality, category_appropriateness, effort_accuracy, false_positive_risk |
| `overall` | float64 | Average of criterion scores |
| `verdict` | string | accept (>=4.0), review (>=2.5), reject (<2.5) |
| `evidence` | []string | Specific citations for each deduction |

### TestsFile

kantra test definition. Imported from `kantra/pkg/testing`.

| Field | Type | Description |
|-------|------|-------------|
| `rulesPath` | string | Relative path to rules YAML |
| `providers` | []ProviderConfig | Provider name + dataPath pairs |
| `tests` | []Test | Per-rule test definitions |

### Test / TestCase

| Field | Type | Description |
|-------|------|-------------|
| `ruleID` | string | Rule to test |
| `testCases` | []TestCase | Test case definitions |
| `name` | string | Test case name |
| `hasIncidents` | IncidentVerification | Expected incident counts or locations |

## Entity Relationships

```
MigrationPattern (extracted by LLM)
    └──> Rule (generated, one or more per pattern)
            ├── Condition (when block)
            ├── Link[] (reference docs)
            └── CustomVariable[] (capture groups)

Ruleset (metadata)
    └── labels shared by Rules

Rule[]
    └──> TestsFile (scaffolded from rules)
            └── Test[] (one per rule)
                └── TestCase[] (incidents to verify)

Rule[]
    └──> ConfidenceResult[] (one per rule, scored independently)
```

## Output Directory Structure

```
output_path/
├── rules/
│   ├── ruleset.yaml              # Ruleset metadata
│   ├── <concern-1>.yaml          # Rules grouped by concern
│   └── <concern-2>.yaml
├── tests/
│   ├── <concern-1>.test.yaml     # Test definitions
│   ├── <concern-2>.test.yaml
│   └── data/
│       ├── <concern-1>/          # Test project (pom.xml + source + config)
│       └── <concern-2>/
└── confidence/
    └── scores.yaml               # Per-rule confidence scores
```
