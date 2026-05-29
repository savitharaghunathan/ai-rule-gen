#!/usr/bin/env bash
set -euo pipefail

VALID_RUNTIMES="claude-code opencode goose"
VALID_MODELS="sonnet opus haiku gemini-pro"
VALID_MIGRATIONS="httpclient4-to-httpclient5 spring-boot3-to-spring-boot4"

# --- Model ID mapping ---
model_id() {
    local model="$1"
    case "$model" in
        sonnet)     echo "claude-sonnet-4-6" ;;
        opus)       echo "claude-opus-4-6" ;;
        haiku)      echo "claude-haiku-4-5-20251001" ;;
        gemini-pro) echo "gemini-2.5-pro" ;;

        *)          echo "$model" ;;
    esac
}

# --- Guide URL lookup ---
guide_url() {
    local migration="$1"
    grep "^guide_url:" "evals/${migration}/eval_config.yaml" | awk '{print $2}'
}

usage() {
    echo "Usage: $0 <runtime> <model> <migration>"
    echo ""
    echo "Runs the full /generate-rules pipeline with the specified runtime and model,"
    echo "then collects benchmark results (eval metrics, rules, timing)."
    echo ""
    echo "Arguments:"
    echo "  runtime    Agent runtime: claude-code, opencode, goose"
    echo "  model      LLM model: sonnet, opus, gemini-pro"
    echo "  migration  Migration name: httpclient4-to-httpclient5, spring-boot3-to-spring-boot4"
    echo ""
    echo "Environment variables (for non-Claude runtimes):"
    echo "  ANTHROPIC_API_KEY   Required for sonnet/opus on opencode/goose"
    echo "  GOOGLE_API_KEY      Required for gemini-pro on opencode/goose"
    echo ""
    echo ""
    echo "Example:"
    echo "  $0 claude-code sonnet httpclient4-to-httpclient5"
    echo "  GOOGLE_API_KEY=... $0 opencode gemini-pro spring-boot3-to-spring-boot4"
    exit 1
}

if [ $# -lt 3 ]; then
    usage
fi

RUNTIME="$1"
MODEL="$2"
MIGRATION="$3"

# --- Validate inputs ---

if ! echo "$VALID_RUNTIMES" | grep -qw "$RUNTIME"; then
    echo "Error: invalid runtime '$RUNTIME'. Must be one of: $VALID_RUNTIMES"
    exit 1
fi

if ! echo "$VALID_MODELS" | grep -qw "$MODEL"; then
    echo "Error: invalid model '$MODEL'. Must be one of: $VALID_MODELS"
    exit 1
fi

if ! echo "$VALID_MIGRATIONS" | grep -qw "$MIGRATION"; then
    echo "Error: invalid migration '$MIGRATION'. Must be one of: $VALID_MIGRATIONS"
    exit 1
fi

if [ "$RUNTIME" = "claude-code" ] && [ "$MODEL" != "sonnet" ] && [ "$MODEL" != "opus" ] && [ "$MODEL" != "haiku" ]; then
    echo "Error: claude-code only supports sonnet, opus, and haiku models"
    exit 1
fi

if ! command -v jq &>/dev/null; then
    echo "Error: jq is required but not installed. Install with: brew install jq"
    exit 1
fi

MODEL_ID=$(model_id "$MODEL")
GUIDE_URL=$(guide_url "$MIGRATION")

if [ -z "$GUIDE_URL" ]; then
    echo "Error: could not find guide_url in evals/${MIGRATION}/eval_config.yaml"
    exit 1
fi

BENCHMARK_DIR="benchmarks/${MIGRATION}/${RUNTIME}--${MODEL}"
GROUND_TRUTH="evals/${MIGRATION}/ground_truth.yaml"
EVAL_OUTPUT="/tmp/benchmark-eval-$$.json"

SKILL_PROMPT="Read agents/generate-rules/SKILL.md and follow its instructions. Input: { guide_source: '${GUIDE_URL}', mode: 'non_interactive', checkpoint_behavior: 'continue' }"

echo "============================================"
echo "Benchmark Run"
echo "============================================"
echo "Runtime:   $RUNTIME"
echo "Model:     $MODEL ($MODEL_ID)"
echo "Migration: $MIGRATION"
echo "Guide:     $GUIDE_URL"
echo "Output:    $BENCHMARK_DIR"
echo "============================================"
echo ""

# --- Record start time and find newest output dir before run ---

START_TIME=$(date +%s)
MARKER_FILE="/tmp/benchmark-marker-$$.tmp"
touch "$MARKER_FILE"

# --- Invoke the runtime ---

echo "Starting pipeline at $(date '+%Y-%m-%d %H:%M:%S') ..."
echo ""

case "$RUNTIME" in
    claude-code)
        claude -p "$SKILL_PROMPT" --model "$MODEL_ID"
        ;;
    opencode)
        ANTHROPIC_MODEL="$MODEL_ID" \
        opencode -p "$SKILL_PROMPT"
        ;;
    goose)
        goose run --text "$SKILL_PROMPT"
        ;;
