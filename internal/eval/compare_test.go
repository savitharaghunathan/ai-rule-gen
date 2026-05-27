package eval

import (
	"path/filepath"
	"testing"
)

func TestSnapshotFromResult(t *testing.T) {
	result := &EvalResult{
		RuleCount: 10,
		Quality: QualitySummary{
			AvgScore:         4.5,
			GuidanceDepthAvg: 2.1,
		},
		AppCoverage: &AppCoverage{
			EffectivePct: 90,
		},
		Overlaps: []Overlap{{RuleA: "a", RuleB: "b"}},
	}

	s := SnapshotFromResult(result, "httpclient4-to-httpclient5")

	if s.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", s.SchemaVersion)
	}
	if s.Migration != "httpclient4-to-httpclient5" {
		t.Errorf("Migration = %q", s.Migration)
	}
	if s.RuleCount != 10 {
		t.Errorf("RuleCount = %d, want 10", s.RuleCount)
	}
	if s.EffectiveCoveragePct != 90 {
		t.Errorf("EffectiveCoveragePct = %d, want 90", s.EffectiveCoveragePct)
	}
	if s.QualityAvg != 4.5 {
		t.Errorf("QualityAvg = %f, want 4.5", s.QualityAvg)
	}
	if s.GuidanceDepthAvg != 2.1 {
		t.Errorf("GuidanceDepthAvg = %f, want 2.1", s.GuidanceDepthAvg)
	}
	if s.OverlapConflictCount == nil || *s.OverlapConflictCount != 1 {
		t.Errorf("OverlapConflictCount = %v, want 1", s.OverlapConflictCount)
	}
	if s.Timestamp == "" {
		t.Error("Timestamp is empty")
	}
}

func TestSnapshotFromResult_NoCoverage(t *testing.T) {
	result := &EvalResult{
		RuleCount: 5,
		Quality: QualitySummary{
			AvgScore: 3.0,
		},
	}

	s := SnapshotFromResult(result, "test")
	if s.EffectiveCoveragePct != 0 {
		t.Errorf("EffectiveCoveragePct = %d, want 0", s.EffectiveCoveragePct)
	}
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runs", "test.json")

	original := &DeterministicSnapshot{
		SchemaVersion:    1,
		Timestamp:        "2026-05-27T12:00:00Z",
		Migration:        "test-migration",
		RuleCount:        20,
		EffectiveCoveragePct: 85,
		QualityAvg:       5.0,
		GuidanceDepthAvg: 2.5,
	}
	overlapCount := 3
	original.OverlapConflictCount = &overlapCount

	if err := SaveSnapshot(original, path); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	if loaded.SchemaVersion != original.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", loaded.SchemaVersion, original.SchemaVersion)
	}
	if loaded.Migration != original.Migration {
		t.Errorf("Migration = %q, want %q", loaded.Migration, original.Migration)
	}
	if loaded.RuleCount != original.RuleCount {
		t.Errorf("RuleCount = %d, want %d", loaded.RuleCount, original.RuleCount)
	}
	if loaded.EffectiveCoveragePct != original.EffectiveCoveragePct {
		t.Errorf("EffectiveCoveragePct = %d, want %d", loaded.EffectiveCoveragePct, original.EffectiveCoveragePct)
	}
	if loaded.QualityAvg != original.QualityAvg {
		t.Errorf("QualityAvg = %f, want %f", loaded.QualityAvg, original.QualityAvg)
	}
	if loaded.OverlapConflictCount == nil || *loaded.OverlapConflictCount != 3 {
		t.Errorf("OverlapConflictCount = %v, want 3", loaded.OverlapConflictCount)
	}
}

