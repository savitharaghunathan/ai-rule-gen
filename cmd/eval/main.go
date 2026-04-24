package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/eval"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

func main() {
	goldenPath := flag.String("golden", "", "Path to golden set YAML file")
	patternsPath := flag.String("patterns", "", "Path to patterns.json")
	rulesPath := flag.String("rules", "", "Path to rules directory")
	reportPath := flag.String("report", "", "Path to report.yaml (post-fix)")
	preFixReportPath := flag.String("pre-fix-report", "", "Path to pre-fix report.yaml")
	rulesSnapshotPath := flag.String("rules-snapshot", "", "Path to rules snapshot directory (pre-fix)")
	flag.Parse()

	if *patternsPath == "" && *rulesPath == "" {
		fmt.Fprintln(os.Stderr, "error: at least one of --patterns or --rules is required")
		os.Exit(1)
	}

	if *patternsPath == "" && *rulesPath != "" {
		defaultPatterns := *rulesPath + "/patterns.json"
		if _, err := os.Stat(defaultPatterns); err == nil {
			*patternsPath = defaultPatterns
		}
	}

	ctx := &eval.EvalContext{}

	if *goldenPath != "" {
		gs, err := eval.LoadGoldenSet(*goldenPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading golden set: %v\n", err)
			os.Exit(1)
		}
		ctx.Golden = gs
	}

	if *patternsPath != "" {
		patterns, err := rules.ReadPatternsFile(*patternsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading patterns: %v\n", err)
			os.Exit(1)
		}
		ctx.Patterns = patterns
	}

	if *rulesPath != "" {
		ruleList, err := rules.ReadRulesDir(*rulesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading rules: %v\n", err)
			os.Exit(1)
		}
		ctx.Rules = ruleList
	}

	if *reportPath != "" {
		data, err := os.ReadFile(*reportPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading report: %v\n", err)
			os.Exit(1)
		}
		report, err := workspace.ParseReport(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing report: %v\n", err)
			os.Exit(1)
		}
		ctx.Report = report
	}

	if *preFixReportPath != "" {
		data, err := os.ReadFile(*preFixReportPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading pre-fix report: %v\n", err)
			os.Exit(1)
		}
		report, err := workspace.ParseReport(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing pre-fix report: %v\n", err)
			os.Exit(1)
		}
		ctx.PreFixReport = report
	}

	if *rulesSnapshotPath != "" {
		ruleList, err := rules.ReadRulesDir(*rulesSnapshotPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading rules snapshot: %v\n", err)
			os.Exit(1)
		}
		ctx.RulesSnapshot = ruleList
	}

	result := eval.RunAll(ctx)
	cli.WriteJSON(result)

	for _, ae := range result.Agents {
		for _, c := range ae.Checks {
			if !c.Passed && c.Priority == "P0" {
				os.Exit(1)
			}
		}
	}
}
