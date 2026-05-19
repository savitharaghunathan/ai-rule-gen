# C# Extraction Checklist

Language-specific extraction guidance for C# migration rules. Items not listed here follow the universal guidance in SKILL.md.

C# has **limited location filtering** — only 4 values: `ALL` (default, matches any reference), `METHOD` (method declarations/calls), `FIELD`, and `CLASS`. Java-style locations like `ANNOTATION`, `INHERITANCE`, `CONSTRUCTOR_CALL`, `IMPORT`, and `PACKAGE` are **NOT supported**.

## Checklist item extraction details

### Item 2: Type removed or relocated

Use `csharp.referenced` with the fully qualified name and `location_type: ALL`:
- Type relocated: `System.Web.HttpContext` (matches any reference)
- For namespace-wide relocations: use a wildcard pattern `OldNamespace.*` — see Namespace-level consolidation below

There is no `IMPORT`, `ANNOTATION`, or `INHERITANCE` location type for C#. Use `ALL` for all type-level detection.

### Item 4: Reference table with old→new mappings

Process every row as a `csharp.referenced` pattern with `location_type: ALL`. For namespace-wide renames, emit ONE wildcard pattern on the old namespace, then additional patterns only for types whose names changed. See Namespace-level consolidation below.

### Item 7: Names any specific artifact

| Artifact type | Condition | Location type |
|---|---|---|
| Type usage (any reference) | `csharp.referenced` | `ALL` |
| Method call or definition | `csharp.referenced` | `METHOD` |
| Field declaration | `csharp.referenced` | `FIELD` |
| Class declaration | `csharp.referenced` | `CLASS` |
| NuGet dependency | `builtin.xml` on `.csproj` | N/A |
| Config file entry | `builtin.filecontent` | N/A |
| XML config | `builtin.xml` | N/A |

### Item 9: Before/after code examples

Each API difference produces a `csharp.referenced` pattern. Use `ALL` for most cases; use `METHOD` only when the change is specifically about a method signature or call.

| Diff category | Location type |
|---|---|
| Type/class rename | `ALL` |
| Method rename | `METHOD` |
| Constructor change | `ALL` (no CONSTRUCTOR_CALL in C#) |
| Namespace change | `ALL` with wildcard pattern |
| Attribute (annotation) change | `ALL` (no ANNOTATION in C#) |

## TABLE output format

When a section contains a reference table, enumerate every row:

```
Table: "<section heading>" (<N> rows)
Row 1: OldNamespace.OldType → NewNamespace.NewType — EXTRACT as ALL (type renamed)
Row 2: OldNamespace.SameType → NewNamespace.SameType — namespace wildcard covers (same name)
Row 3: OldType.OldMethod() → NewType.NewMethod() — EXTRACT as METHOD (method renamed)
Row 4: OldType.SameMethod() → NewType.SameMethod() — EXTRACT as ALL (type renamed, method unchanged)
...
```

Every row must appear. This prevents silent drops.

## CODE-DIFF annotation guidance

When annotating code diffs, specify the C# location type (ALL or METHOD):

```
Code diff: "## Migration steps" (source example vs target example)
  OldNamespace.SomeClass → NewNamespace.SomeClass — EXTRACT as ALL (type relocated)
  obj.OldMethod() → obj.NewMethod() — EXTRACT as METHOD (method renamed)
  new OldType() → NewType.Create() — EXTRACT as ALL (construction changed, no CONSTRUCTOR_CALL in C#)
  [OldAttribute] → [NewAttribute] — EXTRACT as ALL (attribute changed, no ANNOTATION in C#)
```

## Namespace-level consolidation

When a migration guide says an entire C# namespace is replaced:

**How to recognize a namespace rename:**
- The guide says "namespace changed from," "using directive must change," or similar
- A table lists old→new type mappings where every old type shares the same namespace

**Strategy:** Use `csharp.referenced` with a wildcard pattern `OldNamespace.*` and `location_type: ALL` to match any reference to types in the old namespace.

**Decision tree:**

| Scenario | Correct output |
|---|---|
| Namespace `Old.NS` replaced by `New.NS`, all types kept same names | ONE `csharp.referenced` pattern on `Old.NS.*` with `ALL` |
| Namespace replaced + some types renamed | ONE namespace wildcard pattern + ONE pattern per renamed type |
| Types moved to different namespaces | Separate per-type patterns |

## Dependency detection

`csharp.dependency` is **NOT implemented**. Use `builtin.xml` with XPath on `.csproj` files:

```yaml
when:
  builtin.xml:
    xpath: //*[local-name()='PackageReference' and @Include='OldPackage']
    filepaths:
      - "*.csproj"
```

For version range matching, use `builtin.filecontent`:

```yaml
when:
  builtin.filecontent:
    pattern: 'PackageReference Include="OldPackage" Version="'
    filePattern: .*\.csproj
```

## Skip reason validation (C#-specific)

"Covered by namespace wildcard" is **NEVER** valid for skipping a type rename. If `Old.NS.Foo` became `New.NS.Bar`, the namespace wildcard catches the using directive but the user needs to know `Foo` is now `Bar`. Each renamed type needs its own pattern.
