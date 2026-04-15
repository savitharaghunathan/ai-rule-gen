package workspace

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Report holds a summary of the rule generation pipeline results.
type Report struct {
	GeneratedAt string   `yaml:"generated_at" json:"generated_at"`
	Source      string   `yaml:"source" json:"source"`
	Target      string   `yaml:"target" json:"target"`
	RulesTotal  int      `yaml:"rules_total" json:"rules_total"`
	TestsPassed int      `yaml:"tests_passed" json:"tests_passed"`
	TestsFailed int      `yaml:"tests_failed" json:"tests_failed"`
	PassRate    float64  `yaml:"pass_rate" json:"pass_rate"`
	FailedRules []string `yaml:"failed_rules,omitempty" json:"failed_rules,omitempty"`
}

// BuildReport creates a report from test results.
func BuildReport(source, target string, rulesTotal, passed, failed int, failedRules []string) *Report {
	total := passed + failed
	var passRate float64
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}
	return &Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Source:      source,
		Target:      target,
		RulesTotal:  rulesTotal,
		TestsPassed: passed,
		TestsFailed: failed,
		PassRate:    passRate,
		FailedRules: failedRules,
	}
}

// WriteReport writes the report to a YAML file.
func WriteReport(path string, report *Report) error {
	data, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing report %s: %w", path, err)
	}
	return nil
}
