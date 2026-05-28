package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DeterministicSnapshot holds the metrics that CI gates on.
type DeterministicSnapshot struct {
	SchemaVersion        int      `json:"schema_version"`
	Timestamp            string   `json:"timestamp"`
	Migration            string   `json:"migration"`
	RuleCount            int      `json:"rule_count"`
	EffectiveCoveragePct int      `json:"effective_coverage_pct"`
	QualityAvg           float64  `json:"quality_avg"`
	GuidanceDepthAvg     float64  `json:"guidance_depth_avg"`
	OverlapConflictCount      *int     `json:"overlap_conflict_count"`
	SpecificityGapCount       *int     `json:"specificity_gap_count"`
	GuideSpecificityGapCount  *int     `json:"guide_specificity_gap_count"`
}

// MetricDelta describes one metric's change between runs.
type MetricDelta struct {
	Name   string `json:"name"`
	Old    any    `json:"old"`
	New    any    `json:"new"`
	Status string `json:"status"`
}

// CompareResult holds the output of comparing two snapshots.
type CompareResult struct {
	MetricDeltas []MetricDelta `json:"metric_deltas"`
	Verdict      string        `json:"verdict"`
}

const currentSchemaVersion = 1

// SnapshotFromResult extracts a deterministic snapshot from an eval result.
func SnapshotFromResult(r *EvalResult, migration string) *DeterministicSnapshot {
	s := &DeterministicSnapshot{
		SchemaVersion:    currentSchemaVersion,
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Migration:        migration,
		RuleCount:        r.RuleCount,
		QualityAvg:       r.Quality.AvgScore,
		GuidanceDepthAvg: r.Quality.GuidanceDepthAvg,
	}

	if r.AppCoverage != nil {
		s.EffectiveCoveragePct = r.AppCoverage.EffectivePct
		gapCount := len(r.AppCoverage.SpecificityGaps)
		s.SpecificityGapCount = &gapCount
	}

	overlapCount := len(r.Overlaps)
	s.OverlapConflictCount = &overlapCount

	if len(r.GuideSpecificityGaps) > 0 {
		guideGapCount := len(r.GuideSpecificityGaps)
		s.GuideSpecificityGapCount = &guideGapCount
	}

	return s
}

// SaveSnapshot writes a snapshot to the given path, creating directories as needed.
func SaveSnapshot(snapshot *DeterministicSnapshot, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing snapshot %s: %w", path, err)
	}
	return nil
}

// LoadSnapshot reads a snapshot from disk.
func LoadSnapshot(path string) (*DeterministicSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot %s: %w", path, err)
	}

	var s DeterministicSnapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing snapshot %s: %w", path, err)
	}
	return &s, nil
}

// Compare computes deltas between current and baseline snapshots.
func Compare(current, baseline *DeterministicSnapshot, thresholds CIThresholds) CompareResult {
	if current.SchemaVersion != baseline.SchemaVersion {
		return CompareResult{
			Verdict: "FAIL",
			MetricDeltas: []MetricDelta{{
				Name:   "schema_version",
				Old:    baseline.SchemaVersion,
				New:    current.SchemaVersion,
				Status: "INCOMPATIBLE",
			}},
		}
	}

	var deltas []MetricDelta
	verdict := "PASS"

	coverageDelta := metricDelta("effective_coverage_pct",
		baseline.EffectiveCoveragePct, current.EffectiveCoveragePct)
	if coverageDrop := baseline.EffectiveCoveragePct - current.EffectiveCoveragePct; coverageDrop > thresholds.EffectiveCoverageDropPct {
		coverageDelta.Status = "REGRESSED"
		verdict = "FAIL"
	}
	deltas = append(deltas, coverageDelta)

	qualityDelta := metricDelta("quality_avg", baseline.QualityAvg, current.QualityAvg)
	if qualityDrop := baseline.QualityAvg - current.QualityAvg; qualityDrop > thresholds.QualityAvgDrop {
		qualityDelta.Status = "REGRESSED"
		if verdict != "FAIL" {
			verdict = "FAIL"
		}
	}
	deltas = append(deltas, qualityDelta)

	depthDelta := metricDelta("guidance_depth_avg", baseline.GuidanceDepthAvg, current.GuidanceDepthAvg)
	if current.GuidanceDepthAvg < baseline.GuidanceDepthAvg {
		depthDelta.Status = "WARN"
		if verdict == "PASS" {
			verdict = "REVIEW"
		}
	}
	deltas = append(deltas, depthDelta)

	if baseline.OverlapConflictCount != nil && current.OverlapConflictCount != nil {
		overlapDelta := metricDelta("overlap_conflict_count",
			*baseline.OverlapConflictCount, *current.OverlapConflictCount)
		if *current.OverlapConflictCount > *baseline.OverlapConflictCount {
			overlapDelta.Status = "WARN"
			if verdict == "PASS" {
				verdict = "REVIEW"
			}
		}
		deltas = append(deltas, overlapDelta)
	}

	if baseline.SpecificityGapCount != nil && current.SpecificityGapCount != nil {
		gapDelta := metricDelta("specificity_gap_count",
			*baseline.SpecificityGapCount, *current.SpecificityGapCount)
		if *current.SpecificityGapCount > *baseline.SpecificityGapCount {
			gapDelta.Status = "WARN"
			if verdict == "PASS" {
				verdict = "REVIEW"
			}
		}
		deltas = append(deltas, gapDelta)
	}

	if baseline.GuideSpecificityGapCount != nil && current.GuideSpecificityGapCount != nil {
		guideGapDelta := metricDelta("guide_specificity_gap_count",
			*baseline.GuideSpecificityGapCount, *current.GuideSpecificityGapCount)
		if *current.GuideSpecificityGapCount > *baseline.GuideSpecificityGapCount {
			guideGapDelta.Status = "WARN"
			if verdict == "PASS" {
				verdict = "REVIEW"
			}
		}
		deltas = append(deltas, guideGapDelta)
	}

	ruleCountDelta := metricDelta("rule_count", baseline.RuleCount, current.RuleCount)
	if current.RuleCount < baseline.RuleCount {
		ruleCountDelta.Status = "WARN"
		if verdict == "PASS" {
			verdict = "REVIEW"
		}
	}
	deltas = append(deltas, ruleCountDelta)

	return CompareResult{
		MetricDeltas: deltas,
		Verdict:      verdict,
	}
}

func metricDelta[T comparable](name string, old, new T) MetricDelta {
	status := "OK"
	if old != new {
		status = "CHANGED"
	}
	return MetricDelta{Name: name, Old: old, New: new, Status: status}
}

// PrintCompare writes a human-readable comparison to stderr.
func PrintCompare(cr CompareResult, migration string) {
	fmt.Fprintf(os.Stderr, "REGRESSION CHECK: %s\n", migration)
	for _, d := range cr.MetricDeltas {
		fmt.Fprintf(os.Stderr, "  %-25s %v -> %v  %s\n", d.Name+":", d.Old, d.New, d.Status)
	}
	fmt.Fprintf(os.Stderr, "  Verdict: %s\n", cr.Verdict)
}
