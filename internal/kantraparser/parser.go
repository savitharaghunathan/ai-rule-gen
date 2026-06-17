package kantraparser

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Summary holds parsed kantra test results.
type Summary struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
}

var reSummary = regexp.MustCompile(`Rules Summary:\s+(\d+)/(\d+)`)
var reResult = regexp.MustCompile(`([\w-]+-\d{5})\s+(\d+)/(\d+)\s+PASSED`)

// ParseSummary extracts passed/total counts from kantra test output.
func ParseSummary(output string) (passed, total int) {
	m := reSummary.FindStringSubmatch(output)
	if len(m) == 3 {
		fmt.Sscanf(m[1], "%d", &passed)
		fmt.Sscanf(m[2], "%d", &total)
	}
	return
}

// ParseResults extracts per-rule pass/fail results from kantra test output.
// Returns a map of ruleID → passed (true if N/N PASSED where N > 0).
func ParseResults(output string) map[string]bool {
	results := make(map[string]bool)
	matches := reResult.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		ruleID := m[1]
		var matched, total int
		fmt.Sscanf(m[2], "%d", &matched)
		fmt.Sscanf(m[3], "%d", &total)
		results[ruleID] = matched > 0 && matched == total
	}
	return results
}

// PassedAndFailed categorizes rule IDs into passed and failed lists
// based on kantra test output and the known set of all rule IDs.
// Rules are only counted as passed if they explicitly show N/N PASSED
// (where N > 0) in the output. Rules not found in the output or with
// non-standard error output default to failed.
// erroredRuleIDs are rules from groups that errored before any rules ran
// (e.g., "unable to get build tool"). These are always marked as failed.
func PassedAndFailed(output string, allRuleIDs []string, erroredRuleIDs ...string) (passed, failed []string) {
	results := ParseResults(output)

	failedSet := make(map[string]bool)
	for _, id := range erroredRuleIDs {
		failedSet[id] = true
	}

	for _, id := range allRuleIDs {
		if failedSet[id] {
			failed = append(failed, id)
			continue
		}
		didPass, found := results[id]
		if found && didPass {
			passed = append(passed, id)
		} else {
			failed = append(failed, id)
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

// TestFileProviders extracts provider names from a .test.yaml file.
func TestFileProviders(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tf struct {
		Providers []struct {
			Name string `yaml:"name"`
		} `yaml:"providers"`
	}
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}
	var names []string
	for _, p := range tf.Providers {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return names, nil
}

// SupportsRunLocal returns true if all providers in the test file
// are supported by kantra's --run-local (containerless) mode.
// Only "java" and "builtin" are supported in containerless mode.
func SupportsRunLocal(path string) bool {
	providers, err := TestFileProviders(path)
	if err != nil || len(providers) == 0 {
		return true // default to run-local if we can't determine
	}
	for _, p := range providers {
		if p != "java" && p != "builtin" {
			return false
		}
	}
	return true
}
