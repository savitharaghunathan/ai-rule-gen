# C# Condition Types

## csharp.referenced

Matches C# symbol references — type usage, method calls, field declarations.

**Fields:**
- `pattern` (required) — Fully qualified name (e.g., `System.Web.HttpContext`, `Microsoft.EntityFrameworkCore.DbContext`).
- `location` (optional) — Where the reference appears. **Only 2 values supported** (vs Java's 14):

| Location | What It Matches |
|---|---|
| `ALL` (default) | Any reference to the symbol |
| `METHOD` | Method declarations or calls |

**Important:** Java-style locations like `ANNOTATION`, `INHERITANCE`, `CONSTRUCTOR_CALL`, etc. are NOT supported for C#. Use `ALL` for most rules.

## csharp.dependency — NOT YET IMPLEMENTED

The C# provider (separate Rust repo `c-sharp-analyzer-provider`) has gRPC interface definitions for `GetDependencyLocationRequest` and `DependencyResponse` but they are not implemented. NuGet package version detection via `csharp.dependency` is not available.

**Workaround:** Use `builtin.xml` with XPath on `.csproj` files:

```yaml
when:
  builtin.xml:
    xpath: //*[local-name()='PackageReference' and @Include='Newtonsoft.Json']
    filepaths:
      - Project.csproj
```

To match version ranges, use `builtin.filecontent`:

```yaml
when:
  builtin.filecontent:
    pattern: 'PackageReference Include="Newtonsoft.Json" Version="'
    filePattern: .*\.csproj
```
