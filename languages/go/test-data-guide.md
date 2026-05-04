# Go Test Data Guide

## Project Structure

```
<data-dir>/
├── go.mod     # Module definition
└── main.go    # Source code
```

## How the Analyzer Matches Go Conditions

| Condition Type | What the Test Code Must Do |
|---|---|
| `go.referenced` | Import and use the package/symbol (e.g., `import "golang.org/x/crypto/md4"` then use `md4.New()`). Both import AND usage are required. |
| `go.dependency` | The `go.mod` must declare the module dependency with a version within the rule's bounds. |
| `builtin.filecontent` | Include text matching the regex pattern in the appropriate file. |

## Dependency Resolution

Always run after writing test files:

```bash
go mod tidy
go mod vendor
```

gopls needs vendored modules — it cannot download them at analysis time.

## go.dependency Version Bounds

Same semantics as Java:
- Version must be **strictly less than** `upperbound` (if set)
- Version must be **greater than or equal to** `lowerbound` (if set)

Use real module versions from pkg.go.dev. Go versions are prefixed with `v` (e.g., `v0.3.7`).

## Compilation Check

Run `go build ./...` to verify test code compiles. Use `go doc <package>` to look up actual function signatures when fixing errors.
