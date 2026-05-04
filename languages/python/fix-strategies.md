# Python Provider — Fix Reference

## Fix lookup — 0 incidents by condition type

| Condition + Location | Fix |
|---|---|
| `python.referenced` | Ensure both import AND usage exist; run `pip install -r requirements.txt` for symbol resolution |
| `builtin.filecontent` | Ensure file has text matching the regex; check `filePattern` is Go regex not glob |

## Details

### python.referenced

Kantra uses pylsp/pyright to resolve Python references. The Python provider runs in `SourceOnlyAnalysisMode`.

#### Import and usage both required

The test code must import AND use the symbol. Import alone may not trigger a match.

```python
# FAILS: import-only
from flask import Flask

# WORKS: import + usage
from flask import Flask
app = Flask(__name__)
```

#### Package installation required

`pip install -r requirements.txt` is required for the language server to resolve imports and types. Use a virtualenv to avoid polluting the system.

#### No location filtering

The Python provider does not support `location` constraints. Any reference to the symbol matches regardless of context (import, call, type annotation).

### builtin.filecontent — dependency workaround

Since `python.dependency` is not implemented, use `builtin.filecontent` to match dependency declarations:

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

### Compilation fixes

1. Check syntax: `python -m py_compile main.py`
2. For type checking (if available): `mypy main.py`
3. After fixing: `pip install -r requirements.txt`
4. Use real package versions from PyPI — do NOT fabricate
