package main

import (
	"flag"
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
	kantraLimitationFlag := flag.String("kantra-limitation", "", "Comma-separated list of kantra limitation rule IDs")
	flag.Parse()

	if *rulesDir == "" {
		cli.Fail("invalid_arguments", "--rules is required", "stamp", "set --rules to a directory containing rule YAML files", nil)
	}

	var passedIDs, failedIDs, kantraLimitationIDs []string

	if *kantraOutput != "" {
		allRules, err := rules.ReadRulesDir(*rulesDir)
		if err != nil {
			cli.Fail("read_rules_failed", err.Error(), "stamp", "verify rules directory path and rule file validity", map[string]string{"rules": *rulesDir})
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
	if *kantraLimitationFlag != "" {
		kantraLimitationIDs = splitCSV(*kantraLimitationFlag)
	}

	if len(passedIDs) == 0 && len(failedIDs) == 0 && len(kantraLimitationIDs) == 0 {
		cli.Fail("invalid_arguments", "no results to stamp: provide --kantra-output or --passed/--failed/--kantra-limitation", "stamp", "pass test results either as raw kantra output or explicit passed/failed/kantra-limitation IDs", nil)
	}

	if err := rules.StampTestResults(*rulesDir, passedIDs, failedIDs, kantraLimitationIDs); err != nil {
		cli.Fail("stamp_failed", err.Error(), "stamp", "check rules directory write permissions and rule file format", map[string]string{"rules": *rulesDir})
	}

	cli.WriteJSON(map[string]interface{}{
		"status":            "ok",
		"passed":            len(passedIDs),
		"failed":            len(failedIDs),
		"kantra_limitation": len(kantraLimitationIDs),
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
