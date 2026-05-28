package feedback

import (
	"fmt"
	"sort"
	"strings"
)

// fqnRecord tracks outcomes for a single source_fqn across runs.
type fqnRecord struct {
	SourceFQN   string
	Verified    int
	NotFound    int
	Passed      int
	Failed      int
	Occurrences int
}

// Analyze aggregates data from multiple runs and produces a FeedbackReport.
func Analyze(runs []*RunSummary, minRuns int) *FeedbackReport {
	report := &FeedbackReport{
		RunsAnalyzed: len(runs),
	}
	if len(runs) == 0 {
		return report
	}

	report.DateRange = dateRange(runs)
	report.MigrationPaths = migrationPaths(runs)
	report.Overall = overallStats(runs)
	report.Verify = verifyAnalysis(runs, minRuns)
	report.Tests = testAnalysis(runs)

	return report
}

func dateRange(runs []*RunSummary) string {
	if len(runs) == 0 {
		return ""
	}
	earliest := runs[0].Timestamp
	latest := runs[0].Timestamp
	for _, r := range runs[1:] {
		if !r.Timestamp.IsZero() && (earliest.IsZero() || r.Timestamp.Before(earliest)) {
			earliest = r.Timestamp
		}
		if r.Timestamp.After(latest) {
			latest = r.Timestamp
		}
	}
	if earliest.IsZero() {
		return "unknown"
	}
	return fmt.Sprintf("%s to %s", earliest.Format("2006-01-02"), latest.Format("2006-01-02"))
}

func migrationPaths(runs []*RunSummary) []string {
	seen := map[string]bool{}
	var paths []string
	for _, r := range runs {
		key := strings.Join(r.Sources, "+") + " -> " + strings.Join(r.Targets, "+")
		if !seen[key] {
			seen[key] = true
			paths = append(paths, key)
		}
	}
	return paths
}

func overallStats(runs []*RunSummary) OverallStats {
	stats := OverallStats{TotalRuns: len(runs)}

	var totalPassRate, totalRules float64
	var verifiedTotal, verifyAttempts int
	testedRuns := 0

	for _, r := range runs {
		totalRules += float64(r.RulesTotal)
		if r.TestsPassed+r.TestsFailed > 0 {
			totalPassRate += r.PassRate
			testedRuns++
		}
		for _, p := range r.Patterns {
			switch p.VerifyStatus {
			case "verified":
				verifiedTotal++
				verifyAttempts++
			case "not_found":
				verifyAttempts++
			}
		}
	}

	if testedRuns > 0 {
		stats.AveragePassRate = totalPassRate / float64(testedRuns)
	}
	stats.AverageRulesPerRun = totalRules / float64(len(runs))
	if verifyAttempts > 0 {
		stats.AverageVerifyRate = float64(verifiedTotal) / float64(verifyAttempts) * 100
	}

	stats.PassRateTrend = passRateTrend(runs)

	return stats
}

func passRateTrend(runs []*RunSummary) string {
	var tested []*RunSummary
	for _, r := range runs {
		if r.TestsPassed+r.TestsFailed > 0 {
			tested = append(tested, r)
		}
	}
	if len(tested) < 4 {
		return "insufficient data"
	}

	sort.Slice(tested, func(i, j int) bool {
		return tested[i].Timestamp.Before(tested[j].Timestamp)
	})

	mid := len(tested) / 2
	firstHalf := avgPassRate(tested[:mid])
	secondHalf := avgPassRate(tested[mid:])

	diff := secondHalf - firstHalf
	switch {
	case diff > 5:
		return "improving"
	case diff < -5:
		return "declining"
	default:
		return "stable"
	}
}

func avgPassRate(runs []*RunSummary) float64 {
	if len(runs) == 0 {
		return 0
	}
	var total float64
	for _, r := range runs {
		total += r.PassRate
	}
	return total / float64(len(runs))
}

