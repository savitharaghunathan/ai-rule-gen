package rules

import (
	"testing"
)

func TestSetLabel_New(t *testing.T) {
	labels := []string{"konveyor.io/source=java-8"}
	labels = SetLabel(labels, LabelTestResult, TestResultPassed)
	if len(labels) != 2 {
		t.Fatalf("len = %d, want 2", len(labels))
	}
	if labels[1] != "konveyor.io/test-result=passed" {
		t.Errorf("label = %q, want %q", labels[1], "konveyor.io/test-result=passed")
	}
}

func TestSetLabel_Replace(t *testing.T) {
	labels := []string{
		"konveyor.io/source=java-8",
		"konveyor.io/test-result=untested",
	}
	labels = SetLabel(labels, LabelTestResult, TestResultPassed)
	if len(labels) != 2 {
		t.Fatalf("len = %d, want 2", len(labels))
	}
	if labels[1] != "konveyor.io/test-result=passed" {
		t.Errorf("label = %q, want %q", labels[1], "konveyor.io/test-result=passed")
	}
}

func TestGetLabel_Found(t *testing.T) {
	labels := []string{
		"konveyor.io/source=java-8",
		"konveyor.io/test-result=passed",
	}
	v := GetLabel(labels, LabelTestResult)
	if v != "passed" {
		t.Errorf("value = %q, want %q", v, "passed")
	}
}

func TestGetLabel_NotFound(t *testing.T) {
	labels := []string{"konveyor.io/source=java-8"}
	v := GetLabel(labels, LabelTestResult)
	if v != "" {
		t.Errorf("value = %q, want empty", v)
	}
}

func TestInitialLabels(t *testing.T) {
	existing := []string{
		"konveyor.io/source=java-8",
		"konveyor.io/target=java-17",
	}
	labels := InitialLabels(existing)

	// Should not mutate original
	if len(existing) != 2 {
		t.Fatalf("existing mutated: len = %d", len(existing))
	}

	// Should have 5 labels: source, target, generated-by, test-result, review
	if len(labels) != 5 {
		t.Fatalf("len = %d, want 5", len(labels))
	}

	if v := GetLabel(labels, LabelGeneratedBy); v != GeneratedByValue {
		t.Errorf("generated-by = %q, want %q", v, GeneratedByValue)
	}
	if v := GetLabel(labels, LabelTestResult); v != TestResultUntested {
		t.Errorf("test-result = %q, want %q", v, TestResultUntested)
	}
	if v := GetLabel(labels, LabelReview); v != ReviewUnreviewed {
		t.Errorf("review = %q, want %q", v, ReviewUnreviewed)
	}
	// Original labels preserved
	if v := GetLabel(labels, "konveyor.io/source"); v != "java-8" {
		t.Errorf("source = %q, want %q", v, "java-8")
	}
}

func TestStampTestResults(t *testing.T) {
	ruleList := []Rule{
		{RuleID: "rule-001", Labels: []string{"konveyor.io/test-result=untested"}},
		{RuleID: "rule-002", Labels: []string{"konveyor.io/test-result=untested"}},
		{RuleID: "rule-003", Labels: []string{"konveyor.io/test-result=untested"}},
	}

	passed := map[string]bool{"rule-001": true}
	failed := map[string]bool{"rule-002": true}

	ruleList = StampTestResults(ruleList, passed, failed)

	if v := GetLabel(ruleList[0].Labels, LabelTestResult); v != TestResultPassed {
		t.Errorf("rule-001 test-result = %q, want %q", v, TestResultPassed)
	}
	if v := GetLabel(ruleList[1].Labels, LabelTestResult); v != TestResultFailed {
		t.Errorf("rule-002 test-result = %q, want %q", v, TestResultFailed)
	}
	if v := GetLabel(ruleList[2].Labels, LabelTestResult); v != TestResultUntested {
		t.Errorf("rule-003 test-result = %q, want unchanged %q", v, TestResultUntested)
	}
}
