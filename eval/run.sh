#!/usr/bin/env bash
set -euo pipefail

# eval/run.sh — container entry script for eval harness
# Usage: eval/run.sh <guide-source> [golden-set]

# Parse args
if [[ $# -lt 1 ]]; then
  echo "[eval] ERROR: missing required argument <guide-source>" >&2
  echo "Usage: $0 <guide-source> [golden-set]" >&2
  exit 1
fi

GUIDE_SOURCE="$1"
GOLDEN_SET="${2:-}"

echo "[eval] Starting pipeline with guide: $GUIDE_SOURCE"
[[ -n "$GOLDEN_SET" ]] && echo "[eval] Golden set: $GOLDEN_SET"

# Create output directory
echo "[eval] Creating output/ directory"
mkdir -p output

# Run the pipeline
MODEL="${OPENCODE_MODEL:-google/gemini-2.5-flash}"
echo "[eval] Model: $MODEL"
echo "[eval] Running generate-rules pipeline..."
opencode run -m "$MODEL" --dangerously-skip-permissions \
  "Read and follow agents/generate-rules/SKILL.md. Input: $GUIDE_SOURCE. When you reach the checkpoint, continue automatically." \
  | tee output/pipeline.log

echo "[eval] Pipeline complete"

# Check that required outputs exist
echo "[eval] Validating pipeline outputs..."
if [[ ! -d output/rules ]]; then
  echo "[eval] ERROR: output/rules/ directory not found" >&2
  exit 1
fi

if [[ ! -f output/rules/patterns.json ]]; then
  echo "[eval] ERROR: output/rules/patterns.json not found" >&2
  exit 1
fi

echo "[eval] Pipeline outputs validated"

# Build eval args
EVAL_ARGS="--rules output/rules"

if [[ -n "$GOLDEN_SET" ]]; then
  EVAL_ARGS="$EVAL_ARGS --golden $GOLDEN_SET"
fi

if [[ -f output/report.yaml ]]; then
  EVAL_ARGS="$EVAL_ARGS --report output/report.yaml"
fi

if [[ -f output/pre-fix-report.yaml ]]; then
  EVAL_ARGS="$EVAL_ARGS --pre-fix-report output/pre-fix-report.yaml"
fi

# Run eval
echo "[eval] Running eval harness with args: $EVAL_ARGS"
go run ./cmd/eval $EVAL_ARGS

echo "[eval] Eval complete"
