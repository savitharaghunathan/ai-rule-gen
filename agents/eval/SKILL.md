---
name: eval
description: Evaluate the generate-rules pipeline against golden sets using a podman-sandboxed run. Use when user wants to test skill quality, measure extraction coverage, or benchmark pipeline changes.
---

# eval

Run the generate-rules pipeline inside a podman container and grade output against golden sets.

## Inputs

- `guide` — URL or file path to a migration guide
- `golden_set` — (optional) Path to golden set YAML, e.g. `eval/golden/spring-boot-3-to-4.yaml`
- `model` — (optional) Model for pipeline run. Default: `google/gemini-2.5-flash`. Override with env var `OPENCODE_MODEL`.

## Returns

- `eval_report` — JSON eval report with per-agent grading
- `pass_rate` — Overall pass rate percentage
- `pipeline_log` — Path to full pipeline output log

## Workflow

### Step 1: Build container image (cached)

```bash
podman build -t ai-rule-gen-eval -f eval/Containerfile eval/
```

Podman caches layers — rebuilds are instant unless Containerfile changes.

### Step 2: Run pipeline + eval

Detect host podman socket:

```bash
PODMAN_SOCK="${XDG_RUNTIME_DIR:-/run}/podman/podman.sock"
if [ ! -S "$PODMAN_SOCK" ]; then
  PODMAN_SOCK="/run/podman/podman.sock"
fi
```

Run container with **same-path mounting**:

```bash
podman run --rm \
  -v $(pwd):$(pwd) -w $(pwd) \
  -v "$PODMAN_SOCK":/run/podman/podman.sock \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -e GOOGLE_GENERATIVE_AI_API_KEY="$GOOGLE_GENERATIVE_AI_API_KEY" \
  ai-rule-gen-eval \
  bash eval/run.sh "<guide>" "[golden_set]"
```

**Why same-path mounting**: kantra (inside the eval container) launches test containers via the host podman socket. Those test containers mount test data using absolute paths. If the repo path differs between host and eval container, kantra's volume mounts fail. Mounting `$(pwd):$(pwd)` ensures paths match.

**Why permissions are safe**: `--dangerously-skip-permissions` only affects opencode inside the container. The eval container is ephemeral and has no access to user data beyond the mounted repo.

**Environment variables**:
- `GOOGLE_GENERATIVE_AI_API_KEY` — passed through from host (or use provider-specific key env vars)
- `CONTAINER_HOST` — tells kantra where to find podman
- `OPENCODE_MODEL` — (optional) override model, e.g. `google/gemini-2.5-pro`

### Step 3: Present results

The eval grader (`cmd/eval`) prints JSON to stdout. The report groups checks by agent (rule-writer, test-generator, validator, pipeline) with per-agent and overall pass rates.

Highlight:
- Per-agent pass rates
- Any P0 failures (these cause non-zero exit)
- Missing golden patterns (rw-003 details)
- Duplicate patterns (rw-005 details)

Print pipeline log path (`output/pipeline.log`) for debugging failures.

## Examples

### Basic eval with golden set

```
Input:
  guide: https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-3.0-Migration-Guide
  golden_set: eval/golden/spring-boot-3-to-4.yaml

Output:
  eval_report: { ... }
  pass_rate: 87.5
  pipeline_log: eval/output/pipeline.log
```

### Eval without golden set (extraction-only)

```
Input:
  guide: https://example.com/new-migration.html

Output:
  eval_report: { "grading": null, "patterns_extracted": 42 }
  pass_rate: N/A
  pipeline_log: eval/output/pipeline.log
```

### Override model

```bash
podman run --rm \
  -v $(pwd):$(pwd) -w $(pwd) \
  -v "$PODMAN_SOCK":/run/podman/podman.sock \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -e GOOGLE_GENERATIVE_AI_API_KEY="$GOOGLE_GENERATIVE_AI_API_KEY" \
  -e OPENCODE_MODEL=google/gemini-2.5-pro \
  ai-rule-gen-eval \
  bash eval/run.sh "https://example.com/guide.html" "eval/golden/set.yaml"
```

## Troubleshooting

**Container build fails**: Check Containerfile syntax, ensure podman is installed.

**Pipeline run fails**: Check `eval/output/pipeline.log`. Common causes:
- Missing GOOGLE_GENERATIVE_AI_API_KEY
- Podman socket not detected (check `/run/podman/podman.sock` and `$XDG_RUNTIME_DIR`)
- kantra image pull failure (check internet connection)

**Eval report missing**: Pipeline may have failed before eval step. Check pipeline.log.

**Low pass rate**: Review missing_patterns, p0_failures. May indicate:
- Guide too complex for current extraction
- Golden set expectations too strict
- Model needs tuning
