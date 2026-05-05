# C# Provider — Fix Reference

## Fix lookup — 0 incidents by condition type

| Condition + Location | Fix |
|---|---|
| `csharp.referenced` | Ensure fully qualified type or symbol is used in code; run `dotnet restore` for type resolution |
| `csharp.referenced` + `METHOD` | Ensure the method is declared or called explicitly |
| `builtin.xml` | Ensure `.csproj` or XML file matches the XPath; check `filepaths` setting |
| `builtin.filecontent` | Ensure file has text matching the regex; check `filePattern` is Go regex not glob |

## Details

### csharp.referenced

Kantra uses .NET analysis to resolve C# references. The C# provider is a separate Rust-based analyzer (`c-sharp-analyzer-provider`).

#### Only 2 location values supported

Unlike Java's 14 location types, C# supports only:
- `ALL` (default) — any reference to the symbol
- `METHOD` — method declarations or calls

Do NOT use Java-style locations (`ANNOTATION`, `INHERITANCE`, `CONSTRUCTOR_CALL`, etc.) — they are not supported and will be ignored.

#### Type resolution requires dotnet restore

`dotnet restore` is always required. Without NuGet packages restored, the analyzer cannot resolve type references and `csharp.referenced` rules fail with unresolved type errors.

```bash
dotnet restore
```

#### Using directives + usage required

The test code must have a `using` directive AND use the type:

```csharp
// FAILS: using-only
using Newtonsoft.Json;

// WORKS: using + usage
using Newtonsoft.Json;
public class Example
{
    public void Run()
    {
        var json = JsonConvert.SerializeObject(new { Name = "test" });
    }
}
```

### builtin.xml — dependency workaround

Since `csharp.dependency` is not implemented, use `builtin.xml` to match `.csproj` entries:

```yaml
when:
  builtin.xml:
    xpath: //*[local-name()='PackageReference' and @Include='Newtonsoft.Json']
    filepaths:
      - Project.csproj
```

Test YAML for `builtin.xml` rules should NOT have `mode: source-only`.

### Compilation fixes

1. Check compilation: `dotnet build --no-restore`
2. Check errors for type/member info (e.g., "'Type' does not contain a definition for 'Member'")
3. After fixing: `dotnet restore`
4. NuGet package versions must be real versions from nuget.org — do NOT fabricate
