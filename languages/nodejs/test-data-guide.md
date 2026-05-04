# Node.js / TypeScript Test Data Guide

## Project Structure

```
<data-dir>/
├── package.json       # NPM package definition
├── tsconfig.json      # TypeScript configuration (required for TSX)
└── src/App.tsx        # Source code
```

## How the Analyzer Matches Node.js Conditions

| Condition Type | What the Test Code Must Do |
|---|---|
| `nodejs.referenced` | Import and use the symbol (e.g., `import { Button } from '@patternfly/react-core';` then use `<Button>` or reference `Button` in code). Both import and usage are matched. |
| `builtin.filecontent` | Include text matching the regex pattern in the appropriate file. |
| `builtin.json` | The JSON file must contain the matching XPath expression (e.g., dependency entry in `package.json`). |

## Dependency Resolution

**Always required.** Run after writing test files:

```bash
npm install
```

Kantra's TypeScript analyzer needs `node_modules` to resolve types. Without it, `nodejs.referenced` rules will fail with unresolved symbol errors.

## package.json Structure

A valid `package.json` for test data:

```json
{
  "name": "test-app",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "@patternfly/react-core": "^4.276.6",
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "@types/react": "^18.0.0"
  }
}
```

**Version ranges:** Use caret ranges (`^4.276.6`) for realistic entries. The exact version doesn't matter for `nodejs.referenced` rules — only the symbol presence in code matters.

## tsconfig.json

Required when using `.tsx` files:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "jsx": "react-jsx",
    "moduleResolution": "node",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src"]
}
```

## JSX vs TSX

- `.tsx` files require TypeScript + JSX support in `tsconfig.json`
- `.jsx` files work with plain JavaScript
- The Node.js provider handles both — match the file extension to what the migration guide targets

## Compilation Check

```bash
npx tsc --noEmit
```

Check TypeScript errors for property/member name info (e.g., "Property 'foo' does not exist on type 'Bar'").
