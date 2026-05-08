package rules

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStampTestResults_PassedFailedKantraLimitation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	ruleList := []Rule{
		{RuleID: "rule-001", Message: "passed rule", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.A", LocationImport)},
		{RuleID: "rule-002", Message: "failed rule", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.B", LocationImport)},
		{RuleID: "rule-003", Message: "kantra limitation", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.C", LocationImport)},
		{RuleID: "rule-004", Message: "untouched rule", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.D", LocationImport)},
	}

	if err := WriteRulesFile(path, ruleList); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := StampTestResults(dir, []string{"rule-001"}, []string{"rule-002"}, []string{"rule-003"})
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}

	got, err := ReadRulesFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	tests := []struct {
		ruleID string
		want   string
	}{
		{"rule-001", "konveyor.io/test-result=passed"},
		{"rule-002", "konveyor.io/test-result=failed"},
		{"rule-003", "konveyor.io/test-result=kantra-limitation"},
		{"rule-004", "konveyor.io/test-result=untested"},
	}

	for _, tt := range tests {
		for _, r := range got {
			if r.RuleID != tt.ruleID {
				continue
			}
			found := false
			for _, l := range r.Labels {
				if l == tt.want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: expected label %q, got labels %v", tt.ruleID, tt.want, r.Labels)
			}
		}
	}
}

func TestStampTestResults_KantraLimitationPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	ruleList := []Rule{
		{RuleID: "rule-001", Message: "in both passed and kantra-limitation", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.A", LocationImport)},
	}

	if err := WriteRulesFile(path, ruleList); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := StampTestResults(dir, []string{"rule-001"}, nil, []string{"rule-001"})
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}

	got, err := ReadRulesFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	for _, l := range got[0].Labels {
		if strings.HasPrefix(l, "konveyor.io/test-result=") {
			if l != "konveyor.io/test-result=kantra-limitation" {
				t.Errorf("kantra-limitation should take precedence, got %q", l)
			}
			return
		}
	}
	t.Error("no test-result label found")
}

func TestStampTestResults_NoLabelsField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	ruleList := []Rule{
		{RuleID: "rule-001", Message: "no labels", When: NewJavaReferenced("com.example.A", LocationImport)},
	}

	if err := WriteRulesFile(path, ruleList); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := StampTestResults(dir, nil, nil, []string{"rule-001"})
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}

	got, err := ReadRulesFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	found := false
	for _, l := range got[0].Labels {
		if l == "konveyor.io/test-result=kantra-limitation" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected kantra-limitation label appended, got %v", got[0].Labels)
	}
}
