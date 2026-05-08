package main

import (
	"flag"
	"strings"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	source := flag.String("source", "", "Source technology")
	target := flag.String("target", "", "Target technology")
	output := flag.String("output", "", "Output report file path (required)")
	rulesTotal := flag.Int("rules-total", 0, "Total number of rules")
	passed := flag.Int("passed", 0, "Number of tests passed")
	failed := flag.Int("failed", 0, "Number of tests failed")
	kantraLimitation := flag.Int("kantra-limitation", 0, "Number of kantra limitation rules (correct but not auto-testable)")
	failedRulesFlag := flag.String("failed-rules", "", "Comma-separated list of failed rule IDs")
	kantraLimitationRulesFlag := flag.String("kantra-limitation-rules", "", "Comma-separated list of kantra limitation rule IDs")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *output == "" {
		cli.Fail("invalid_arguments", "--output is required", "report", "set --output to a writable report.yaml path", nil)
	}

	failedRules := splitCSV(*failedRulesFlag)
	kantraLimitationRules := splitCSV(*kantraLimitationRulesFlag)

	report := workspace.BuildReport(*source, *target, *rulesTotal, *passed, *failed, *kantraLimitation, failedRules, kantraLimitationRules)
	if err := workspace.WriteReport(*output, report); err != nil {
		cli.Fail("write_report_failed", err.Error(), "report", "verify output directory exists and is writable", map[string]string{"output": *output})
	}

	cli.WriteJSON(report)
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
