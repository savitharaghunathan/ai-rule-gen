# Go Condition Types

## go.referenced

Matches Go symbol references — package imports, function calls, type usage.

**Fields:**
- `pattern` (required) — Full import path, optionally with symbol name (e.g., `golang.org/x/crypto/md4`, `net.IP`, `crypto/md5.New`).

**Note:** `kantra test` defaults to `--run-local=true` as of kantra `586d9a4`. Go rules work without special handling in local mode.

## go.dependency

Matches Go module dependencies from `go.mod`.

**Fields:**
- `name` — Module path (e.g., `golang.org/x/crypto`). One of `name` or `nameRegex` required.
- `nameRegex` — Regex alternative.
- `upperbound`, `lowerbound` — Version bounds (e.g., `lowerbound: v0.3.0`, `upperbound: v0.5.0`).

**Version bound rules:** Same semantics as Java — use `lowerbound: 0.0.0` for removed modules, `upperbound` for behavior-changing versions.
