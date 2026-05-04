# Go Provider — Fix Reference

## Fix lookup — 0 incidents by condition type

| Condition + Location | Fix |
|---|---|
| `go.referenced` | Ensure both import AND usage exist; run `go mod tidy && go mod vendor` |
| `go.dependency` | Ensure `go.mod` declares the module with version within bounds; remove `mode: source-only` from `.test.yaml` |
| `builtin.filecontent` | Ensure file has text matching the regex; check `filePattern` is Go regex not glob |

## Details

### go.referenced (gopls)

Kantra uses gopls to resolve Go references. `kantra test` defaults to `--run-local=true`, which uses the local Go toolchain.

#### Vendored modules required

gopls needs vendored modules to resolve symbols. After writing test code:
1. `go mod tidy`
2. `go mod vendor`

The vendor directory must be present in the test data.

#### Import + usage both required

```go
// FAILS: import-only (gopls won't find a reference without usage)
import "golang.org/x/crypto/md4"

// WORKS: import + usage
import "golang.org/x/crypto/md4"
var _ = md4.New()
```

### go.dependency

`go.dependency` rules check `go.mod` for a module dependency with a version within the rule's bounds.

- Test YAML should NOT have `mode: source-only`
- The `go.mod` must declare the module with a version within bounds
- Version must be strictly less than `upperbound` and >= `lowerbound`

### Compilation fixes

1. Check compilation: `go build ./...`
2. Use `go doc <package>` to get actual function signatures — don't guess
3. After fixing: `go mod tidy` then `go mod vendor`
