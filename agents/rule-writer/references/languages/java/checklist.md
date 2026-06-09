# Java Extraction Checklist

Language-specific extraction guidance for Java migration rules. Items not listed here follow the universal guidance in SKILL.md.

## Checklist item extraction details

### Item 2: Class, annotation, or interface removed or relocated

Use `java.referenced` with the appropriate location type:

| Change type | Location type | Example |
|---|---|---|
| Class/interface relocated | `IMPORT` | `org.apache.http.HttpResponse` → `org.apache.hc.core5.http.ClassicHttpResponse` |
| Annotation relocated | `ANNOTATION` | `@javax.ejb.Stateless` → `@jakarta.ejb.Stateless` |
| Superclass changed | `INHERITANCE` | `extends HttpServlet` where `HttpServlet` moved packages |
| Interface changed | `IMPLEMENTS_TYPE` | `implements SessionBean` where `SessionBean` moved |

### Item 4: Reference table with old→new mappings

Process every row as a separate pattern — **unless** the section describes a package-level rename (see Package-level consolidation below), in which case emit ONE `PACKAGE` rule and only create additional rules for rows where the class name changed or the method name or signature genuinely changed.

For each row, decompose: check the class/type name first, then the method/member name. Assign the correct location type:

| Row change | Condition | Location type |
|---|---|---|
| Class kept same name, just moved packages | `PACKAGE` rule covers this | Skip (PACKAGE covers) |
| Class renamed (different name in new package) | `java.referenced` | `IMPORT` |
| Class replaced with different API | `java.referenced` | `IMPORT` |
| Method name changed | `java.referenced` | `METHOD_CALL` |
| Method signature changed (same name) | `java.referenced` | `METHOD_CALL` |
| Constructor changed | `java.referenced` | `CONSTRUCTOR_CALL` |

### Item 7: Names any specific artifact

Choose location type based on how the artifact appears in code:

| Artifact type | Location type |
|---|---|
| Class/interface used via import | `IMPORT` |
| Annotation usage (`@Foo`) | `ANNOTATION` |
| Method invocation | `METHOD_CALL` |
| `new` expression | `CONSTRUCTOR_CALL` |
| Field declared with this type | `FIELD` |
| Entire package reference | `PACKAGE` |
| Enum type or constant | `ENUM` |
| Superclass | `INHERITANCE` |
| Implemented interface | `IMPLEMENTS_TYPE` |
| JVM flag, CLI option, system property | `builtin.filecontent` (in startup scripts, Dockerfiles, CI configs) |

See `condition-types.md` for the full list of 14 location types and their matching behavior.

### Item 9: Before/after code examples

Each API difference maps to a specific location type:

| Diff category | Location type |
|---|---|
| Function/method rename | `METHOD_CALL` |
| Parameter/argument type change | `METHOD_CALL` (signature changed even if name unchanged) |
| Construction change (`new X` → `Builder.create()`) | `CONSTRUCTOR_CALL` |
| Import/package change | `IMPORT` or `PACKAGE` |
| Constant/enum moved | `ENUM` or `builtin.filecontent` |
| Annotation change | `ANNOTATION` |
| Superclass/interface change | `INHERITANCE` / `IMPLEMENTS_TYPE` |

## TABLE output format

When a section contains a reference table, enumerate every row with its disposition:

```
Table: "<section heading>" (<N> rows)
Row 1: OldThing → NewThing — EXTRACT as IMPORT (class renamed)
Row 2: OldThing → OldThing — PACKAGE covers (same name, same API)
Row 3: OldThing.method() → NewThing.method() — EXTRACT as IMPORT (class renamed, method unchanged)
Row 4: OldThing.foo() → OldThing.bar() — EXTRACT as METHOD_CALL (method renamed)
Row 5: new OldThing() → NewThing.create() — EXTRACT as CONSTRUCTOR_CALL (construction API changed)
...
```

Every row must appear. This prevents silent drops.

## CODE-DIFF annotation guidance

When annotating code diffs, specify the Java location type:

```
Code diff: "## Migration steps" (source example vs target example)
  old.SomeClass → new.SomeClass — EXTRACT as IMPORT (class renamed)
  response.getStatusLine() → response.getCode() — EXTRACT as METHOD_CALL (method removed)
  new HttpPost(url) → ClassicRequestBuilder.post(url) — EXTRACT as CONSTRUCTOR_CALL (construction API changed)
  org.apache.http.* → org.apache.hc.* — EXTRACT as PACKAGE (package renamed)
  @OldAnnotation → @NewAnnotation — EXTRACT as ANNOTATION (annotation renamed)
```

## Package-level consolidation (Java-specific location types)

The universal consolidation logic is in SKILL.md. This section maps it to Java location types.

For package renames, the namespace-level rule uses `location_type: PACKAGE`. Additional rules use:

| Consolidation case (from SKILL.md) | Java location type |
|---|---|
| Method/function rename | `METHOD_CALL` — always use FQN patterns (e.g., `org.apache.http.HttpResponse.getStatusLine`). When the method is defined on an interface but called on subtypes, add `alternative_fqns`. Never use bare method names. See `condition-types.md` for the full METHOD_CALL decision framework. |
| Type replacement (different API) | `IMPORT` — emit with `source_fqn` on the old FQN |
| Type renamed | `IMPORT` — emit with `source_fqn` on the old FQN and a message stating the new class name |
| Same name, just moved packages | Skip — `PACKAGE` rule covers this |

See `../examples/java.md` for worked examples including reference tables and METHOD_CALL alongside PACKAGE rules.

## Dependency detection (checklist item 3)

Item 3 asks: *"Does the section mention a dependency that changed scope, was renamed, was replaced by a different artifact, or was removed and replaced by an add-new instruction?"*

**"Add X and remove Y" = dependency replacement.** Migration guides often phrase dependency changes as "add the new dependency and optionally remove the old one" rather than "dependency X was renamed to Y." Both mean the same thing: the old Maven coordinates are replaced by new ones. When the guide says "add `org.example:new-lib`", check whether an old artifact exists that this replaces — if so, create a `java.dependency` rule on the **OLD** artifact's coordinates.

Common Maven coordinate changes to watch for:
- GroupId changed (e.g., `javax.servlet` → `jakarta.servlet`)
- ArtifactId changed (e.g., `old-lib` → `new-lib`)
- Both changed simultaneously (the most common case in major version migrations)

Use `java.dependency` with `name` (groupId.artifactId) and version bounds:

```yaml
when:
  java.dependency:
    name: org.springframework.boot.spring-boot-starter
    upperbound: 4.0.0
```

The `name` field uses dot-separated `groupId.artifactId` format. The rule detects the OLD dependency in `pom.xml`; the migration message tells the user what to replace it with.

See `instructions.md` for Maven Central pre-check requirements and version bound derivation rules.

## Skip reason validation (Java-specific)

The universal skip-reason rules are in SKILL.md. In Java terms, "covered by PACKAGE rule" means the same as "covered by namespace-level rule" — see SKILL.md for when this is valid vs. invalid.
