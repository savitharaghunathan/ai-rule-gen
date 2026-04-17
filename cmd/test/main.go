package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/testrunner"
)

func main() {
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	testsDir := flag.String("tests", "", "Path to tests directory containing .test.yaml files (required)")
	filesFlag := flag.String("files", "", "Comma-separated list of specific .test.yaml files to run (optional, overrides --tests scan)")
	flag.Parse()

	if *rulesDir == "" || *testsDir == "" {
		fmt.Fprintln(os.Stderr, "error: --rules and --tests are required")
		os.Exit(1)
	}

	cfg := testrunner.Config{
		RulesDir: *rulesDir,
		TestsDir: *testsDir,
	}

	if *filesFlag != "" {
		for _, f := range strings.Split(*filesFlag, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				cfg.Files = append(cfg.Files, f)
			}
		}
	}

	result, err := testrunner.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(result)
}