func TestLoadSnapshot_NotFound(t *testing.T) {
	_, err := LoadSnapshot("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCompare_NoRegression(t *testing.T) {
	baseline := &DeterministicSnapshot{
		SchemaVersion:        1,
		RuleCount:            10,
		EffectiveCoveragePct: 90,
		QualityAvg:           4.5,
		GuidanceDepthAvg:     2.0,
	}
	overlapZero := 0
	baseline.OverlapConflictCount = &overlapZero

	current := &DeterministicSnapshot{
		SchemaVersion:        1,
		RuleCount:            10,
		EffectiveCoveragePct: 92,
		QualityAvg:           4.5,
		GuidanceDepthAvg:     2.1,
	}
	current.OverlapConflictCount = &overlapZero

	cr := Compare(current, baseline, DefaultThresholds())
	if cr.Verdict != "PASS" {
		t.Errorf("Verdict = %q, want PASS", cr.Verdict)
	}
}

func TestCompare_CoverageRegression(t *testing.T) {
	baseline := &DeterministicSnapshot{
		SchemaVersion:        1,
		EffectiveCoveragePct: 95,
		QualityAvg:           4.5,
	}
	current := &DeterministicSnapshot{
		SchemaVersion:        1,
		EffectiveCoveragePct: 85,
		QualityAvg:           4.5,
	}

	cr := Compare(current, baseline, DefaultThresholds())
	if cr.Verdict != "FAIL" {
		t.Errorf("Verdict = %q, want FAIL (coverage dropped 10%%)", cr.Verdict)
	}

	found := false
	for _, d := range cr.MetricDeltas {
		if d.Name == "effective_coverage_pct" && d.Status == "REGRESSED" {
			found = true
		}
	}
	if !found {
		t.Error("expected REGRESSED status on effective_coverage_pct")
	}
}

func TestCompare_QualityRegression(t *testing.T) {
	baseline := &DeterministicSnapshot{
		SchemaVersion: 1,
		QualityAvg:    4.5,
	}
	current := &DeterministicSnapshot{
		SchemaVersion: 1,
		QualityAvg:    4.0,
	}

	cr := Compare(current, baseline, DefaultThresholds())
	if cr.Verdict != "FAIL" {
		t.Errorf("Verdict = %q, want FAIL (quality dropped)", cr.Verdict)
	}
}

func TestCompare_GuidanceDepthWarn(t *testing.T) {
	baseline := &DeterministicSnapshot{
		SchemaVersion:    1,
		GuidanceDepthAvg: 2.5,
		QualityAvg:       4.5,
	}
	current := &DeterministicSnapshot{
		SchemaVersion:    1,
		GuidanceDepthAvg: 2.0,
		QualityAvg:       4.5,
	}

	cr := Compare(current, baseline, DefaultThresholds())
	if cr.Verdict != "REVIEW" {
		t.Errorf("Verdict = %q, want REVIEW (guidance depth dropped)", cr.Verdict)
	}
}

func TestCompare_SchemaMismatch(t *testing.T) {
	baseline := &DeterministicSnapshot{SchemaVersion: 1}
	current := &DeterministicSnapshot{SchemaVersion: 2}

	cr := Compare(current, baseline, DefaultThresholds())
	if cr.Verdict != "FAIL" {
		t.Errorf("Verdict = %q, want FAIL (schema mismatch)", cr.Verdict)
	}
	if len(cr.MetricDeltas) != 1 || cr.MetricDeltas[0].Status != "INCOMPATIBLE" {
		t.Errorf("expected INCOMPATIBLE delta, got %v", cr.MetricDeltas)
	}
}

func TestCompare_WithinThreshold(t *testing.T) {
	baseline := &DeterministicSnapshot{
		SchemaVersion:        1,
		EffectiveCoveragePct: 95,
		QualityAvg:           4.5,
	}
	current := &DeterministicSnapshot{
		SchemaVersion:        1,
		EffectiveCoveragePct: 91,
		QualityAvg:           4.5,
	}

	cr := Compare(current, baseline, DefaultThresholds())
	if cr.Verdict != "PASS" {
		t.Errorf("Verdict = %q, want PASS (4%% drop within 5%% threshold)", cr.Verdict)
	}
}

func TestCompare_CustomThresholds(t *testing.T) {
	baseline := &DeterministicSnapshot{
		SchemaVersion:        1,
		EffectiveCoveragePct: 95,
		QualityAvg:           4.5,
	}
	current := &DeterministicSnapshot{
		SchemaVersion:        1,
		EffectiveCoveragePct: 91,
		QualityAvg:           4.5,
	}

	tight := CIThresholds{EffectiveCoverageDropPct: 2}
	cr := Compare(current, baseline, tight)
	if cr.Verdict != "FAIL" {
		t.Errorf("Verdict = %q, want FAIL (4%% drop exceeds 2%% threshold)", cr.Verdict)
	}
}
