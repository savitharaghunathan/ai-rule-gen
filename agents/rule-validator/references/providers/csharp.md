# C# Provider — Fix Reference

How `csharp.referenced` rules work in kantra, and what goes wrong.

## csharp.referenced

Kantra uses .NET analysis to resolve C# references.

### Usage required

The test code must reference the fully qualified type or symbol.

### Type resolution

If rules fail, `dotnet restore` may be needed for type resolution from NuGet packages.

## Compilation fixes (C#-specific)

1. Check compilation: `dotnet build --no-restore`
2. Check errors for type/member info (e.g., "'Type' does not contain a definition for 'Member'")
3. After fixing: `dotnet restore`
