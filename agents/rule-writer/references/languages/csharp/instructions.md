# C#-Specific Instructions

## Package Registry Pre-Check

Not yet implemented for C#. NuGet packages use `api.nuget.org` for version resolution. Skip the registry pre-check for now — emit `csharp.dependency` patterns without version verification.

## Source Artifact Resolution

For `csharp.referenced` patterns, `source_artifact` is not currently supported by the verifier. Omit it.

## Validation Notes

- C# has 4 valid location types: `ALL`, `METHOD`, `FIELD`, `CLASS`
- `csharp.dependency` matches NuGet package references in `.csproj` files
