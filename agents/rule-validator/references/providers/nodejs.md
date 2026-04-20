# Node.js Provider — Fix Reference

How `nodejs.referenced` rules work in kantra, and what goes wrong.

## nodejs.referenced

Kantra uses TypeScript analysis to resolve Node.js references.

### Import and usage required

The test code must import AND use the symbol:
```typescript
import { Button } from '@patternfly/react-core';
```

### Type resolution

If rules fail, `npm install` may be needed for TypeScript to resolve types from `node_modules`.

## Compilation fixes (Node.js-specific)

1. Check compilation: `npx tsc --noEmit`
2. Check TypeScript errors for property/member name info (e.g., "Property 'foo' does not exist on type 'Bar'")
3. After fixing: `npm install`