esac

END_TIME=$(date +%s)
DURATION_SECONDS=$((END_TIME - START_TIME))
DURATION_MINUTES=$(echo "scale=1; $DURATION_SECONDS / 60" | bc)

echo ""
echo "Pipeline completed in ${DURATION_MINUTES} minutes."
echo ""

# --- Find the output directory (newest dir created during the run) ---

OUTPUT_DIR=$(find output -maxdepth 1 -type d -newer "$MARKER_FILE" | sort | tail -1)
rm -f "$MARKER_FILE"

if [ -z "$OUTPUT_DIR" ]; then
    echo "Error: no new output directory found after pipeline run."
    echo "Available directories:"
    ls -lt output/ | head -10
    echo ""
    echo "Enter the output directory path manually:"
    read -r OUTPUT_DIR
fi

if [ ! -d "$OUTPUT_DIR/rules" ]; then
    echo "Error: $OUTPUT_DIR/rules does not exist. Pipeline may have failed."
    exit 1
fi

echo "Found output directory: $OUTPUT_DIR"

# --- Check for report.yaml ---

if [ ! -f "$OUTPUT_DIR/report.yaml" ]; then
    echo "Warning: $OUTPUT_DIR/report.yaml does not exist. Pipeline may have failed before testing."
    echo "Proceeding with eval only (pass rate will be 0/0)."
    TESTS_PASSED=0
    TESTS_FAILED=0
    KANTRA_LIM=0
else
    TESTS_PASSED=$(grep "^tests_passed:" "$OUTPUT_DIR/report.yaml" | awk '{print $2}')
    TESTS_FAILED=$(grep "^tests_failed:" "$OUTPUT_DIR/report.yaml" | awk '{print $2}')
    KANTRA_LIM=$(grep "^kantra_limitation:" "$OUTPUT_DIR/report.yaml" | awk '{print $2}')
fi

TESTED=$((TESTS_PASSED + TESTS_FAILED + KANTRA_LIM))
PASS_RATE="${TESTS_PASSED}/${TESTED}"

# --- Run full eval skill (deterministic + LLM judge) ---

MARKER_FILE_EVAL="/tmp/benchmark-marker-eval-$$.tmp"
touch "$MARKER_FILE_EVAL"

EVAL_PROMPT="Read agents/eval/SKILL.md and follow its instructions. Input: { guide_source: '${GUIDE_URL}', rules_dir: '${OUTPUT_DIR}/rules', migration: '${MIGRATION}' }"

echo "Running full eval (deterministic + LLM judge) on ${OUTPUT_DIR}/rules ..."
echo ""

case "$RUNTIME" in
    claude-code)
        claude -p "$EVAL_PROMPT" --model "$MODEL_ID"
        ;;
    opencode)
        ANTHROPIC_MODEL="$MODEL_ID" \
        opencode -p "$EVAL_PROMPT"
        ;;
    goose)
        goose run --text "$EVAL_PROMPT"
        ;;
esac

echo ""
echo "Eval skill complete."

# --- Also run deterministic eval for structured metrics ---

echo "Running deterministic eval for metrics ..."

