# Node.js-Specific Instructions

## Package Registry Pre-Check

Not yet implemented for Node.js. npm packages use `registry.npmjs.org` for version resolution. Skip the registry pre-check for now — emit `nodejs.dependency` patterns without version verification.

## Source Artifact Resolution

For `nodejs.referenced` patterns, `source_artifact` is not currently supported by the verifier. Omit it.

## Validation Notes

- Node.js has 2 valid location types: `IMPORT`, `PACKAGE`
- `nodejs.dependency` matches npm package names in `package.json`
