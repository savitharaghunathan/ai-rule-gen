# Go Extraction Examples

## Example 1: `go.referenced` -- non-FIPS crypto replacement

### Guide Excerpt

> ### Removed: golang.org/x/crypto/md4
>
> The `golang.org/x/crypto/md4` package has been removed because MD4 is not
> FIPS 140-2 compliant. Applications that hash with MD4 must migrate to a
> FIPS-approved algorithm such as `crypto/sha256` from the standard library.

### Checklist

Section: "Removed: golang.org/x/crypto/md4" -> EXTRACT: removed crypto package (item 1)

### patterns.json

```json
{
  "source_pattern": "golang.org/x/crypto/md4 removed (not FIPS compliant)",
  "target_pattern": "crypto/sha256",
  "source_fqn": "golang.org/x/crypto/md4",
  "rationale": "MD4 is not FIPS 140-2 compliant; replace with crypto/sha256 from the standard library",
  "complexity": "low",
  "category": "mandatory",
  "concern": "security",
  "provider_type": "go",
  "documentation_url": "https://pkg.go.dev/crypto/sha256"
}
```

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: go-non-fips-to-go-fips-00010
  description: MD4 is not FIPS 140-2 compliant; replace with crypto/sha256 from the standard library
  category: mandatory
  effort: 3
  labels:
    - konveyor.io/source=go-non-fips
    - konveyor.io/target=go-fips
  message: "golang.org/x/crypto/md4 removed (not FIPS compliant): MD4 is not FIPS 140-2 compliant; replace with crypto/sha256 from the standard library"
  links:
    - title: Migration Documentation
      url: https://pkg.go.dev/crypto/sha256
  when:
    go.referenced:
      pattern: golang.org/x/crypto/md4
```

### Test Data (what triggers this rule)

```go
package main

import "golang.org/x/crypto/md4"

func hash(data []byte) []byte {
    h := md4.New()
    h.Write(data)
    return h.Sum(nil)
}
```