func verifyAnalysis(runs []*RunSummary, minRuns int) VerifyAnalysis {
	va := VerifyAnalysis{
		ByLocationType: make(map[string]Rate),
	}

	fqns := map[string]*fqnRecord{}

	for _, r := range runs {
		for _, p := range r.Patterns {
			switch p.VerifyStatus {
			case "verified":
				va.TotalVerified++
			case "not_found":
				va.TotalNotFound++
			case "skipped":
				va.TotalSkipped++
			default:
				continue
			}

			if p.SourceFQN != "" && (p.VerifyStatus == "verified" || p.VerifyStatus == "not_found") {
				rec, ok := fqns[p.SourceFQN]
				if !ok {
					rec = &fqnRecord{SourceFQN: p.SourceFQN}
					fqns[p.SourceFQN] = rec
				}
				rec.Occurrences++
				if p.VerifyStatus == "verified" {
					rec.Verified++
				} else {
					rec.NotFound++
				}
			}

			lt := p.LocationType
			if lt == "" {
				lt = "unknown"
			}
			rate := va.ByLocationType[lt]
			if p.VerifyStatus == "verified" {
				rate.Good++
			} else if p.VerifyStatus == "not_found" {
				rate.Bad++
			}
			va.ByLocationType[lt] = rate
		}
	}

	for lt, rate := range va.ByLocationType {
		total := rate.Good + rate.Bad
		if total > 0 {
			rate.Value = float64(rate.Good) / float64(total) * 100
		}
		va.ByLocationType[lt] = rate
	}

	for _, rec := range fqns {
		total := rec.Verified + rec.NotFound
		if rec.Occurrences >= minRuns && total > 0 {
			failRate := float64(rec.NotFound) / float64(total) * 100
			if failRate >= 80 {
				va.RecurringFailures = append(va.RecurringFailures, RecurringFQN{
					SourceFQN:   rec.SourceFQN,
					Occurrences: rec.Occurrences,
					FailCount:   rec.NotFound,
					VerifyCount: rec.Verified,
					FailRate:    failRate,
				})
			}
		}
	}

	sort.Slice(va.RecurringFailures, func(i, j int) bool {
		if va.RecurringFailures[i].Occurrences != va.RecurringFailures[j].Occurrences {
			return va.RecurringFailures[i].Occurrences > va.RecurringFailures[j].Occurrences
		}
		return va.RecurringFailures[i].SourceFQN < va.RecurringFailures[j].SourceFQN
	})

	return va
}

func testAnalysis(runs []*RunSummary) TestAnalysis {
	ta := TestAnalysis{
		ByLocationType: make(map[string]Rate),
		ByComplexity:   make(map[string]Rate),
	}

	for _, r := range runs {
		for _, p := range r.Patterns {
			switch p.TestStatus {
			case "passed":
				ta.TotalPassed++
			case "failed":
				ta.TotalFailed++
			case "kantra-limitation":
				ta.TotalKantraLimitation++
			default:
				continue
			}

			lt := p.LocationType
			if lt == "" {
				lt = "unknown"
			}
			rate := ta.ByLocationType[lt]
			if p.TestStatus == "passed" {
				rate.Good++
			} else if p.TestStatus == "failed" {
				rate.Bad++
			}
			ta.ByLocationType[lt] = rate

			cx := p.Complexity
			if cx == "" {
				cx = "unknown"
			}
			crate := ta.ByComplexity[cx]
			if p.TestStatus == "passed" {
				crate.Good++
			} else if p.TestStatus == "failed" {
				crate.Bad++
			}
			ta.ByComplexity[cx] = crate
		}
	}

	for lt, rate := range ta.ByLocationType {
		total := rate.Good + rate.Bad
		if total > 0 {
			rate.Value = float64(rate.Good) / float64(total) * 100
		}
		ta.ByLocationType[lt] = rate
	}
	for cx, rate := range ta.ByComplexity {
		total := rate.Good + rate.Bad
		if total > 0 {
			rate.Value = float64(rate.Good) / float64(total) * 100
		}
		ta.ByComplexity[cx] = rate
	}

	return ta
}
