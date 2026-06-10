// compare diffs two Konveyor rulesets: a coverage matrix and, with --app-dir,
// a side-by-side kantra-analyze run on the same app.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/compare"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	rulesA := flag.String("a", "", "Path to ruleset A (required)")
	rulesB := flag.String("b", "", "Path to ruleset B (required)")
	nameA := flag.String("name-a", "", "Display label for ruleset A (default: directory name)")
	nameB := flag.String("name-b", "", "Display label for ruleset B (default: directory name)")
	appDir := flag.String("app-dir", "", "Optional: run kantra against this app dir with both rulesets")
	outPath := flag.String("out", "", "Optional: write markdown report to this path (stderr otherwise)")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *rulesA == "" || *rulesB == "" {
		cli.Fail("invalid_arguments", "--a and --b are required", "compare", "provide both ruleset directories", nil)
	}

	rsA, warnA, err := compare.LoadRulesDirRaw(*rulesA)
	if err != nil {
		cli.Fail("rules_load_failed", err.Error(), "compare", "check --a path", map[string]string{"path": *rulesA})
	}
	rsB, warnB, err := compare.LoadRulesDirRaw(*rulesB)
	if err != nil {
		cli.Fail("rules_load_failed", err.Error(), "compare", "check --b path", map[string]string{"path": *rulesB})
	}
	for _, w := range warnA {
		fmt.Fprintf(os.Stderr, "[compare] warn A: %s\n", w)
	}
	for _, w := range warnB {
		fmt.Fprintf(os.Stderr, "[compare] warn B: %s\n", w)
	}
	if len(rsA) == 0 {
		cli.Fail("empty_rules", "no rules in --a", "compare", "check that --a contains rule yaml files", nil)
	}
	if len(rsB) == 0 {
		cli.Fail("empty_rules", "no rules in --b", "compare", "check that --b contains rule yaml files", nil)
	}

	result := &compare.Result{
		NameA:      defaultName(*nameA, *rulesA),
		NameB:      defaultName(*nameB, *rulesB),
		RulesDirA:  *rulesA,
		RulesDirB:  *rulesB,
		RuleCountA: len(rsA),
		RuleCountB: len(rsB),
		Matrix:     compare.BuildMatrix(rsA, rsB),
	}

	if *appDir != "" {
		diff, err := compare.RunKantraDiff(*rulesA, *rulesB, *appDir)
		if err != nil {
			cli.Fail("kantra_diff_failed", err.Error(), "compare", "ensure kantra is installed and app dir is valid", nil)
		}
		result.KantraDiff = diff
	}

	if *outPath != "" {
		if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
			cli.Fail("output_dir_failed", err.Error(), "compare", "check --out path", nil)
		}
		f, err := os.Create(*outPath)
		if err != nil {
			cli.Fail("output_open_failed", err.Error(), "compare", "check --out path", nil)
		}
		if err := compare.WriteMarkdown(f, result); err != nil {
			f.Close()
			cli.Fail("output_write_failed", err.Error(), "compare", "check disk space", nil)
		}
		f.Close()
		fmt.Fprintf(os.Stderr, "[compare] report written to %s\n", *outPath)
	} else {
		_ = compare.WriteMarkdown(os.Stderr, result)
	}

	cli.WriteJSON(result)
}

func defaultName(provided, path string) string {
	if provided != "" {
		return provided
	}
	clean := filepath.Clean(path)
	parts := []string{filepath.Base(clean)}
	if parent := filepath.Base(filepath.Dir(clean)); parent != "" && parent != "." && parent != "/" {
		parts = []string{parent + "/" + parts[0]}
	}
	return parts[0]
}
