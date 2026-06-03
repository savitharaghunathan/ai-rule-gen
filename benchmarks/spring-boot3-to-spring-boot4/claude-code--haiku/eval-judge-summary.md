## Eval Judge Report — claude-code / haiku / spring-boot3-to-spring-boot4

- **Pipeline failed** — no rules produced, no eval possible
- Haiku extracted 83 patterns from the migration guide but failed at the construct stage
- **3 `read_patterns_failed` errors**: could not read its own pattern JSON files back
- **3 `construct_failed` errors**: produced invalid regex (`*` — bare repetition operator) in rule conditions
- The SUMMARY.md shows Haiku understood the guide content (83 patterns across 12 categories) but lacked the capability to translate patterns into valid Konveyor YAML rules
