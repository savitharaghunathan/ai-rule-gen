package testgen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TestResult holds the result of running kantra test.
type TestResult struct {
	Passed    int       `json:"passed"`
	Total     int       `json:"total"`
	PassRate  float64   `json:"pass_rate"`
	Failures  []Failure `json:"failures,omitempty"`
	RawOutput string   `json:"-"`
}

// Failure holds info about a single failed rule.
type Failure struct {
	RuleID    string `json:"rule_id"`
	DebugPath string `json:"debug_path,omitempty"`
}

// FailureInfo holds analyzed details about why a rule failed.
type FailureInfo struct {
	RuleID   string
	Pattern  string
	Provider string
}

// RunKantraTests runs kantra test on all .test.yaml files in testsDir.
//
// Kantra test runs inside a container. As of kantra v0.9.0-alpha.6, the container
// image does NOT include a Go toolchain — only gopls. This means go.referenced rules
// always fail with "no views" because gopls can't resolve modules without `go`.
// Java/Node.js/C# providers work inside the container.
//
// Workaround: if kantra test reports 0/total, we fall back to `kantra analyze --run-local`
// which uses the host's local toolchain (Go, gopls, etc.) and parse violations from
// the output.yaml to produce test results.
//
// TODO: Remove the --run-local fallback once kantra ships a Go toolchain in the container image.
// Track: https://github.com/konveyor-ecosystem/kantra — file issue for missing Go in container.
func RunKantraTests(ctx context.Context, testsDir string, timeoutSeconds int) (*TestResult, error) {
	testFiles, err := findTestFiles(testsDir)
	if err != nil {
		return nil, fmt.Errorf("finding test files: %w", err)
	}
	if len(testFiles) == 0 {
		return nil, fmt.Errorf("no .test.yaml files found in %s", testsDir)
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 900
	}

	// Try kantra test first
	result, err := runKantraTest(ctx, testFiles, timeoutSeconds)
	if err != nil {
		return nil, err
	}

	// If all rules failed, try kantra analyze --run-local as fallback
	// (kantra test container may lack language toolchain, e.g. Go)
	if result.Passed == 0 && result.Total > 0 {
		fmt.Println("  All rules failed in container — trying kantra analyze --run-local...")
		expectedRules := collectExpectedRules(testFiles)
		localResult, localErr := runKantraAnalyzeLocal(ctx, testsDir, testFiles, expectedRules, timeoutSeconds)
		if localErr != nil {
			fmt.Printf("  Warning: --run-local fallback failed: %v\n", localErr)
		} else if localResult.Passed > result.Passed {
			fmt.Printf("  Fallback succeeded: %d/%d passed\n", localResult.Passed, localResult.Total)
			return localResult, nil
		} else {
			fmt.Printf("  Fallback also reported %d/%d passed\n", localResult.Passed, localResult.Total)
		}
	}

	return result, nil
}

// runKantraTest runs `kantra test` with the given test files.
func runKantraTest(ctx context.Context, testFiles []string, timeoutSeconds int) (*TestResult, error) {
	args := []string{"test"}
	args = append(args, testFiles...)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kantra", args...)
	out, runErr := cmd.CombinedOutput()
	output := string(out)

	if ctx.Err() != nil {
		return nil, fmt.Errorf("kantra timed out after %d seconds", timeoutSeconds)
	}

	passed, total := parseSummary(output)
	failures := parseFailures(output)

	// kantra returns non-zero when tests fail — only error if we can't parse anything
	if runErr != nil && passed == 0 && total == 0 {
		return nil, fmt.Errorf("kantra failed: %w\noutput: %s", runErr, output)
	}

	var passRate float64
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	return &TestResult{
		Passed:    passed,
		Total:     total,
		PassRate:  passRate,
		Failures:  failures,
		RawOutput: output,
	}, nil
}

// runKantraAnalyzeLocal runs `kantra analyze --run-local` and compares violations
// against expected rules to produce a TestResult.
//
// TODO: This is a workaround for kantra test container lacking Go toolchain.
// It does NOT validate hasIncidents counts from .test.yaml — only checks presence/absence
// of violations per rule. Enhance to compare actual incident counts once kantra test works.
func runKantraAnalyzeLocal(ctx context.Context, testsDir string, testFiles []string, expectedRules []string, timeoutSeconds int) (*TestResult, error) {
	// Parse test files to find rules path and data paths
	rulesDir, dataDirs, providers := parseTestFilesPaths(testsDir, testFiles)
	if rulesDir == "" || len(dataDirs) == 0 {
		return nil, fmt.Errorf("could not parse rules/data paths from test files (rulesDir=%q, dataDirs=%v)", rulesDir, dataDirs)
	}
	fmt.Printf("  --run-local: rules=%s input=%s providers=%v\n", rulesDir, dataDirs[0], providers)

	outputDir, err := os.MkdirTemp("", "kantra-analyze-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(outputDir)

	// TODO: Currently only uses the first data dir. Support multiple data dirs
	// when test files reference different data paths per provider.
	args := []string{"analyze",
		"--input", dataDirs[0],
		"--rules", rulesDir,
		"--run-local",
		"--output", outputDir,
		"--overwrite",
	}
	for _, p := range providers {
		args = append(args, "--provider", p)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kantra", args...)
	analyzeOut, analyzeErr := cmd.CombinedOutput()

	if ctx.Err() != nil {
		return nil, fmt.Errorf("kantra analyze timed out")
	}
	if analyzeErr != nil {
		return nil, fmt.Errorf("kantra analyze failed: %w\noutput: %s", analyzeErr, string(analyzeOut))
	}

	// Parse output.yaml for violations
	outputFile := filepath.Join(outputDir, "output.yaml")
	matched := parseAnalyzeViolations(outputFile)

	// Compare against expected rules
	total := len(expectedRules)
	passed := 0
	var failures []Failure
	for _, ruleID := range expectedRules {
		if matched[ruleID] {
			passed++
		} else {
			failures = append(failures, Failure{RuleID: ruleID})
		}
	}

	var passRate float64
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	return &TestResult{
		Passed:   passed,
		Total:    total,
		PassRate: passRate,
		Failures: failures,
	}, nil
}

// collectExpectedRules extracts rule IDs from .test.yaml files.
func collectExpectedRules(testFiles []string) []string {
	var rules []string
	for _, f := range testFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tf TestFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}
		for _, t := range tf.Tests {
			rules = append(rules, t.RuleID)
		}
	}
	return rules
}

