package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/eval"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	rulesDir := flag.String("rules-dir", "", "Path to generated rules directory (required)")
	appDir := flag.String("app-dir", "", "Path to app for kantra analyze coverage check")
	save := flag.Bool("save", false, "Save results to evals/<migration>/runs/<timestamp>.json")
	saveBaseline := flag.Bool("save-baseline", false, "Save results as evals/<migration>/det_baseline.json")
	comparePath := flag.String("compare", "", "Path to baseline snapshot for regression comparison")
	migration := flag.String("migration", "", "Migration name (inferred from rules-dir if not set)")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *rulesDir == "" {
		cli.Fail("invalid_arguments", "--rules-dir is required", "eval", "provide a rules directory", nil)
	}

	cfg := eval.Config{
		RulesDir: *rulesDir,
		AppDir:   *appDir,
	}

	result, err := eval.RunEval(cfg)
	if err != nil {
		cli.Fail("eval_failed", err.Error(), "eval", "check rules directory and kantra availability", nil)
	}

	eval.PrintReport(result)

	migrationName := *migration
	if migrationName == "" {
		migrationName = inferMigration(*rulesDir)
	}

	if (*save || *saveBaseline) && migrationName != "" {
		snapshot := eval.SnapshotFromResult(result, migrationName)

		if *save {
			ts := time.Now().UTC().Format("20060102-150405")
			runPath := filepath.Join("evals", migrationName, "runs", ts+".json")
			if err := eval.SaveSnapshot(snapshot, runPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save run: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Run saved: %s\n", runPath)
			}
		}

		if *saveBaseline {
			baselinePath := filepath.Join("evals", migrationName, "det_baseline.json")
			if err := eval.SaveSnapshot(snapshot, baselinePath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save baseline: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Baseline saved: %s\n", baselinePath)
			}
		}
	}

	if *comparePath != "" {
		baseline, err := eval.LoadSnapshot(*comparePath)
		if err != nil {
			cli.Fail("compare_failed", err.Error(), "compare", "check baseline path", nil)
		}

		thresholds := eval.DefaultThresholds()
		if migrationName != "" {
			configPath := filepath.Join("evals", migrationName, "eval_config.yaml")
			if ecfg, err := eval.LoadEvalConfig(configPath); err == nil {
				thresholds = ecfg.ResolvedThresholds()
			}
		}

		snapshot := eval.SnapshotFromResult(result, migrationName)
		cr := eval.Compare(snapshot, baseline, thresholds)
		eval.PrintCompare(cr, migrationName)

		if cr.Verdict == "FAIL" {
			os.Exit(1)
		}
	}

	cli.WriteJSON(result)
}

func inferMigration(rulesDir string) string {
	dir := filepath.Clean(rulesDir)
	parts := strings.Split(dir, string(filepath.Separator))
	for i, p := range parts {
		if p == "evals" && i+1 < len(parts) {
			return parts[i+1]
		}
		if p == "output" && i+1 < len(parts) {
			name := parts[i+1]
			if idx := strings.LastIndex(name, "-"); idx > 0 {
				if _, err := time.Parse("20060102", name[idx+1:]); err == nil {
					name = name[:idx]
				}
			}
			return name
		}
	}
	return ""
}
