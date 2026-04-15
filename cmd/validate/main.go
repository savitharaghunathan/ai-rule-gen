package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	rulesPath := flag.String("rules", "", "Path to rules directory or file (required)")
	flag.Parse()

	if *rulesPath == "" {
		fmt.Fprintln(os.Stderr, "error: --rules is required")
		os.Exit(1)
	}

	info, err := os.Stat(*rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot access %s: %v\n", *rulesPath, err)
		os.Exit(1)
	}

	var ruleList []rules.Rule
	if info.IsDir() {
		ruleList, err = rules.ReadRulesDir(*rulesPath)
	} else {
		ruleList, err = rules.ReadRulesFile(*rulesPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result := rules.Validate(ruleList)
	cli.WriteJSON(result)
	if !result.Valid {
		os.Exit(1)
	}
}
