package testgen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
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
func RunKantraTests(ctx context.Context, testsDir string, timeoutSeconds int) (*TestResult, error) {
	testFiles, err := kantraparser.FindTestFiles(testsDir)
	if err != nil {
		return nil, fmt.Errorf("finding test files: %w", err)
	}
	if len(testFiles) == 0 {
		return nil, fmt.Errorf("no .test.yaml files found in %s", testsDir)
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 900
	}

	return runKantraTest(ctx, testFiles, timeoutSeconds)
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

	passed, total := kantraparser.ParseSummary(output)
	kFailures := kantraparser.ParseFailures(output)

	// Convert kantraparser.Failure to testgen.Failure
	failures := make([]Failure, len(kFailures))
	for i, f := range kFailures {
		failures[i] = Failure{RuleID: f.RuleID, DebugPath: f.DebugPath}
	}

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
