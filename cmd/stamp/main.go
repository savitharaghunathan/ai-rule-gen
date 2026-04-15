package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	kantraOutput := flag.String("kantra-output", "", "Raw kantra test output to parse")
	passedFlag := flag.String("passed", "", "Comma-separated list of passed rule IDs")
	failedFlag := flag.String("failed", "", "Comma-separated list of failed rule IDs")
	flag.Parse()

	if *rulesDir == "" {
		fmt.Fprintln(os.Stderr, "error: --rules is required")
		os.Exit(1)
	}

	var passedIDs, failedIDs []string

	if *kantraOutput != "" {
		allRules, err := rules.ReadRulesDir(*rulesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: reading rules: %v\n", err)
			os.Exit(1)
		}
		var allIDs []string
		for _, r := range allRules {
			allIDs = append(allIDs, r.RuleID)
		}
		passedIDs, failedIDs = kantraparser.PassedAndFailed(*kantraOutput, allIDs)
	}

	if *passedFlag != "" {
		passedIDs = splitCSV(*passedFlag)
	}
	if *failedFlag != "" {
		failedIDs = splitCSV(*failedFlag)
	}

	if len(passedIDs) == 0 && len(failedIDs) == 0 {
		fmt.Fprintln(os.Stderr, "error: no results to stamp: provide --kantra-output or --passed/--failed")
		os.Exit(1)
	}

	if err := rules.StampTestResults(*rulesDir, passedIDs, failedIDs); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(map[string]interface{}{
		"status": "ok",
		"passed": len(passedIDs),
		"failed": len(failedIDs),
	})
}

func splitCSV(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
