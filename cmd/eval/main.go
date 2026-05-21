package main

import (
	"flag"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/eval"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	rulesDir := flag.String("rules-dir", "", "Path to generated rules directory (required)")
	appDir := flag.String("app-dir", "", "Path to app for kantra analyze coverage check")
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
	cli.WriteJSON(result)
}
