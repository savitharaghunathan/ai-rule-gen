# Python Test Data Guide

## Project Structure

```
<data-dir>/
├── requirements.txt    # pip dependencies
└── main.py             # Source code
```

## How the Analyzer Matches Python Conditions

| Condition Type | What the Test Code Must Do |
|---|---|
| `python.referenced` | Import and use the symbol (e.g., `from flask import Flask` then `app = Flask(__name__)`). Both import and usage are needed for reliable matching. |
| `builtin.filecontent` | Include text matching the regex pattern in the appropriate file (e.g., dependency entries in `requirements.txt` or `pyproject.toml`). |

## Dependency Resolution

Run after writing test files:

```bash
pip install -r requirements.txt
```

The Python language server (pylsp/pyright) needs installed packages to resolve imports and types. Use a virtualenv to avoid polluting the system.

## requirements.txt Structure

```
flask==2.3.3
django==4.2.7
cryptography==41.0.0
```

Use exact versions with `==` for reproducibility. Use real versions from PyPI.

## main.py Structure

```python
# Rule: migration-rule-00010
from flask import Flask

# Rule: migration-rule-00020
from cryptography.fernet import Fernet

app = Flask(__name__)
key = Fernet.generate_key()
```

Use `# Rule: <ruleID>` comments (Python comment syntax).

## Compilation Check

Basic syntax check:
```bash
python -m py_compile main.py
```

For type checking (if available):
```bash
mypy main.py
```