if [ -f "$GROUND_TRUTH" ]; then
    go run ./cmd/eval \
        --rules-dir "${OUTPUT_DIR}/rules" \
        --ground-truth "$GROUND_TRUTH" \
        --migration "$MIGRATION" \
        2>/dev/null > "$EVAL_OUTPUT"
else
    go run ./cmd/eval \
        --rules-dir "${OUTPUT_DIR}/rules" \
        --migration "$MIGRATION" \
        2>/dev/null > "$EVAL_OUTPUT"
fi

echo "Deterministic eval complete."

# --- Extract metrics from eval JSON ---

RULE_COUNT=$(jq '.rule_count' "$EVAL_OUTPUT")
QUALITY_AVG=$(jq '.quality.avg_score' "$EVAL_OUTPUT")
OVERLAP_COUNT=$(jq '.overlaps | length' "$EVAL_OUTPUT")

COVERAGE_PCT=$(jq '.app_coverage.effective_coverage_pct // empty' "$EVAL_OUTPUT" 2>/dev/null || echo "null")
if [ "$COVERAGE_PCT" = "" ]; then
    COVERAGE_PCT="null"
fi

SPEC_GAP_COUNT=$(jq '.specificity_gaps // [] | length' "$EVAL_OUTPUT" 2>/dev/null || echo "0")

# --- Copy eval skill findings if they exist ---

EVAL_FINDINGS=$(find /tmp -maxdepth 1 -name "eval-*" -type d -newer "$MARKER_FILE_EVAL" 2>/dev/null | sort | tail -1)

# --- Create benchmark directory and copy artifacts ---

echo "Copying artifacts to ${BENCHMARK_DIR} ..."
mkdir -p "$BENCHMARK_DIR"
cp -r "${OUTPUT_DIR}/rules" "${BENCHMARK_DIR}/rules"
cp "$EVAL_OUTPUT" "${BENCHMARK_DIR}/eval-snapshot.json"

if [ -n "$EVAL_FINDINGS" ] && [ -f "$EVAL_FINDINGS/findings.json" ]; then
    cp "$EVAL_FINDINGS/findings.json" "${BENCHMARK_DIR}/findings.json"
    echo "Copied eval findings from $EVAL_FINDINGS"
fi
rm -f "$MARKER_FILE_EVAL"

# --- Write result.json ---

TODAY=$(date +%Y-%m-%d)

cat > "${BENCHMARK_DIR}/result.json" <<RESULTEOF
{
  "runtime": "${RUNTIME}",
  "model": "${MODEL}",
  "model_id": "${MODEL_ID}",
  "migration": "${MIGRATION}",
  "date": "${TODAY}",
  "duration_minutes": ${DURATION_MINUTES},
  "rule_count": ${RULE_COUNT},
  "quality_avg": ${QUALITY_AVG},
  "effective_coverage_pct": ${COVERAGE_PCT},
  "overlap_conflict_count": ${OVERLAP_COUNT},
  "specificity_gap_count": ${SPEC_GAP_COUNT},
  "pass_rate": "${PASS_RATE}",
  "notes": ""
}
RESULTEOF

echo "Result written to ${BENCHMARK_DIR}/result.json"

# --- Clean up temp file ---

rm -f "$EVAL_OUTPUT"

# --- Regenerate README ---

echo "Regenerating benchmarks/README.md ..."

generate_table() {
    local migration="$1"
    local dir="benchmarks/${migration}"

    echo "| Runtime | Model | Rules | Pass Rate | Quality Avg | Coverage | Overlaps | Time (min) |"
    echo "|---------|-------|-------|-----------|-------------|----------|----------|------------|"

    for rt in claude-code opencode goose; do
        local models_for_rt
        if [ "$rt" = "claude-code" ]; then
            models_for_rt="sonnet opus haiku"
        else
            models_for_rt="sonnet opus gemini-pro"
        fi

        for mdl in $models_for_rt; do
            local result_file="${dir}/${rt}--${mdl}/result.json"
            if [ -f "$result_file" ]; then
                local rules qual cov overlaps pass_rate duration
                rules=$(jq -r '.rule_count' "$result_file")
                qual=$(jq -r '.quality_avg | . * 100 | round / 100' "$result_file")
                cov=$(jq -r 'if .effective_coverage_pct == null then "—" else (.effective_coverage_pct | tostring) + "%" end' "$result_file")
                overlaps=$(jq -r '.overlap_conflict_count' "$result_file")
                pass_rate=$(jq -r '.pass_rate' "$result_file")
                duration=$(jq -r 'if .duration_minutes == null then "—" else (.duration_minutes | tostring) end' "$result_file")
                echo "| ${rt} | ${mdl} | ${rules} | ${pass_rate} | ${qual} | ${cov} | ${overlaps} | ${duration} |"
            else
                echo "| ${rt} | ${mdl} | — | — | — | — | — | — |"
            fi
        done
    done
}

