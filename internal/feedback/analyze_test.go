package feedback

import (
	"testing"
	"time"
)

func makeRun(ts time.Time, patterns []PatternOutcome, passed, failed int) *RunSummary {
	passRate := 0.0
	if passed+failed > 0 {
		passRate = float64(passed) / float64(passed+failed) * 100
	}
	return &RunSummary{
		Timestamp:   ts,
		Sources:     []string{"src"},
		Targets:     []string{"tgt"},
		RulesTotal:  len(patterns),
		TestsPassed: passed,
		TestsFailed: failed,
		PassRate:    passRate,
		Patterns:    patterns,
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	report := Analyze(nil, 2)
	if report.RunsAnalyzed != 0 {
		t.Errorf("expected 0 runs analyzed, got %d", report.RunsAnalyzed)
	}
}

func TestAnalyzeRecurringFailures(t *testing.T) {
	p1 := PatternOutcome{SourceFQN: "com.example.AlwaysFails", VerifyStatus: "not_found", LocationType: "METHOD_CALL"}
	p2 := PatternOutcome{SourceFQN: "com.example.AlwaysPasses", VerifyStatus: "verified", LocationType: "PACKAGE"}

	t1 := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)

	runs := []*RunSummary{
		makeRun(t1, []PatternOutcome{p1, p2}, 1, 0),
		makeRun(t2, []PatternOutcome{p1, p2}, 1, 0),
		makeRun(t3, []PatternOutcome{p1, p2}, 1, 0),
	}

	report := Analyze(runs, 3)

	if report.Verify.TotalVerified != 3 {
		t.Errorf("TotalVerified: got %d, want 3", report.Verify.TotalVerified)
	}
	if report.Verify.TotalNotFound != 3 {
		t.Errorf("TotalNotFound: got %d, want 3", report.Verify.TotalNotFound)
	}

	if len(report.Verify.RecurringFailures) != 1 {
		t.Fatalf("RecurringFailures: got %d, want 1", len(report.Verify.RecurringFailures))
	}
	rf := report.Verify.RecurringFailures[0]
	if rf.SourceFQN != "com.example.AlwaysFails" {
		t.Errorf("SourceFQN: got %q", rf.SourceFQN)
	}
	if rf.FailRate != 100 {
		t.Errorf("FailRate: got %.1f, want 100", rf.FailRate)
	}
}

func TestAnalyzeByLocationType(t *testing.T) {
	patterns := []PatternOutcome{
		{SourceFQN: "a", LocationType: "PACKAGE", VerifyStatus: "verified"},
		{SourceFQN: "b", LocationType: "PACKAGE", VerifyStatus: "verified"},
		{SourceFQN: "c", LocationType: "METHOD_CALL", VerifyStatus: "not_found"},
		{SourceFQN: "d", LocationType: "METHOD_CALL", VerifyStatus: "not_found"},
		{SourceFQN: "e", LocationType: "METHOD_CALL", VerifyStatus: "verified"},
	}

	runs := []*RunSummary{makeRun(time.Now(), patterns, 0, 0)}
	report := Analyze(runs, 2)

	pkg := report.Verify.ByLocationType["PACKAGE"]
	if pkg.Good != 2 || pkg.Bad != 0 {
		t.Errorf("PACKAGE: got %d/%d, want 2/0", pkg.Good, pkg.Bad)
	}

	mc := report.Verify.ByLocationType["METHOD_CALL"]
	if mc.Good != 1 || mc.Bad != 2 {
		t.Errorf("METHOD_CALL: got %d/%d, want 1/2", mc.Good, mc.Bad)
	}
}

func TestAnalyzeMinRunsThreshold(t *testing.T) {
	p := PatternOutcome{SourceFQN: "com.once", VerifyStatus: "not_found"}
	runs := []*RunSummary{makeRun(time.Now(), []PatternOutcome{p}, 0, 0)}

	report := Analyze(runs, 3)
	if len(report.Verify.RecurringFailures) != 0 {
		t.Error("should not flag FQN below minRuns threshold")
	}
}

func TestAnalyzeTestBreakdowns(t *testing.T) {
	patterns := []PatternOutcome{
		{LocationType: "PACKAGE", Complexity: "low", TestStatus: "passed"},
		{LocationType: "PACKAGE", Complexity: "low", TestStatus: "passed"},
		{LocationType: "METHOD_CALL", Complexity: "high", TestStatus: "failed"},
	}
	runs := []*RunSummary{makeRun(time.Now(), patterns, 2, 1)}
	report := Analyze(runs, 2)

	if report.Tests.TotalPassed != 2 {
		t.Errorf("TotalPassed: got %d, want 2", report.Tests.TotalPassed)
	}
	if report.Tests.TotalFailed != 1 {
		t.Errorf("TotalFailed: got %d, want 1", report.Tests.TotalFailed)
	}

	hi := report.Tests.ByComplexity["high"]
	if hi.Good != 0 || hi.Bad != 1 {
		t.Errorf("high complexity: got %d/%d, want 0/1", hi.Good, hi.Bad)
	}
}
