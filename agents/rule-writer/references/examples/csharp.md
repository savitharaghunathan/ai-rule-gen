# C# Extraction Examples

## Example 1: `csharp.referenced` -- namespace-level wildcard migration

### Guide Excerpt

> ### System.Web.Mvc Namespace
>
> The `System.Web.Mvc` namespace is not available in .NET Core. All MVC
> controllers, action results, and filters must be migrated to
> `Microsoft.AspNetCore.Mvc`. This includes `Controller`, `ActionResult`,
> `AuthorizeAttribute`, and all related types.

### Checklist

Section: "System.Web.Mvc Namespace" -> EXTRACT: relocated namespace (item 2)

### patterns.json

```json
{
  "source_pattern": "System.Web.Mvc namespace removed in .NET Core",
  "target_pattern": "Microsoft.AspNetCore.Mvc",
  "source_fqn": "System.Web.Mvc.*",
  "location_type": "ALL",
  "rationale": "System.Web.Mvc is not available in .NET Core; replace with Microsoft.AspNetCore.Mvc",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "web",
  "provider_type": "csharp",
  "documentation_url": "https://learn.microsoft.com/en-us/aspnet/core/migration/mvc"
}
```

Note: C# uses wildcard `.*` to match the entire namespace. `location_type: "ALL"` matches any reference to the symbol.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: dotnet-to-dotnet-core-00010
  description: System.Web.Mvc is not available in .NET Core; replace with Microsoft.AspNetCore.Mvc
  category: mandatory
  effort: 5
  labels:
    - konveyor.io/source=dotnet
    - konveyor.io/target=dotnet-core
  message: "System.Web.Mvc namespace removed in .NET Core: System.Web.Mvc is not available in .NET Core; replace with Microsoft.AspNetCore.Mvc"
  links:
    - title: Migration Documentation
      url: https://learn.microsoft.com/en-us/aspnet/core/migration/mvc
  when:
    csharp.referenced:
      pattern: System.Web.Mvc.*
      location: ALL
```

### Test Data (what triggers this rule)

```csharp
using System.Web.Mvc;

public class HomeController : Controller
{
    public ActionResult Index()
    {
        return View();
    }
}
```

---

## Example 2: `csharp.referenced` -- specific FQN replacement

### Guide Excerpt

> ### WebMatrix.WebData.WebSecurity
>
> `WebMatrix.WebData.WebSecurity` is not available in .NET Core. Replace all
> calls to `WebSecurity.Login`, `WebSecurity.Logout`, and
> `WebSecurity.CreateUserAndAccount` with ASP.NET Core Identity equivalents
> (`SignInManager.PasswordSignInAsync`, `SignInManager.SignOutAsync`,
> `UserManager.CreateAsync`).

### Checklist

Section: "WebMatrix.WebData.WebSecurity" -> EXTRACT: removed library (item 1)

### patterns.json

```json
{
  "source_pattern": "WebMatrix.WebData.WebSecurity removed in .NET Core",
  "target_pattern": "ASP.NET Core Identity (SignInManager, UserManager)",
  "source_fqn": "WebMatrix.WebData.WebSecurity",
  "location_type": "ALL",
  "rationale": "WebSecurity is not available in .NET Core; replace with ASP.NET Core Identity (SignInManager, UserManager)",
  "complexity": "high",
  "category": "mandatory",
  "concern": "security",
  "provider_type": "csharp",
  "documentation_url": "https://learn.microsoft.com/en-us/aspnet/core/security/authentication/identity"
}
```

Note: Specific FQN (no wildcard) because WebSecurity is a single class, not a namespace. `location_type: "ALL"` is always explicit for C#.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: dotnet-to-dotnet-core-00020
  description: WebSecurity is not available in .NET Core; replace with ASP.NET Core Identity (SignInManager, UserManager)
  category: mandatory
  effort: 7
  labels:
    - konveyor.io/source=dotnet
    - konveyor.io/target=dotnet-core
  message: "WebMatrix.WebData.WebSecurity removed in .NET Core: WebSecurity is not available in .NET Core; replace with ASP.NET Core Identity (SignInManager, UserManager)"
  links:
    - title: Migration Documentation
      url: https://learn.microsoft.com/en-us/aspnet/core/security/authentication/identity
  when:
    csharp.referenced:
      pattern: WebMatrix.WebData.WebSecurity
      location: ALL
```

### Test Data (what triggers this rule)

```csharp
using WebMatrix.WebData;

public class AccountController
{
    public void Login(string email, string password)
    {
        WebSecurity.Login(email, password);
    }
}
```
