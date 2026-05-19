# Python Extraction Checklist

Language-specific extraction guidance for Python migration rules. Items not listed here follow the universal guidance in SKILL.md.

Python has **no location filtering** — `python.referenced` matches any reference to the module path + symbol via LSP workspace symbol search. Python runs in `SourceOnlyAnalysisMode`.

## Checklist item extraction details

### Item 2: Symbol or module removed or relocated

Use `python.referenced` with the module path + symbol name:
- Class relocated: `flask.Flask` (matches any reference)
- Function removed: `django.conf.urls.url`
- For entire module relocations: use the old module path (e.g., `old.module`)

No location filtering is needed or available — `python.referenced` matches any reference site.

### Item 4: Reference table with old→new mappings

Process every row as a `python.referenced` pattern. For module-wide renames where all symbols keep the same names, emit ONE pattern on the old module path — additional patterns are only needed for symbols whose names changed. See Module-level consolidation below.

### Item 7: Names any specific artifact

| Artifact type | Condition | Pattern example |
|---|---|---|
| Module-level symbol | `python.referenced` | `flask.Flask` |
| Decorator | `python.referenced` | `flask.app.route` |
| Configuration value | `builtin.filecontent` | Regex on settings files |
| String-based reference | `builtin.filecontent` | Regex on template names, URL patterns |
| pip dependency | `builtin.filecontent` on `requirements.txt` / `pyproject.toml` | Package name pattern |

### Item 9: Before/after code examples

Each API difference produces a `python.referenced` or `builtin.filecontent` pattern. There is no location type distinction.

| Diff category | Condition | Pattern |
|---|---|---|
| Class/type rename | `python.referenced` | `old.module.OldClass` |
| Function rename | `python.referenced` | `old.module.old_func` |
| Module relocation | `python.referenced` | `old.module` |
| Decorator change | `python.referenced` | `old.module.old_decorator` |
| Config/settings change | `builtin.filecontent` | Regex on config files |

## TABLE output format

When a section contains a reference table, enumerate every row:

```
Table: "<section heading>" (<N> rows)
Row 1: old.module.OldClass → new.module.NewClass — EXTRACT (class renamed/relocated)
Row 2: old.module.SameClass → new.module.SameClass — module-path covers (same name, just moved)
Row 3: old.module.old_func() → new.module.new_func() — EXTRACT (function renamed)
Row 4: old.module.CONST → new.module.CONST — module-path covers (same name)
...
```

Every row must appear. This prevents silent drops.

## CODE-DIFF annotation guidance

When annotating code diffs, there is no location type to specify — all Python patterns use `python.referenced`:

```
Code diff: "## Migration steps" (source example vs target example)
  from old.module import OldClass → from new.module import NewClass — EXTRACT (detect via python.referenced)
  old.module.func() → new.module.new_func() — EXTRACT (detect via python.referenced)
  @old_decorator → @new_decorator — EXTRACT (detect via python.referenced)
```

## Module-level consolidation

When a migration guide says an entire Python module/package is renamed:

**How to recognize a module rename:**
- The guide says "import from new module," "package renamed," or similar
- A table lists old→new symbol mappings where every old symbol shares the same module path

**Strategy:** Use `python.referenced` on the old module path to catch imports of any symbol from that module.

**Decision tree:**

| Scenario | Correct output |
|---|---|
| Module `old.module` replaced by `new.module`, all symbols kept same names | ONE `python.referenced` pattern on `old.module` |
| Module replaced + some symbols renamed | ONE module-path pattern + ONE pattern per renamed symbol |
| Symbols moved to different modules | Separate per-symbol patterns |

## Dependency detection

`python.dependency` is **NOT implemented**. Use `builtin.filecontent` to match dependency declarations:

```yaml
# requirements.txt
when:
  builtin.filecontent:
    pattern: 'flask[=<>!~]'
    filePattern: 'requirements.*\.txt'
```

```yaml
# pyproject.toml
when:
  builtin.filecontent:
    pattern: '"flask[=<>!~]'
    filePattern: 'pyproject\.toml'
```

## Skip reason validation (Python-specific)

"Covered by module path pattern" is **NEVER** valid for skipping a symbol rename. If `old.module.Foo` became `new.module.Bar`, the module path pattern catches the import but the user needs to know `Foo` is now `Bar`. Each renamed symbol needs its own pattern.
