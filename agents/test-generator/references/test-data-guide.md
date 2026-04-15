# Test Data Generation Guide

This guide explains how to generate test source code that triggers Konveyor analyzer rules. Derived from the test data generation template.

## Goal

For each rule, generate a COMPLETE, COMPILABLE project where the source code contains code that EXACTLY matches the rule's `when` condition pattern. The analyzer will run against this code and must find at least 1 incident per rule.

## Requirements

1. Create a COMPLETE, COMPILABLE project for the detected language
2. For EACH rule, include code that EXACTLY matches the pattern in the `when` condition
3. Add a comment before each pattern: `// Rule: <ruleID>`
4. Keep code minimal — one example per rule, just enough to trigger the pattern
5. All imports/dependencies must be valid and resolve

## How the Analyzer Matches Each Condition Type

This is CRITICAL — the analyzer matches patterns against fully qualified names and source references. If you don't follow these rules, the test code won't trigger the rule.

| Condition Type | Location | What the Test Code Must Do |
|---|---|---|
| `java.referenced` | `ANNOTATION` | Use the annotation on a class, method, or field (e.g., `@Stateless`) |
| `java.referenced` | `IMPORT` | Include the import statement (e.g., `import javax.servlet.http.HttpServlet;`) |
| `java.referenced` | `TYPE` | Declare or use the type (e.g., `HttpServlet servlet;` or a cast) |
| `java.referenced` | `METHOD_CALL` | Call the method on an explicitly typed variable — do NOT chain calls (e.g., use `Foo f = Foo.get(); f.bar();` not `Foo.get().bar()`). JDTLS in source-only mode cannot resolve return types of chained calls without the dependency JAR. |
| `java.referenced` | `CONSTRUCTOR_CALL` | Use `new ClassName()` |
| `java.referenced` | `INHERITANCE` | `class Foo extends TargetClass` |
| `java.referenced` | `IMPLEMENTS_TYPE` | `class Foo implements TargetInterface` |
| `java.referenced` | `FIELD` | Declare a field of that type |
| `java.referenced` | `VARIABLE_DECLARATION` | Declare a local variable of that type |
| `java.referenced` | `RETURN_TYPE` | Declare a method with that return type |
| `go.referenced` | — | Import and use the package/symbol (e.g., `import "golang.org/x/crypto/md4"` then use `md4.New()`) |
| `nodejs.referenced` | — | Import and use the symbol (e.g., `import { Button } from '@patternfly/react-core';`) |
| `csharp.referenced` | — | Use the fully qualified type/symbol |
| `builtin.filecontent` | — | Include text in the appropriate file that matches the regex pattern. Check the `filePattern` field to know which file type |
| `java.dependency` | — | The `pom.xml` must declare the dependency with a version within the rule's bounds. Use the `name` field as `groupId.artifactId` (dot-separated). E.g., name `org.springframework.boot.spring-boot-starter-undertow` + upperbound `4.0.0` → pom.xml needs `<artifactId>spring-boot-starter-undertow</artifactId>` with a version below 4.0.0. **No source code needed** — only the pom.xml matters. |
| `go.dependency` | — | The `go.mod` must declare the module dependency with a version within the rule's bounds |
| `builtin.xml` | — | The XML file (usually `pom.xml`) must contain elements matching the XPath expression. If `filepaths` is set, the file must be at that path. If `namespaces` is set, ensure the XML uses those namespace URIs |

## Output Format

Generate EXACTLY TWO fenced code blocks per test group:

**FIRST block:** Build file contents (pom.xml, go.mod, package.json, or Project.csproj)

**SECOND block:** Main source file contents (Application.java, main.go, App.tsx, or Program.cs)

Do NOT include any other text or code blocks.

## Project Structure Per Language

### Java
```
<data-dir>/
├── pom.xml                                        # Maven build file
└── src/main/java/com/example/Application.java     # Source code
```
- Build file: `pom.xml` (type: xml)
- Source dir: `src/main/java/com/example`
- Main file: `Application.java` (type: java)
- Dependencies must be valid Maven coordinates that resolve

### Go
```
<data-dir>/
├── go.mod     # Module definition
└── main.go    # Source code
```
- Build file: `go.mod` (type: go)
- Source dir: `.` (root)
- Main file: `main.go` (type: go)
- After writing, run `go mod tidy` and `go mod vendor` so gopls in the kantra container can resolve modules

### Node.js / TypeScript
```
<data-dir>/
├── package.json    # NPM package definition
└── src/App.tsx     # Source code
```
- Build file: `package.json` (type: json)
- Source dir: `src`
- Main file: `App.tsx` (type: tsx)

### C# / .NET
```
<data-dir>/
├── Project.csproj    # .NET project file
└── Program.cs        # Source code
```
- Build file: `Project.csproj` (type: xml)
- Source dir: `.` (root)
- Main file: `Program.cs` (type: csharp)

## Reading the manifest.json

`go run ./cmd/scaffold` outputs a `manifest.json` that tells you exactly what files to generate:

```json
{
  "language": "java",
  "groups": [
    {
      "name": "web",
      "data_dir": "tests/data/web",
      "test_file": "tests/web.test.yaml",
      "rule_count": 3,
      "providers": ["java"],
      "files": [
        {"path": "tests/data/web/pom.xml", "file_type": "xml", "purpose": "build"},
        {"path": "tests/data/web/src/main/java/com/example/Application.java", "file_type": "java", "purpose": "source"}
      ],
      "rule_ids": ["rule-00010", "rule-00020", "rule-00030"]
    }
  ]
}
```

For each group:
1. Read the rules referenced by `rule_ids` to see what patterns must be matched
2. Generate the build file at the `path` with `purpose: "build"`
3. Generate the source file at the `path` with `purpose: "source"`
4. The source code must trigger ALL rules in that group

## Dependency Resolution

- **Go:** Always run `go mod tidy` then `go mod vendor` (gopls inside the kantra container can't download modules)
- **Java:** `mvn dependency:resolve -q -B` only if METHOD_CALL rules fail. Source-only analysis resolves IMPORT/ANNOTATION/TYPE patterns without downloading dependencies.
- **Node.js:** `npm install` only if needed for type resolution
- **C#:** `dotnet restore` only if needed for type resolution

### Java pom.xml: prefer minimal dependencies

JDTLS runs inside the kantra container with limited memory. Prefer the lightest dependency that provides the class you need — e.g., `spring-boot-autoconfigure` instead of a full starter like `spring-boot-starter-web`.

### java.dependency and builtin.xml tests

These condition types do not use JDTLS. Test YAML files for `java.dependency` and `builtin.xml` rules should omit `analysisParams: mode: source-only`.

## XML Sanitization

After generating all test files, run `go run ./cmd/sanitize --dir <tests-dir>` to clean XML files. LLMs frequently generate comments like `<!-- --add-opens flag -->` which contains `--` inside a comment — this is illegal XML and breaks Maven's POM parser. The sanitizer replaces `--` sequences inside XML comments with spaces.

## Fix Iterations

When a rule fails kantra tests, you'll receive:
- The failing rule IDs
- Their patterns (from the rule YAML)
- Failure context from the validator

On fix iterations:
- Regenerate ONLY the failing test groups (preserve passing groups)
- Use the fix guidance to understand what the test code is missing
- The most common failure is: the test code doesn't actually use the API that the rule pattern matches