cat > benchmarks/README.md <<'HEADEREOF'
# Benchmark Results

Comparison of rule generation quality across agent runtimes and LLM models.

## Methodology

- **Pipeline**: `/generate-rules` skill invoked with the same migration guide URL
- **Evaluation**: `cmd/eval` with japicmp-derived ground truth
- **Timing**: Wall-clock from pipeline start to report completion
- **Date**: May–June 2026

## Runtime × Model Matrix

| Runtime | Models |
|---------|--------|
| Claude Code | Sonnet, Opus |
| OpenCode | Sonnet, Opus, Gemini Pro |
| Goose | Sonnet, Opus, Gemini Pro |

HEADEREOF

echo "## httpclient4-to-httpclient5" >> benchmarks/README.md
echo "" >> benchmarks/README.md
generate_table "httpclient4-to-httpclient5" >> benchmarks/README.md
echo "" >> benchmarks/README.md

echo "## spring-boot3-to-spring-boot4" >> benchmarks/README.md
echo "" >> benchmarks/README.md
generate_table "spring-boot3-to-spring-boot4" >> benchmarks/README.md
echo "" >> benchmarks/README.md

cat >> benchmarks/README.md <<'FOOTEREOF'
## How to Reproduce

### Prerequisites

- Go 1.25+
- `jq` command-line JSON processor
- One or more agent runtimes installed: Claude Code, OpenCode, Goose

### Running a Benchmark

```bash
./scripts/benchmark-collect.sh <runtime> <model> <migration>
```

The script will:
1. Invoke the agent runtime with the specified model
2. Run the full `/generate-rules` pipeline
3. Time the entire run
4. Run deterministic eval against ground truth
5. Collect rules, eval snapshot, and metrics into `benchmarks/`
6. Regenerate the comparison table

### Examples

```bash
# Claude Code with Sonnet
./scripts/benchmark-collect.sh claude-code sonnet httpclient4-to-httpclient5

# OpenCode with Gemini Pro
GOOGLE_API_KEY=... ./scripts/benchmark-collect.sh opencode gemini-pro spring-boot3-to-spring-boot4

```

### Runtime Setup

#### Claude Code
- Models: `sonnet` → claude-sonnet-4-6, `opus` → claude-opus-4-6
- No extra env vars needed (uses your Claude Code auth)

#### OpenCode
- Set `ANTHROPIC_API_KEY` for sonnet/opus
- Set `GOOGLE_API_KEY` for gemini-pro

#### Goose
- Set API keys same as OpenCode, or use `goose configure`

### Migration Guide URLs

| Migration | Guide URL |
|-----------|-----------|
| httpclient4-to-httpclient5 | https://hc.apache.org/httpcomponents-client-5.6.x/migration-guide |
| spring-boot3-to-spring-boot4 | https://github.com/spring-projects/spring-boot/wiki/Spring-Boot-4.0-Migration-Guide |
FOOTEREOF

echo ""
echo "============================================"
echo "Benchmark Complete"
echo "============================================"
echo "Runtime:   $RUNTIME"
echo "Model:     $MODEL ($MODEL_ID)"
echo "Migration: $MIGRATION"
echo "Duration:  ${DURATION_MINUTES} min"
echo "Rules:     $RULE_COUNT"
echo "Pass rate: $PASS_RATE"
echo "Quality:   $QUALITY_AVG"
echo "Results:   $BENCHMARK_DIR/result.json"
echo "============================================"