// parseTestFilesPaths extracts the rules directory and data directories from test files.
func parseTestFilesPaths(testsDir string, testFiles []string) (rulesDir string, dataDirs []string, providers []string) {
	for _, f := range testFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tf TestFile
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}
		if tf.RulesPath != "" && rulesDir == "" {
			// RulesPath is relative to the test file — resolve to the directory containing the rules
			absRules := filepath.Join(testsDir, tf.RulesPath)
			rulesDir = filepath.Dir(absRules)
		}
		for _, p := range tf.Providers {
			if p.DataPath != "" {
				absData := filepath.Join(testsDir, p.DataPath)
				dataDirs = append(dataDirs, absData)
			}
			if p.Name != "" {
				providers = append(providers, p.Name)
			}
		}
	}
	return
}

// parseAnalyzeViolations reads kantra analyze output.yaml and returns which rule IDs had violations.
func parseAnalyzeViolations(outputFile string) map[string]bool {
	matched := make(map[string]bool)

	data, err := os.ReadFile(outputFile)
	if err != nil {
		return matched
	}

	// output.yaml is a list of rulesets, each with violations map
	var rulesets []struct {
		Violations map[string]interface{} `yaml:"violations"`
	}
	if err := yaml.Unmarshal(data, &rulesets); err != nil {
		return matched
	}

	for _, rs := range rulesets {
		for ruleID := range rs.Violations {
			matched[ruleID] = true
		}
	}
	return matched
}

// findTestFiles returns .test.yaml/.test.yml files in a directory.
func findTestFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".test.yaml") || strings.HasSuffix(e.Name(), ".test.yml") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

var reSummary = regexp.MustCompile(`Rules Summary:\s+(\d+)/(\d+)`)
var reFailure = regexp.MustCompile(`([\w-]+-\d{5})\s+0/\d+\s+PASSED(?:.*?find debug data in (/[^\s]+))?`)

func parseSummary(output string) (passed, total int) {
	m := reSummary.FindStringSubmatch(output)
	if len(m) == 3 {
		fmt.Sscanf(m[1], "%d", &passed)
		fmt.Sscanf(m[2], "%d", &total)
	}
	return
}

func parseFailures(output string) []Failure {
	var failures []Failure
	matches := reFailure.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		f := Failure{RuleID: m[1]}
		if len(m) > 2 {
			f.DebugPath = m[2]
		}
		failures = append(failures, f)
	}
	return failures
}

// AnalyzeFailure reads kantra debug output to extract pattern and provider info.
func AnalyzeFailure(f Failure) (*FailureInfo, error) {
	if f.DebugPath == "" {
		return &FailureInfo{RuleID: f.RuleID}, nil
	}

	// Read output.yaml for unmatched rules
	outputPath := filepath.Join(f.DebugPath, "output.yaml")
	rulesPath := filepath.Join(f.DebugPath, "rules.yaml")

	// Try to read the rules file to extract pattern and provider
	rulesData, err := os.ReadFile(rulesPath)
	if err != nil {
		// Debug path might not exist or be accessible
		return &FailureInfo{RuleID: f.RuleID}, nil
	}

	pattern, provider := extractPatternFromDebug(rulesData, f.RuleID)

	_ = outputPath // output.yaml confirms which rules are unmatched; we already know from kantra output

	return &FailureInfo{
		RuleID:   f.RuleID,
		Pattern:  pattern,
		Provider: provider,
	}, nil
}

// extractPatternFromDebug parses kantra's debug rules.yaml to find the pattern for a rule.
func extractPatternFromDebug(data []byte, ruleID string) (pattern, provider string) {
	// kantra's debug rules.yaml is a list of rule objects
	var debugRules []map[string]interface{}
	if err := yaml.Unmarshal(data, &debugRules); err != nil {
		return "", ""
	}

	for _, r := range debugRules {
		id, _ := r["ruleID"].(string)
		if id != ruleID {
			continue
		}
		when, ok := r["when"].(map[string]interface{})
		if !ok {
			continue
		}
		// Check each provider type
		for _, prov := range []string{"go.referenced", "java.referenced", "nodejs.referenced", "csharp.referenced", "builtin.filecontent"} {
			if cond, ok := when[prov]; ok {
				if condMap, ok := cond.(map[string]interface{}); ok {
					if p, ok := condMap["pattern"].(string); ok {
						return p, prov
					}
				}
			}
		}
	}
	return "", ""
}
