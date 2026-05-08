package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/testrunner"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	testsDir := flag.String("tests", "", "Path to tests directory containing .test.yaml files (required)")
	filesFlag := flag.String("files", "", "Comma-separated list of specific .test.yaml files to run (optional, overrides --tests scan)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Per-test-file timeout (e.g., 5m, 3m30s). 0 = no timeout")
	retryTimeouts := flag.Bool("retry-timeouts", true, "Automatically retry timed-out test files once")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *rulesDir == "" || *testsDir == "" {
		cli.Fail("invalid_arguments", "--rules and --tests are required", "test", "provide rules and tests directories", nil)
	}

	cfg := testrunner.Config{
		RulesDir:      *rulesDir,
		TestsDir:      *testsDir,
		TestTimeout:   *timeout,
		RetryTimeouts: *retryTimeouts,
		OnProgress: func(file string, index, fileCount, passed, total int, timedOut bool, elapsed time.Duration) {
			prefix := fmt.Sprintf("[%d/%d]", index, fileCount)
			if passed == -1 {
				fmt.Fprintf(os.Stderr, "  %-7s %s...\n", prefix, file)
				cli.Log("%s %s...", prefix, file)
			} else if timedOut {
				fmt.Fprintf(os.Stderr, "  %-7s %-40s TIMED OUT (%s)\n", prefix, file, elapsed.Truncate(time.Second))
				cli.Log("%s %-40s TIMED OUT (%s)", prefix, file, elapsed.Truncate(time.Second))
			} else {
				fmt.Fprintf(os.Stderr, "  %-7s %-40s %d/%d passed (%s)\n", prefix, file, passed, total, elapsed.Truncate(time.Second))
				cli.Log("%s %-40s %d/%d passed (%s)", prefix, file, passed, total, elapsed.Truncate(time.Second))
			}
		},
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
