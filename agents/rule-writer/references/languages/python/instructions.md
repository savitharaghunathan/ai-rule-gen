# Python-Specific Instructions

## Package Registry Pre-Check

Not yet implemented for Python. PyPI packages use `pypi.org/pypi/<package>/json` for version resolution. Skip the registry pre-check for now — emit `python.dependency` patterns without version verification.

## Source Artifact Resolution

For `python.referenced` patterns, `source_artifact` is not currently supported by the verifier. Omit it.

## Validation Notes

- Python provider support in kantra is limited — check `condition-types.md` for current capabilities
- `python.dependency` matches package names in `requirements.txt` and `pyproject.toml`
