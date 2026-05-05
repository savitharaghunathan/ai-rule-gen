package main

import (
	"flag"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/testrunner"
)

func main() {
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	testsDir := flag.String("tests", "", "Path to tests directory containing .test.yaml files (required)")
	filesFlag := flag.String("files", "", "Comma-separated list of specific .test.yaml files to run (optional, overrides --tests scan)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Per-test-file timeout (e.g., 5m, 3m30s). 0 = no timeout")
	retryTimeouts := flag.Bool("retry-timeouts", true, "Automatically retry timed-out test files once")
	flag.Parse()

	if *rulesDir == "" || *testsDir == "" {
		cli.Fail("invalid_arguments", "--rules and --tests are required", "test", "provide rules and tests directories", nil)
	}

	cfg := testrunner.Config{
		RulesDir:      *rulesDir,
		TestsDir:      *testsDir,
		TestTimeout:   *timeout,
		RetryTimeouts: *retryTimeouts,
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
		cli.Fail("test_runner_failed", err.Error(), "test", "verify tests directory, kantra availability, and rule validity", map[string]interface{}{"rules": *rulesDir, "tests": *testsDir, "files": cfg.Files})
	}

	cli.WriteJSON(result)
}
