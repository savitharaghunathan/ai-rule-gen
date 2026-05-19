# Go Extraction Checklist

Language-specific extraction guidance for Go migration rules. Items not listed here follow the universal guidance in SKILL.md.

Go has **no location filtering** — `go.referenced` matches any reference to the import path or symbol. There are no `METHOD_CALL`, `CONSTRUCTOR_CALL`, `ANNOTATION`, or other fine-grained location types.

## Checklist item extraction details

### Item 2: Type or symbol removed or relocated

Use `go.referenced` with the full import path + optional symbol name:
- Package relocated: `golang.org/x/crypto/md4` (matches any import of this package)
- Specific symbol: `net/http.Client` (matches usage of `Client` from `net/http`)

No location filtering is needed or available — `go.referenced` matches any reference site (import, usage, type declaration).

### Item 4: Reference table with old→new mappings

Process every row as a separate `go.referenced` pattern. For module-wide renames where all symbols keep the same names, emit ONE pattern on the old module path — additional patterns are only needed for symbols whose names changed. See Module-level consolidation below.

### Item 7: Names any specific artifact

| Artifact type | Condition | Pattern example |
|---|---|---|
| Package import | `go.referenced` | `golang.org/x/crypto/md4` |
| Specific function/type | `go.referenced` | `crypto/md5.New` |
| Module dependency | `go.dependency` | `name: golang.org/x/crypto` |
| Config file entry | `builtin.filecontent` | Pattern on config file content |

### Item 9: Before/after code examples

Each API difference produces a `go.referenced` pattern. There is no location type distinction — all patterns match any reference site.

| Diff category | Condition | Pattern |
|---|---|---|
| Function rename | `go.referenced` | `old/pkg.OldFunc` |
| Type rename | `go.referenced` | `old/pkg.OldType` |
| Package relocation | `go.referenced` | `old/import/path` |
| Constant moved | `go.referenced` | `old/pkg.CONSTANT` |

## TABLE output format

When a section contains a reference table, enumerate every row:

```
Table: "<section heading>" (<N> rows)
Row 1: old/pkg.OldType → new/pkg.NewType — EXTRACT (type renamed)
Row 2: old/pkg.SameType → new/pkg.SameType — module-path covers (same name, just moved)
Row 3: old/pkg.OldFunc() → new/pkg.NewFunc() — EXTRACT (function renamed)
Row 4: old/pkg.CONST → new/pkg.CONST — module-path covers (same name)
...
```

Every row must appear. This prevents silent drops.

## CODE-DIFF annotation guidance

When annotating code diffs, there is no location type to specify — all Go patterns use `go.referenced`:

```
Code diff: "## Migration steps" (source example vs target example)
  old/pkg.SomeType → new/pkg.SomeType — EXTRACT (type relocated, detect via go.referenced)
  old/pkg.Func() → new/pkg.NewFunc() — EXTRACT (function renamed, detect via go.referenced)
  old/import/path → new/import/path — EXTRACT (package relocated, detect via go.referenced)
```

## Module-level consolidation

When a migration guide says an entire Go module is replaced:

**How to recognize a module rename:**
- The guide says "replace import path," "module moved to," or similar
- A table lists old→new symbol mappings where every old symbol shares the same module path prefix

**Decision tree:**

| Scenario | Correct output |
|---|---|
| Module `old/path` replaced by `new/path`, all symbols kept same names | ONE `go.referenced` pattern on `old/path` |
| Module replaced + some symbols renamed | ONE module-path pattern + ONE pattern per renamed symbol |
| Symbols moved to different modules | Separate per-symbol patterns |

## Dependency detection

Use `go.dependency` with module path and version bounds:

```yaml
when:
  go.dependency:
    name: golang.org/x/crypto
    upperbound: v0.5.0
```

See `instructions.md` for registry pre-check status (not yet implemented for Go).

## Skip reason validation (Go-specific)

"Covered by module path pattern" is **NEVER** valid for skipping a symbol rename. If `old/pkg.Foo` became `new/pkg.Bar`, the module path pattern catches the import but the user needs to know `Foo` is now `Bar`. Each renamed symbol needs its own pattern.
