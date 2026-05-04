# C# Test Data Guide

## Project Structure

```
<data-dir>/
├── Project.csproj    # .NET project file
└── Program.cs        # Source code
```

## How the Analyzer Matches C# Conditions

| Condition Type | What the Test Code Must Do |
|---|---|
| `csharp.referenced` | Use the fully qualified type or symbol in code. The analyzer resolves FQNs from `using` directives + type usage. |
| `csharp.referenced` (location: `METHOD`) | Call or declare the method. |
| `builtin.filecontent` | Include text matching the regex pattern in the appropriate file. |
| `builtin.xml` | The `.csproj` or other XML file must contain elements matching the XPath expression. |

## Dependency Resolution

**Always required.** Run after writing test files:

```bash
dotnet restore
```

Kantra's C# analyzer needs NuGet packages restored for type resolution. Without it, `csharp.referenced` rules will fail with unresolved type errors.

## Project.csproj Structure

A valid `.csproj` for test data:

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="Microsoft.EntityFrameworkCore" Version="8.0.0" />
  </ItemGroup>
</Project>
```

**NuGet package versions:** Use real versions from nuget.org. Do NOT fabricate version numbers.

## Program.cs Structure

```csharp
using System;
using Newtonsoft.Json;

// Rule: migration-rule-00010
public class Program
{
    // Rule: migration-rule-00020
    public static void Main(string[] args)
    {
        var obj = JsonConvert.SerializeObject(new { Name = "test" });
    }
}
```

## Compilation Check

```bash
dotnet build --no-restore
```

Check errors for type/member info (e.g., "'Type' does not contain a definition for 'Member'").
