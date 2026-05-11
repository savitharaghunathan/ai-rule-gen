package workspace

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReport_NoKantraLimitation(t *testing.T) {
	r := BuildReport("spring-boot-3", "spring-boot-4", 10, 8, 2, 0, []string{"rule-001", "rule-002"}, nil)

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
	r := BuildReport("spring-boot-3", "spring-boot-4", 10, 7, 1, 2,
		[]string{"rule-003"}, []string{"rule-008", "rule-009"})

	if r.KantraLimitation != 2 {
		t.Errorf("kantra_limitation: got %d, want 2", r.KantraLimitation)
	}
	// pass rate = 7 / (7 + 1) * 100 = 87.5
	want := 87.5
	if math.Abs(r.PassRate-want) > 0.01 {
		t.Errorf("pass_rate: got %.2f, want %.2f (should exclude kantra limitations)", r.PassRate, want)
	}
	if len(r.KantraLimitationRule) != 2 {
		t.Errorf("kantra_limitation_rules: got %v, want 2 items", r.KantraLimitationRule)
	}
}

func TestBuildReport_AllKantraLimitation(t *testing.T) {
	r := BuildReport("x", "y", 3, 0, 0, 3, nil, []string{"a", "b", "c"})

	if r.PassRate != 0 {
		t.Errorf("pass_rate: got %.2f, want 0 (no testable rules)", r.PassRate)
	}
}

func TestBuildReport_WithVerification(t *testing.T) {
	r := BuildReport("sb3", "sb4", 20, 15, 3, 2, []string{"r1", "r2", "r3"}, []string{"k1", "k2"})
	r.Verification = &VerificationStats{
		Verified: 14,
		NotFound: 3,
		Skipped:  3,
		NotFoundRules: []NotFoundRule{
			{RuleID: "r1", SourceFQN: "com.example.Fake", Reason: "not found in foo-1.0.jar"},
		},
	}

	if r.Verification.Verified != 14 {
		t.Errorf("verified = %d, want 14", r.Verification.Verified)
	}
	if len(r.Verification.NotFoundRules) != 1 {
		t.Fatalf("not_found_rules length = %d, want 1", len(r.Verification.NotFoundRules))
	}
	if r.Verification.NotFoundRules[0].RuleID != "r1" {
		t.Errorf("not_found rule_id = %q, want r1", r.Verification.NotFoundRules[0].RuleID)
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
	if !strings.Contains(content, "verification:") {
		t.Error("report YAML missing verification section")
	}
	if !strings.Contains(content, "verified: 14") {
		t.Error("report YAML missing verified count")
	}
}
