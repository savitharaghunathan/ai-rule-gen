# Node.js Extraction Checklist

Language-specific extraction guidance for Node.js/TypeScript migration rules. Items not listed here follow the universal guidance in SKILL.md.

Node.js has **no location filtering** — `nodejs.referenced` matches any reference to the package + symbol (import, usage, type annotation). There is also **no package-level wildcard** — each renamed export needs its own pattern.

## Checklist item extraction details

### Item 2: Component or export removed or renamed

Use `nodejs.referenced` with the package + symbol name:
- Component removed: `@patternfly/react-core.Button`
- Module export renamed: `express.Router`
- For framework globals or template syntax not accessible via standard imports: use `builtin.filecontent` instead

No location filtering is needed or available — `nodejs.referenced` matches import and usage equally.

### Item 4: Reference table with old→new mappings

Process every row as a `nodejs.referenced` pattern. Unlike Java or Go, there is **no single-pattern shortcut** for package-wide renames — each renamed export needs its own pattern. Even exports that kept the same name may need patterns if the import path changed (since `nodejs.referenced` includes the package name).

### Item 7: Names any specific artifact

| Artifact type | Condition | Pattern example |
|---|---|---|
| Module export | `nodejs.referenced` | `@patternfly/react-core.Button` |
| Framework global | `builtin.filecontent` | Regex on source files |
| Template syntax | `builtin.filecontent` | Regex on template files |
| npm dependency | `builtin.json` on `package.json` | XPath on dependencies |
| Config file entry | `builtin.filecontent` or `builtin.json` | Pattern on config content |

### Item 9: Before/after code examples

Each API difference produces a `nodejs.referenced` or `builtin.filecontent` pattern. There is no location type distinction.

| Diff category | Condition | Pattern |
|---|---|---|
| Component rename | `nodejs.referenced` | `old-package.OldComponent` |
| Function rename | `nodejs.referenced` | `old-package.oldFunc` |
| Import path change | `nodejs.referenced` | `old-package.ExportName` |
| Config/template change | `builtin.filecontent` | Regex on config files |

## TABLE output format

When a section contains a reference table, enumerate every row:

```
Table: "<section heading>" (<N> rows)
Row 1: OldComponent → NewComponent — EXTRACT (component renamed)
Row 2: old-pkg.method() → new-pkg.method() — EXTRACT (function moved to new package)
Row 3: old-pkg.SameName → new-pkg.SameName — EXTRACT (import path changed, needs own pattern)
...
```

Every row must appear. There is no "PACKAGE covers" equivalent in Node.js — every export needs its own pattern when the package name changes.

## CODE-DIFF annotation guidance

When annotating code diffs, there is no location type to specify — all Node.js patterns use `nodejs.referenced`:

```
Code diff: "## Migration steps" (source example vs target example)
  import { OldName } from 'old-pkg' → import { NewName } from 'new-pkg' — EXTRACT (detect via nodejs.referenced)
  OldComponent.method() → NewComponent.method() — EXTRACT (detect via nodejs.referenced)
  <OldComponent /> → <NewComponent /> — EXTRACT (detect via nodejs.referenced)
```

## Package-level consolidation

Node.js has **no `PACKAGE` location type equivalent**. When a migration guide says an entire npm package is replaced:

**Strategy:** Each renamed/removed export needs its own `nodejs.referenced` pattern. There is no single-pattern shortcut to cover an entire package rename.

**Exception:** If the change is purely a dependency rename (e.g., `@old/pkg` → `@new/pkg`) with all exports unchanged, emit a dependency pattern using `builtin.json` on `package.json` to detect the old package name. Individual export patterns are still needed if the import paths or export names changed.

**Decision tree:**

| Scenario | Correct output |
|---|---|
| Package renamed, all exports kept same names, import paths changed | ONE `builtin.json` dependency pattern + ONE `nodejs.referenced` pattern per export |
| Package renamed + some exports renamed | ONE dependency pattern + ONE `nodejs.referenced` pattern per export (both renamed and same-named) |
| Individual exports moved to different packages | Separate per-export patterns |

## Dependency detection

`nodejs.dependency` is **NOT implemented**. Use `builtin.json` with XPath on `package.json`:

```yaml
when:
  builtin.json:
    xpath: /dependencies/@old-package
    filepaths:
      - package.json
```

Or use `builtin.filecontent` with a regex:

```yaml
when:
  builtin.filecontent:
    pattern: '"@old/package"\s*:\s*"'
    filePattern: package\.json
```

## Skip reason validation (Node.js-specific)

There is **no package-level wildcard** in Node.js. "Covered by dependency pattern" is **NEVER** valid for skipping an export rename or relocation. Every renamed export needs its own `nodejs.referenced` pattern — the dependency pattern only detects the old package in `package.json`, not the usage sites in source code.
