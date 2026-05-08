package workspace

import (
	"math"
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
