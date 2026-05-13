package workspace

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReport_NoKantraLimitation(t *testing.T) {
	passed := []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"}
	failed := []string{"rule-001", "rule-002"}
	r := BuildReport([]string{"spring-boot-3"}, []string{"spring-boot-4"}, 10, 8, 2, 0, passed, failed, nil, nil, nil)

	if r.RulesTotal != 10 {
		t.Errorf("rules_total: got %d, want 10", r.RulesTotal)
	}
	if r.TestsPassed != 8 {
		t.Errorf("tests_passed: got %d, want 8", r.TestsPassed)
	}
	if r.TestsFailed != 2 {
		t.Errorf("tests_failed: got %d, want 2", r.TestsFailed)
	}
	if r.KantraLimitation != 0 {
		t.Errorf("kantra_limitation: got %d, want 0", r.KantraLimitation)
	}
	want := 80.0
	if math.Abs(r.PassRate-want) > 0.01 {
		t.Errorf("pass_rate: got %.2f, want %.2f", r.PassRate, want)
	}
}

func TestBuildReport_WithKantraLimitation(t *testing.T) {
	passed := []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"}
	failed := []string{"rule-003"}
	kantra := []string{"rule-008", "rule-009"}
	r := BuildReport([]string{"spring-boot-3"}, []string{"spring-boot-4"}, 10, 7, 1, 2, passed, failed, kantra, nil, nil)

	if r.KantraLimitation != 2 {
		t.Errorf("kantra_limitation: got %d, want 2", r.KantraLimitation)
	}
	want := 87.5
	if math.Abs(r.PassRate-want) > 0.01 {
		t.Errorf("pass_rate: got %.2f, want %.2f (should exclude kantra limitations)", r.PassRate, want)
	}

	kantraCount := 0
	for _, rs := range r.Rules {
		if rs.TestStatus == "kantra-limitation" {
			kantraCount++
		}
	}
	if kantraCount != 2 {
		t.Errorf("kantra-limitation rules in report: got %d, want 2", kantraCount)
	}
}

func TestBuildReport_AllKantraLimitation(t *testing.T) {
	kantra := []string{"a", "b", "c"}
	r := BuildReport([]string{"x"}, []string{"y"}, 3, 0, 0, 3, nil, nil, kantra, nil, nil)

	if r.PassRate != 0 {
		t.Errorf("pass_rate: got %.2f, want 0 (no testable rules)", r.PassRate)
	}
}

func TestBuildReport_PerRuleStatus(t *testing.T) {
	passed := []string{"r1", "r2"}
	failed := []string{"r3"}
	kantra := []string{"r4"}
	verified := []string{"r1", "r3"}
	notFound := []string{"r2", "r5"}

	r := BuildReport([]string{"sb3"}, []string{"sb4"}, 5, 2, 1, 1, passed, failed, kantra, verified, notFound)

	if len(r.Rules) != 5 {
		t.Fatalf("rules count: got %d, want 5", len(r.Rules))
	}

	statusMap := make(map[string]RuleStatus)
	for _, rs := range r.Rules {
		statusMap[rs.RuleID] = rs
	}

	tests := []struct {
		ruleID     string
		testStatus string
		verified   string
	}{
		{"r1", "passed", "true"},
		{"r2", "passed", "false"},
		{"r3", "failed", "true"},
		{"r4", "kantra-limitation", ""},
		{"r5", "untested", "false"},
	}

	for _, tt := range tests {
		rs, ok := statusMap[tt.ruleID]
		if !ok {
			t.Errorf("rule %s: missing from report", tt.ruleID)
			continue
		}
		if rs.TestStatus != tt.testStatus {
			t.Errorf("rule %s test_status: got %q, want %q", tt.ruleID, rs.TestStatus, tt.testStatus)
		}
		if rs.SourceVerified != tt.verified {
			t.Errorf("rule %s source_verified: got %q, want %q", tt.ruleID, rs.SourceVerified, tt.verified)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "report.yaml")
	if err := WriteReport(path, r); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "rules:") {
		t.Error("report YAML missing rules section")
	}
	if !strings.Contains(content, "test_status: passed") {
		t.Error("report YAML missing test_status")
	}
	if !strings.Contains(content, "source_verified: \"true\"") {
		t.Error("report YAML missing source_verified")
	}
}
