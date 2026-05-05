package main

import (
	"flag"
	"strings"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

func main() {
	source := flag.String("source", "", "Source technology")
	target := flag.String("target", "", "Target technology")
	output := flag.String("output", "", "Output report file path (required)")
	rulesTotal := flag.Int("rules-total", 0, "Total number of rules")
	passed := flag.Int("passed", 0, "Number of tests passed")
	failed := flag.Int("failed", 0, "Number of tests failed")
	failedRulesFlag := flag.String("failed-rules", "", "Comma-separated list of failed rule IDs")
	flag.Parse()

	if *output == "" {
		cli.Fail("invalid_arguments", "--output is required", "report", "set --output to a writable report.yaml path", nil)
	}

	var failedRules []string
	if *failedRulesFlag != "" {
		for _, item := range strings.Split(*failedRulesFlag, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				failedRules = append(failedRules, item)
			}
		}
	}

	report := workspace.BuildReport(*source, *target, *rulesTotal, *passed, *failed, failedRules)
	if err := workspace.WriteReport(*output, report); err != nil {
		cli.Fail("write_report_failed", err.Error(), "report", "verify output directory exists and is writable", map[string]string{"output": *output})
	}

	cli.WriteJSON(report)
}
