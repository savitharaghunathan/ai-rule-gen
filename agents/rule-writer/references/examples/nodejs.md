# Node.js Extraction Examples

## Contents

- Example 1: `nodejs.referenced` -- component rename
- Example 2: `builtin.filecontent` -- API removal

## Example 1: `nodejs.referenced` -- component rename

### Guide Excerpt

> ### Text Component Consolidation
>
> The `Text`, `TextContent`, `TextList`, and `TextListItem` components have
> been replaced by a single `Content` component. Import `Content` from
> `@patternfly/react-core` instead.
>
> | v5 Component | v6 Replacement |
> |---|---|
> | `Text` | `Content` |
> | `TextContent` | `Content` |
> | `TextList` | `Content component="ul"` |
> | `TextListItem` | `Content component="li"` |

### Checklist

Section: "Text Component Consolidation" -> EXTRACT: reference table with old->new mappings (item 4); each row is a separate pattern

### patterns.json

Each row in the reference table produces a separate pattern:

```json
{
  "source_pattern": "Text component replaced by Content in PatternFly v6",
  "target_pattern": "Content",
  "source_fqn": "Text",
  "rationale": "Text component removed in PatternFly v6; use Content from @patternfly/react-core",
  "complexity": "low",
  "category": "mandatory",
  "concern": "ui",
  "provider_type": "nodejs"
}
```

```json
{
  "source_pattern": "TextContent component replaced by Content in PatternFly v6",
  "target_pattern": "Content",
  "source_fqn": "TextContent",
  "rationale": "TextContent removed in PatternFly v6; use Content from @patternfly/react-core",
  "complexity": "low",
  "category": "mandatory",
  "concern": "ui",
  "provider_type": "nodejs"
}
```

Note: `nodejs.referenced` has no `location_type` filter -- it matches any reference to the named export. Each old component name is a separate pattern, even when they all map to the same replacement.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: patternfly-v5-to-patternfly-v6-00010
  description: Text component removed in PatternFly v6; use Content from @patternfly/react-core
  category: mandatory
  effort: 3
  labels:
    - konveyor.io/source=patternfly-v5
    - konveyor.io/target=patternfly-v6
  when:
    nodejs.referenced:
      pattern: Text
```

### Test Data (what triggers this rule)

```tsx
import { Text } from '@patternfly/react-core';

const App = () => (
  <Text component="h3">Hello World</Text>
);
```

---

## Example 2: `builtin.filecontent` -- API removal

### Guide Excerpt

> ### Removed: Astro.glob()
>
> `Astro.glob()` has been removed in Astro 5. Use Vite's
> `import.meta.glob()` or content collections (`getCollection` from
> `astro:content`) instead.
>
> ```typescript
> // Before
> const posts = await Astro.glob('./posts/*.md');
>
> // After
> const posts = await import.meta.glob('./posts/*.md', { eager: true });
> ```

### Checklist

Section: "Removed: Astro.glob()" -> EXTRACT: removed API (item 1)

### patterns.json

```json
{
  "source_pattern": "Astro.glob() removed in Astro 5",
  "target_pattern": "import.meta.glob()",
  "source_fqn": "Astro.glob",
  "rationale": "Astro.glob() removed in Astro 5; use import.meta.glob() or content collections",
  "complexity": "medium",
  "category": "mandatory",
  "concern": "build",
  "provider_type": "builtin",
  "documentation_url": "https://docs.astro.build/en/guides/upgrade-to/v5/"
}
```

Note: `provider_type` is `"builtin"` (not `"nodejs"`) because `Astro.glob` is a framework global, not a module export detectable by `nodejs.referenced`. Use `builtin.filecontent` when the API is not a standard import/export.

### Resulting Rule YAML (produced by cmd/construct, not by you)

```yaml
- ruleID: astro-v4-to-astro-v5-00010
  description: Astro.glob() removed in Astro 5; use import.meta.glob() or content collections
  category: mandatory
  effort: 5
  labels:
    - konveyor.io/source=astro-v4
    - konveyor.io/target=astro-v5
  when:
    builtin.filecontent:
      pattern: Astro.glob
```

### Test Data (what triggers this rule)

```astro
---
const posts = await Astro.glob('./posts/*.md');
---

<ul>
  {posts.map(post => <li>{post.frontmatter.title}</li>)}
</ul>
```
