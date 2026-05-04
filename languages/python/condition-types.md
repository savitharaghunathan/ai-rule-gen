# Python Condition Types

## python.referenced

Matches Python symbol references — module imports, function calls, class usage.

**Fields:**
- `pattern` (required) — Module path with optional symbol name (e.g., `flask.Flask`, `django.db.models.Model`, `cryptography.fernet.Fernet`).

**No location filtering.** The Python provider uses LSP workspace symbol search without location constraints.

**Analysis mode:** Python runs in `SourceOnlyAnalysisMode` — it analyzes source files only, no binary or bytecode analysis.

## python.dependency — NOT YET IMPLEMENTED

The Python provider's `GetDependencies()` returns nil. Dependency version detection in `requirements.txt` or `pyproject.toml` is not supported via `python.dependency`.

**Workaround:** Use `builtin.filecontent` to match dependency declarations:

```yaml
# Match in requirements.txt
when:
  builtin.filecontent:
    pattern: flask[=<>!~]
    filePattern: requirements.*\.txt
```

```yaml
# Match in pyproject.toml
when:
  builtin.filecontent:
    pattern: '"flask[=<>!~]'
    filePattern: pyproject\.toml
```
