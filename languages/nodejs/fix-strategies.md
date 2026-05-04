# Node.js Provider — Fix Reference

## Fix lookup — 0 incidents by condition type

| Condition + Location | Fix |
|---|---|
| `nodejs.referenced` | Ensure both import AND usage exist; run `npm install` for type resolution |
| `builtin.json` | Ensure JSON file matches the XPath; check file path matches `filepaths` |
| `builtin.filecontent` | Ensure file has text matching the regex; check `filePattern` is Go regex not glob |

## Details

### nodejs.referenced

Kantra uses TypeScript analysis to resolve Node.js references.

#### Import and usage both required

The test code must import AND use the symbol. Import alone may not trigger a match.

```typescript
// FAILS: import-only
import { Button } from '@patternfly/react-core';

// WORKS: import + usage
import { Button } from '@patternfly/react-core';
const element = <Button variant="primary">Click</Button>;
```

#### Type resolution requires npm install

`npm install` is always required. Without `node_modules`, TypeScript cannot resolve types and `nodejs.referenced` rules fail with unresolved symbol errors.

#### TSX requires tsconfig.json

If source files use `.tsx`, a `tsconfig.json` with `"jsx": "react-jsx"` is required. Without it, the TypeScript compiler cannot parse JSX syntax.

```json
{
  "compilerOptions": {
    "jsx": "react-jsx",
    "moduleResolution": "node",
    "esModuleInterop": true
  }
}
```

#### No location filtering

Unlike Java, Node.js does not support `location` constraints. If a rule was generated with a location field, it will be ignored — any reference to the symbol matches.

### builtin.json — dependency workaround

Since `nodejs.dependency` is not implemented, use `builtin.json` to match `package.json` entries:

```yaml
when:
  builtin.json:
    xpath: /dependencies/@patternfly/react-core
    filepaths:
      - package.json
```

Test YAML for `builtin.json` rules should NOT have `mode: source-only`.

### Compilation fixes

1. Check compilation: `npx tsc --noEmit`
2. Check TypeScript errors for property/member name info (e.g., "Property 'foo' does not exist on type 'Bar'")
3. After fixing: `npm install`
4. If errors persist with type resolution, check that `@types/*` packages are in `devDependencies`
