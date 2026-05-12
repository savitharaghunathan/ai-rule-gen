package main

import (
	"flag"
	"strings"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	kantraOutput := flag.String("kantra-output", "", "Raw kantra test output to parse")
	passedFlag := flag.String("passed", "", "Comma-separated list of passed rule IDs")
	failedFlag := flag.String("failed", "", "Comma-separated list of failed rule IDs")
	kantraLimitationFlag := flag.String("kantra-limitation", "", "Comma-separated list of kantra limitation rule IDs")
	verifiedFlag := flag.String("verified", "", "Comma-separated list of source-verified rule IDs")
	notFoundFlag := flag.String("not-found", "", "Comma-separated list of source-not-found rule IDs")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

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

	var verifiedIDs, notFoundIDs []string
	if *verifiedFlag != "" {
		verifiedIDs = splitCSV(*verifiedFlag)
	}
	if *notFoundFlag != "" {
		notFoundIDs = splitCSV(*notFoundFlag)
	}

	hasTestResults := len(passedIDs) > 0 || len(failedIDs) > 0 || len(kantraLimitationIDs) > 0
	hasVerifyResults := len(verifiedIDs) > 0 || len(notFoundIDs) > 0

	if !hasTestResults && !hasVerifyResults {
		cli.Fail("invalid_arguments", "no results to stamp: provide --kantra-output, --passed/--failed/--kantra-limitation, or --verified/--not-found", "stamp", "pass test results or verification results", nil)
	}

	if hasTestResults {
		if err := rules.StampTestResults(*rulesDir, passedIDs, failedIDs, kantraLimitationIDs); err != nil {
			cli.Fail("stamp_failed", err.Error(), "stamp", "check rules directory write permissions and rule file format", map[string]string{"rules": *rulesDir})
		}
	}

	if hasVerifyResults {
		if err := rules.StampVerificationResults(*rulesDir, verifiedIDs, notFoundIDs); err != nil {
			cli.Fail("stamp_verify_failed", err.Error(), "stamp", "check rules directory write permissions and rule file format", map[string]string{"rules": *rulesDir})
		}
	}

	cli.WriteJSON(map[string]interface{}{
		"status":            "ok",
		"passed":            len(passedIDs),
		"failed":            len(failedIDs),
		"kantra_limitation": len(kantraLimitationIDs),
		"verified":          len(verifiedIDs),
		"not_found":         len(notFoundIDs),
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
