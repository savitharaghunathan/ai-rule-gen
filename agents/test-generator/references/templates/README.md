# Template-Driven Test Generation

Each language folder provides a minimal build template and source template.

Required placeholders:

- `{{DEPENDENCIES}}` — dependency entries for the build file
- `{{RULE_SNIPPETS}}` — source code snippets that trigger assigned rule patterns

Workflow:

1. Copy templates for the target language into the group's manifest file paths.
2. Fill placeholders using rule requirements.
3. Validate compile constraints from `references/languages/<language>/test-data-guide.md`.
