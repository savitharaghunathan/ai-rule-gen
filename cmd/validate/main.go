package main

import (
	"flag"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	rulesPath := flag.String("rules", "", "Path to rules directory or file (required)")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *rulesPath == "" {
		cli.Fail("invalid_arguments", "--rules is required", "validate", "set --rules to a rules directory or file path", nil)
	}

	info, err := os.Stat(*rulesPath)
	if err != nil {
		cli.Fail("rules_path_unavailable", err.Error(), "validate", "confirm the rules path exists and is readable", map[string]string{"rules": *rulesPath})
	}

	var ruleList []rules.Rule
	if info.IsDir() {
		ruleList, err = rules.ReadRulesDir(*rulesPath)
	} else {
		ruleList, err = rules.ReadRulesFile(*rulesPath)
	}
	if err != nil {
		cli.Fail("read_rules_failed", err.Error(), "validate", "verify rule YAML syntax and schema fields", map[string]string{"rules": *rulesPath})
	}

	result := rules.Validate(ruleList)
	cli.WriteJSON(result)
	if !result.Valid {
		os.Exit(1)
	}
}
