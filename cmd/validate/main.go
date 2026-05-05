package main

import (
	"flag"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	rulesPath := flag.String("rules", "", "Path to rules directory or file (required)")
	flag.Parse()

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
