package workspace

import (
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// RulesReport is a human-readable summary of rule verification status.
type RulesReport struct {
	TotalRules int                   `yaml:"totalRules"`
	Passed     int                   `yaml:"passed"`
	Failed     int                   `yaml:"failed"`
	Untested   int                   `yaml:"untested"`
	PassRate   float64               `yaml:"passRate"`
	Rules      []RulesReportEntry `yaml:"rules"`
}

// RulesReportEntry is a single rule's status in the report.
type RulesReportEntry struct {
	RuleID     string `yaml:"ruleID"`
	TestResult string `yaml:"testResult"`
	Review     string `yaml:"review"`
	Pattern    string `yaml:"pattern,omitempty"`
}

// WriteRulesReport generates a rules report YAML from a list of rules.
func WriteRulesReport(path string, ruleList []rules.Rule) error {
	report := BuildRulesReport(ruleList)

	data, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling rules report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing rules report: %w", err)
	}
	return nil
}

// BuildRulesReport creates a RulesReport from a list of rules.
func BuildRulesReport(ruleList []rules.Rule) RulesReport {
	report := RulesReport{
		TotalRules: len(ruleList),
	}

	for _, r := range ruleList {
		testResult := rules.GetLabel(r.Labels, rules.LabelTestResult)
		review := rules.GetLabel(r.Labels, rules.LabelReview)

		switch testResult {
		case rules.TestResultPassed:
			report.Passed++
		case rules.TestResultFailed:
			report.Failed++
		default:
			report.Untested++
		}

		report.Rules = append(report.Rules, RulesReportEntry{
			RuleID:     r.RuleID,
			TestResult: testResult,
			Review:     review,
			Pattern:    extractPattern(r),
		})
	}

	if report.TotalRules > 0 {
		report.PassRate = float64(report.Passed) / float64(report.TotalRules) * 100
	}

	return report
}

// extractPattern returns the primary pattern string from a rule's condition.
func extractPattern(r rules.Rule) string {
	c := r.When
	if c.JavaReferenced != nil {
		return c.JavaReferenced.Pattern
	}
	if c.JavaDependency != nil {
		return c.JavaDependency.Name
	}
	if c.GoReferenced != nil {
		return c.GoReferenced.Pattern
	}
	if c.GoDependency != nil {
		return c.GoDependency.Name
	}
	if c.NodejsReferenced != nil {
		return c.NodejsReferenced.Pattern
	}
	if c.CSharpReferenced != nil {
		return c.CSharpReferenced.Pattern
	}
	if c.BuiltinFilecontent != nil {
		return c.BuiltinFilecontent.Pattern
	}
	if c.BuiltinFile != nil {
		return c.BuiltinFile.Pattern
	}
	return ""
}
