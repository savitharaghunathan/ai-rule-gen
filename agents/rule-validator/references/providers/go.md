# Go Provider — Fix Reference

How `go.referenced` and `go.dependency` rules work in kantra, and what goes wrong.

## go.referenced (gopls)

Kantra uses gopls to resolve Go references. Tests run locally via `--run-local=true`, using the local Go toolchain.

### Vendored modules recommended

For reproducible test results, vendor modules after writing test code:
1. `go mod tidy`
2. `go mod vendor`

## go.dependency

`go.dependency` rules check `go.mod` for a module dependency with a version within the rule's bounds.

- Test YAML should NOT have `mode: source-only`
- The `go.mod` must declare the module with a version within bounds

## Compilation fixes (Go-specific)

1. Check compilation: `go build ./...`
2. Use `go doc <package>` to get actual function signatures — don't guess
3. After fixing: `go mod tidy` then `go mod vendor`
