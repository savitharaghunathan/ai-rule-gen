package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/feedback"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	outputDir := flag.String("output-dir", "output", "Base directory containing pipeline run subdirectories")
	migrationPath := flag.String("migration-path", "", "Filter to runs whose directory name contains this string")
	format := flag.String("format", "text", "Output format: text or json")
	minRuns := flag.Int("min-runs", 2, "Minimum run appearances before a pattern is flagged as recurring")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	dirs, err := feedback.DiscoverRuns(*outputDir, *migrationPath)
	if err != nil {
		cli.Fail("discover_failed", err.Error(), "feedback", "verify --output-dir exists", nil)
	}
	if len(dirs) == 0 {
		cli.Fail("no_runs", "no pipeline runs found", "feedback", "check --output-dir and --migration-path", nil)
	}

	runs := feedback.LoadRuns(dirs)
	if len(runs) == 0 {
		cli.Fail("no_valid_runs", "no runs with report.yaml found", "feedback", "ensure runs completed successfully", nil)
	}

	report := feedback.Analyze(runs, *minRuns)
	report.Recommendations = feedback.Recommend(report)

	switch *format {
	case "json":
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
	default:
		printText(report)
	}
}

func printText(r *feedback.FeedbackReport) {
	fmt.Println("# Feedback Report")
	fmt.Println()
	fmt.Printf("**Runs analyzed:** %d  \n", r.RunsAnalyzed)
	fmt.Printf("**Date range:** %s  \n", r.DateRange)
	fmt.Printf("**Migration paths:** %s  \n", strings.Join(r.MigrationPaths, ", "))
	fmt.Println()

	fmt.Println("## Overall Stats")
	fmt.Println()
	fmt.Println("| Metric | Value |")
	fmt.Println("|---|---|")
	fmt.Printf("| Runs | %d |\n", r.Overall.TotalRuns)
	fmt.Printf("| Avg pass rate | %.1f%% |\n", r.Overall.AveragePassRate)
	fmt.Printf("| Avg rules/run | %.1f |\n", r.Overall.AverageRulesPerRun)
	fmt.Printf("| Avg verify rate | %.1f%% |\n", r.Overall.AverageVerifyRate)
	fmt.Printf("| Trend | %s |\n", r.Overall.PassRateTrend)
	fmt.Println()

	fmt.Println("## Verification Analysis")
	fmt.Println()
	fmt.Printf("Verified: %d | Not found: %d | Skipped: %d\n\n",
		r.Verify.TotalVerified, r.Verify.TotalNotFound, r.Verify.TotalSkipped)

	if len(r.Verify.ByLocationType) > 0 {
		fmt.Println("### By Location Type")
		fmt.Println()
		fmt.Println("| Location | Verified | Not Found | Rate |")
		fmt.Println("|---|---|---|---|")
		for _, lt := range sortedKeys(r.Verify.ByLocationType) {
			rate := r.Verify.ByLocationType[lt]
			fmt.Printf("| %s | %d | %d | %.0f%% |\n", lt, rate.Good, rate.Bad, rate.Value)
		}
		fmt.Println()
	}

	if len(r.Verify.RecurringFailures) > 0 {
		fmt.Println("### Recurring Verification Failures")
		fmt.Println()
		fmt.Println("| Source FQN | Runs | Not Found | Verified | Fail Rate |")
		fmt.Println("|---|---|---|---|---|")
		for _, rf := range r.Verify.RecurringFailures {
			fmt.Printf("| `%s` | %d | %d | %d | %.0f%% |\n",
				rf.SourceFQN, rf.Occurrences, rf.FailCount, rf.VerifyCount, rf.FailRate)
		}
		fmt.Println()
	}

	if r.Tests.TotalPassed+r.Tests.TotalFailed > 0 {
		fmt.Println("## Test Analysis")
		fmt.Println()
		fmt.Printf("Passed: %d | Failed: %d | Kantra limitation: %d\n\n",
			r.Tests.TotalPassed, r.Tests.TotalFailed, r.Tests.TotalKantraLimitation)

		if len(r.Tests.ByLocationType) > 0 {
			fmt.Println("### Test Results by Location Type")
			fmt.Println()
			fmt.Println("| Location | Passed | Failed | Pass Rate |")
			fmt.Println("|---|---|---|---|")
			for _, lt := range sortedKeys(r.Tests.ByLocationType) {
				rate := r.Tests.ByLocationType[lt]
				fmt.Printf("| %s | %d | %d | %.0f%% |\n", lt, rate.Good, rate.Bad, rate.Value)
			}
			fmt.Println()
		}
	}

	if len(r.Recommendations) > 0 {
		fmt.Println("## Recommendations")
		fmt.Println()
		for i, rec := range r.Recommendations {
			fmt.Printf("### %d. [%s] %s\n\n", i+1, strings.ToUpper(rec.Severity), rec.Title)
			fmt.Printf("%s\n\n", rec.Description)
			if rec.Evidence != "" {
				fmt.Printf("**Evidence:**\n%s\n", rec.Evidence)
			}
			fmt.Printf("**Action:** %s\n\n", rec.Action)
		}
	}
}

func sortedKeys(m map[string]feedback.Rate) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
