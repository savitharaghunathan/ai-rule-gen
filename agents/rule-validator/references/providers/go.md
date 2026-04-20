# Go Provider — Fix Reference

How `go.referenced` and `go.dependency` rules work in kantra, and what goes wrong.

## go.referenced (gopls)

Kantra uses gopls to resolve Go references. The kantra container has gopls but NOT the Go toolchain.

### Container limitation

gopls in the kantra container cannot resolve modules without `go`. Rules using `go.referenced` fail with "no views" errors in container mode.

**Workaround:** Use `kantra analyze --run-local` for Go rules. The `cmd/test` runner detects Go provider from test files and uses `--run-local` automatically.

### Vendored modules required

gopls in the container can't download modules. After writing test code:
1. `go mod tidy`
2. `go mod vendor`

The vendor directory must be committed to the test data.

## go.dependency

`go.dependency` rules check `go.mod` for a module dependency with a version within the rule's bounds.

- Test YAML should NOT have `mode: source-only`
- The `go.mod` must declare the module with a version within bounds

## Compilation fixes (Go-specific)

1. Check compilation: `go build ./...`
2. Use `go doc <package>` to get actual function signatures — don't guess
3. After fixing: `go mod tidy` then `go mod vendor`
