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
	Offline       int            `yaml:"offline,omitempty" json:"offline,omitempty"`
	NotFoundRules []NotFoundRule `yaml:"not_found_rules,omitempty" json:"not_found_rules,omitempty"`
}

// NotFoundRule records a rule whose source FQN was not found in the published artifact.
type NotFoundRule struct {
	RuleID    string `yaml:"rule_id" json:"rule_id"`
	SourceFQN string `yaml:"source_fqn" json:"source_fqn"`
	Reason    string `yaml:"reason" json:"reason"`
}

// RuleStatus records the per-rule test and verification status.
type RuleStatus struct {
	RuleID         string `yaml:"rule_id" json:"rule_id"`
	TestStatus     string `yaml:"test_status" json:"test_status"`
	SourceVerified string `yaml:"source_verified,omitempty" json:"source_verified,omitempty"`
}

// Report holds a summary of the rule generation pipeline results.
type Report struct {
	GeneratedAt      string             `yaml:"generated_at" json:"generated_at"`
	Sources          []string           `yaml:"sources" json:"sources"`
	Targets          []string           `yaml:"targets" json:"targets"`
	RulesTotal       int                `yaml:"rules_total" json:"rules_total"`
	TestsPassed      int                `yaml:"tests_passed" json:"tests_passed"`
	TestsFailed      int                `yaml:"tests_failed" json:"tests_failed"`
	KantraLimitation int                `yaml:"kantra_limitation" json:"kantra_limitation"`
	PassRate         float64            `yaml:"pass_rate" json:"pass_rate"`
	Verification     *VerificationStats `yaml:"verification,omitempty" json:"verification,omitempty"`
	Rules            []RuleStatus       `yaml:"rules,omitempty" json:"rules,omitempty"`
}

// BuildReport creates a report from test results with per-rule status.
// Pass rate is computed as passed / (passed + failed) to exclude
// rules that are correct but cannot be auto-tested by kantra.
func BuildReport(sources, targets []string, rulesTotal, passed, failed, kantraLimitation int, passedRules, failedRules, kantraLimitationRules, verifiedRules, notFoundRules []string) *Report {
	testable := passed + failed
	var passRate float64
	if testable > 0 {
		passRate = float64(passed) / float64(testable) * 100
	}

	verifiedSet := toSet(verifiedRules)
	notFoundSet := toSet(notFoundRules)

	var ruleStatuses []RuleStatus
	for _, id := range passedRules {
		ruleStatuses = append(ruleStatuses, RuleStatus{
			RuleID:         id,
			TestStatus:     "passed",
			SourceVerified: verifyLabel(id, verifiedSet, notFoundSet),
		})
	}
	for _, id := range failedRules {
		ruleStatuses = append(ruleStatuses, RuleStatus{
			RuleID:         id,
			TestStatus:     "failed",
			SourceVerified: verifyLabel(id, verifiedSet, notFoundSet),
		})
	}
	for _, id := range kantraLimitationRules {
		ruleStatuses = append(ruleStatuses, RuleStatus{
			RuleID:         id,
			TestStatus:     "kantra-limitation",
			SourceVerified: verifyLabel(id, verifiedSet, notFoundSet),
		})
	}

	seen := toSet(append(append(passedRules, failedRules...), kantraLimitationRules...))
	allVerifyRules := append(verifiedRules, notFoundRules...)
	for _, id := range allVerifyRules {
		if seen[id] {
			continue
		}
		seen[id] = true
		ruleStatuses = append(ruleStatuses, RuleStatus{
			RuleID:         id,
			TestStatus:     "untested",
			SourceVerified: verifyLabel(id, verifiedSet, notFoundSet),
		})
	}

	return &Report{
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		Sources:           sources,
		Targets:           targets,
		RulesTotal:        rulesTotal,
		TestsPassed:       passed,
		TestsFailed:       failed,
		KantraLimitation: kantraLimitation,
		PassRate:          passRate,
		Rules:             ruleStatuses,
	}
}

func toSet(ids []string) map[string]bool {
	s := make(map[string]bool, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}

func verifyLabel(id string, verified, notFound map[string]bool) string {
	if verified[id] {
		return "true"
	}
	if notFound[id] {
		return "false"
	}
	return ""
}

// ReadReport reads a Report from a YAML file.
// It handles both the old singular source/target format and the
// current plural sources/targets format.
func ReadReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading report %s: %w", path, err)
	}
	var report Report
	if err := yaml.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing report %s: %w", path, err)
	}

	// Handle old singular format.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err == nil {
		if s, ok := raw["source"].(string); ok && len(report.Sources) == 0 {
			report.Sources = []string{s}
		}
		if t, ok := raw["target"].(string); ok && len(report.Targets) == 0 {
			report.Targets = []string{t}
		}
	}

	return &report, nil
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
