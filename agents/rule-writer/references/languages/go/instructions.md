# Go-Specific Instructions

## Package Registry Pre-Check

Not yet implemented for Go. Go modules use `proxy.golang.org` for version resolution. Skip the registry pre-check for now — emit `go.dependency` patterns without version verification.

## Source Artifact Resolution

For `go.referenced` patterns, `source_artifact` is not currently supported by the verifier. Omit it.

## Validation Notes

- Go has 2 valid location types: `IMPORT`, `PACKAGE`
- `go.dependency` matches Go module paths in `go.mod`
