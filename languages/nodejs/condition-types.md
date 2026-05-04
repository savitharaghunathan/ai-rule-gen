# Node.js / TypeScript Condition Types

## nodejs.referenced

Matches Node.js/TypeScript symbol references — imports, component usage, type references.

**Fields:**
- `pattern` (required) — Package + exported symbol (e.g., `express.Router`, `@patternfly/react-core.Button`, `React`).

**No location filtering.** Unlike Java, the Node.js provider does not support `location` constraints. The analyzer matches any reference to the symbol regardless of where it appears (import, usage, type annotation).

**Pattern matching behavior:**
- Matches symbol names in `.ts`, `.tsx`, `.js`, `.jsx` files
- Import and usage are both matched — a single `pattern: "React"` matches both `import React from 'react'` and `React.FC`
- Node module files (`node_modules/`) are excluded from matching

## nodejs.dependency — NOT YET IMPLEMENTED

The Node.js provider's `GetDependencies()` is a stub that returns nil. Dependency version detection in `package.json` is not supported via `nodejs.dependency`.

**Workaround:** Use `builtin.json` with XPath to match `package.json` entries:

```yaml
when:
  builtin.json:
    xpath: /dependencies/@patternfly/react-core
    filepaths:
      - package.json
```

Or use `builtin.filecontent` with a regex:

```yaml
when:
  builtin.filecontent:
    pattern: '"@patternfly/react-core"\s*:\s*"'
    filePattern: package\.json
```
