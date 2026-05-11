package workspace

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// VerificationStats holds aggregate results from deterministic FQN verification.
type VerificationStats struct {
	Verified      int            `yaml:"verified" json:"verified"`
	NotFound      int            `yaml:"not_found" json:"not_found"`
	Skipped       int            `yaml:"skipped" json:"skipped"`
	NotFoundRules []NotFoundRule `yaml:"not_found_rules,omitempty" json:"not_found_rules,omitempty"`
}

// NotFoundRule records a rule whose source FQN was not found in the published artifact.
type NotFoundRule struct {
	RuleID    string `yaml:"rule_id" json:"rule_id"`
	SourceFQN string `yaml:"source_fqn" json:"source_fqn"`
	Reason    string `yaml:"reason" json:"reason"`
}

// Report holds a summary of the rule generation pipeline results.
type Report struct {
	GeneratedAt          string             `yaml:"generated_at" json:"generated_at"`
	Source               string             `yaml:"source" json:"source"`
	Target               string             `yaml:"target" json:"target"`
	RulesTotal           int                `yaml:"rules_total" json:"rules_total"`
	TestsPassed          int                `yaml:"tests_passed" json:"tests_passed"`
	TestsFailed          int                `yaml:"tests_failed" json:"tests_failed"`
	KantraLimitation     int                `yaml:"kantra_limitation" json:"kantra_limitation"`
	PassRate             float64            `yaml:"pass_rate" json:"pass_rate"`
	FailedRules          []string           `yaml:"failed_rules,omitempty" json:"failed_rules,omitempty"`
	KantraLimitationRule []string           `yaml:"kantra_limitation_rules,omitempty" json:"kantra_limitation_rules,omitempty"`
	Verification         *VerificationStats `yaml:"verification,omitempty" json:"verification,omitempty"`
}

// BuildReport creates a report from test results.
// Pass rate is computed as passed / (total - kantraLimitation) to exclude
// rules that are correct but cannot be auto-tested by kantra.
func BuildReport(source, target string, rulesTotal, passed, failed, kantraLimitation int, failedRules, kantraLimitationRules []string) *Report {
	testable := passed + failed
	var passRate float64
	if testable > 0 {
		passRate = float64(passed) / float64(testable) * 100
	}
	return &Report{
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
		Source:               source,
		Target:               target,
		RulesTotal:           rulesTotal,
		TestsPassed:          passed,
		TestsFailed:          failed,
		KantraLimitation:     kantraLimitation,
		PassRate:             passRate,
		FailedRules:          failedRules,
		KantraLimitationRule: kantraLimitationRules,
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
