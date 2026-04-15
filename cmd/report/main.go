package main

import (
	"flag"
	"fmt"
	"os"
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
		fmt.Fprintln(os.Stderr, "error: --output is required")
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(report)
}
