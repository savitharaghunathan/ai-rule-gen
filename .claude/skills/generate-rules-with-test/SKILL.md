---
name: generate-rules-with-test
description: Generate Konveyor analyzer migration rules with test generation and validation from a migration guide (URL, file path, or pasted text). Use when the user wants to create tested Konveyor rules with full validation.
---

Read `agents/generate-rules/SKILL.md` and follow its instructions.

Two command presets are available:
- `/generate-rules` — rules only (no testing). Sets `checkpoint_behavior=stop_after_extract`.
- `/generate-rules-with-test` — full pipeline with test generation and validation. Sets `checkpoint_behavior=continue`.

**Input:** checkpoint_behavior=continue $ARGUMENTS
