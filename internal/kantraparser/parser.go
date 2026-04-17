package kantraparser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Summary holds parsed kantra test results.
type Summary struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
}

// Failure holds info about a single failed rule.
type Failure struct {
	RuleID    string `json:"rule_id"`
	DebugPath string `json:"debug_path,omitempty"`
}

var reSummary = regexp.MustCompile(`Rules Summary:\s+(\d+)/(\d+)`)
var reFailure = regexp.MustCompile(`([\w-]+-\d{5})\s+0/\d+\s+PASSED(?:.*?find debug data in (/[^\s]+))?`)

// ParseSummary extracts passed/total counts from kantra test output.
func ParseSummary(output string) (passed, total int) {
	m := reSummary.FindStringSubmatch(output)
	if len(m) == 3 {
		fmt.Sscanf(m[1], "%d", &passed)
		fmt.Sscanf(m[2], "%d", &total)
	}
	return
}

// ParseFailures extracts failing rule IDs from kantra test output.
func ParseFailures(output string) []Failure {
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

// FindTestFiles returns .test.yaml/.test.yml files in a directory.
func FindTestFiles(dir string) ([]string, error) {
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

// ParseAnalyzeViolations reads kantra analyze output.yaml and returns which rule IDs had violations.
func ParseAnalyzeViolations(outputFile string) map[string]bool {
	matched := make(map[string]bool)
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return matched
	}
	var rulesets []struct {
		Violations map[string]any `yaml:"violations"`
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

// PassedAndFailed categorizes rule IDs into passed and failed lists
// based on kantra test output and the known set of all rule IDs.
// erroredRuleIDs are rules from groups that errored before any rules ran
// (e.g., "unable to get build tool"). These are always marked as failed.
func PassedAndFailed(output string, allRuleIDs []string, erroredRuleIDs ...string) (passed, failed []string) {
	failures := ParseFailures(output)

	failedSet := make(map[string]bool)
	for _, f := range failures {
		failedSet[f.RuleID] = true
		failed = append(failed, f.RuleID)
	}
	for _, id := range erroredRuleIDs {
		if !failedSet[id] {
			failedSet[id] = true
			failed = append(failed, id)
		}
	}

	for _, id := range allRuleIDs {
		if !failedSet[id] {
			passed = append(passed, id)
		}
	}

	return passed, failed
}

// TestFileRuleIDs extracts rule IDs from a .test.yaml file.
func TestFileRuleIDs(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tf struct {
		Tests []struct {
			RuleID string `yaml:"ruleID"`
		} `yaml:"tests"`
	}
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}
	var ids []string
	for _, t := range tf.Tests {
		if t.RuleID != "" {
			ids = append(ids, t.RuleID)
		}
	}
	return ids, nil
}
