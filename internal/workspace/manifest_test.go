package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

func TestBuildRulesReport(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-001",
			Labels: []string{
				"konveyor.io/test-result=passed",
				"konveyor.io/review=unreviewed",
			},
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless"},
			},
		},
		{
			RuleID: "rule-002",
			Labels: []string{
				"konveyor.io/test-result=failed",
				"konveyor.io/review=unreviewed",
			},
			When: rules.Condition{
				GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/md4"},
			},
		},
		{
			RuleID: "rule-003",
			Labels: []string{
				"konveyor.io/test-result=untested",
				"konveyor.io/review=unreviewed",
			},
		},
	}

	m := BuildRulesReport(ruleList)

	if m.TotalRules != 3 {
		t.Errorf("TotalRules = %d, want 3", m.TotalRules)
	}
	if m.Passed != 1 {
		t.Errorf("Passed = %d, want 1", m.Passed)
	}
	if m.Failed != 1 {
		t.Errorf("Failed = %d, want 1", m.Failed)
	}
	if m.Untested != 1 {
		t.Errorf("Untested = %d, want 1", m.Untested)
	}
	// 1/3 ≈ 33.33%
	if m.PassRate < 33.0 || m.PassRate > 34.0 {
		t.Errorf("PassRate = %.1f, want ~33.3", m.PassRate)
	}
	if len(m.Rules) != 3 {
		t.Fatalf("len(Rules) = %d, want 3", len(m.Rules))
	}
	if m.Rules[0].Pattern != "javax.ejb.Stateless" {
		t.Errorf("Rules[0].Pattern = %q, want %q", m.Rules[0].Pattern, "javax.ejb.Stateless")
	}
	if m.Rules[1].Pattern != "golang.org/x/crypto/md4" {
		t.Errorf("Rules[1].Pattern = %q, want %q", m.Rules[1].Pattern, "golang.org/x/crypto/md4")
	}
}

func TestWriteRulesReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules-report.yaml")

	ruleList := []rules.Rule{
		{
			RuleID: "rule-001",
			Labels: []string{"konveyor.io/test-result=passed", "konveyor.io/review=unreviewed"},
		},
	}

	if err := WriteRulesReport(path, ruleList); err != nil {
		t.Fatalf("WriteRulesReport: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading report: %v", err)
	}

	var m RulesReport
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshaling report: %v", err)
	}
	if m.TotalRules != 1 {
		t.Errorf("TotalRules = %d, want 1", m.TotalRules)
	}
	if m.Passed != 1 {
		t.Errorf("Passed = %d, want 1", m.Passed)
	}
}
