package feedback

import (
	"fmt"
	"strings"
)

// Recommend generates actionable recommendations from analysis results.
func Recommend(report *FeedbackReport) []Recommendation {
	var recs []Recommendation

	recs = append(recs, recurringFQNRecs(report)...)
	recs = append(recs, verifyRateRecs(report)...)
	recs = append(recs, locationTypeBiasRecs(report)...)
	recs = append(recs, trendRecs(report)...)

	return recs
}

func recurringFQNRecs(report *FeedbackReport) []Recommendation {
	if len(report.Verify.RecurringFailures) == 0 {
		return nil
	}

	var sb strings.Builder
	for i, rf := range report.Verify.RecurringFailures {
		if i >= 5 {
			fmt.Fprintf(&sb, "  ... and %d more\n", len(report.Verify.RecurringFailures)-5)
			break
		}
		fmt.Fprintf(&sb, "  - %s (not_found in %d/%d runs)\n", rf.SourceFQN, rf.FailCount, rf.Occurrences)
	}
	examples := sb.String()

	return []Recommendation{{
		Severity: "high",
		Category: "prompt",
		Title:    "Recurring verification failures",
		Description: fmt.Sprintf(
			"%d source FQNs consistently fail verification across multiple runs. "+
				"The LLM is generating plausible but incorrect fully-qualified names.",
			len(report.Verify.RecurringFailures)),
		Evidence: examples,
		Action: "Add these FQNs as negative examples in agents/rule-writer/references/. " +
			"Consider adding canonical FQN examples from verified patterns to the prompt.",
	}}
}

func verifyRateRecs(report *FeedbackReport) []Recommendation {
	if report.Overall.AverageVerifyRate == 0 {
		return nil
	}
	if report.Overall.AverageVerifyRate >= 50 {
		return nil
	}

	return []Recommendation{{
		Severity: "high",
		Category: "reference_doc",
		Title:    "Low overall verification rate",
		Description: fmt.Sprintf(
			"Only %.0f%% of generated FQNs pass source verification. "+
				"More than half of patterns reference names that don't exist in published artifacts.",
			report.Overall.AverageVerifyRate),
		Evidence: fmt.Sprintf("Verified: %d, Not found: %d across %d runs",
			report.Verify.TotalVerified, report.Verify.TotalNotFound, report.Overall.TotalRuns),
		Action: "Update agents/rule-writer/references/ with canonical FQN examples " +
			"from actual JAR contents. Emphasize source_artifact coordinates in the patterns schema.",
	}}
}

func locationTypeBiasRecs(report *FeedbackReport) []Recommendation {
	pkg, hasPkg := report.Verify.ByLocationType["PACKAGE"]
	mc, hasMC := report.Verify.ByLocationType["METHOD_CALL"]

	if !hasPkg || !hasMC || (pkg.Good+pkg.Bad) < 3 || (mc.Good+mc.Bad) < 3 {
		return nil
	}

	if pkg.Value-mc.Value < 30 {
		return nil
	}

	return []Recommendation{{
		Severity: "medium",
		Category: "prompt",
		Title:    "METHOD_CALL patterns verify poorly compared to PACKAGE",
		Description: fmt.Sprintf(
			"PACKAGE patterns verify at %.0f%% but METHOD_CALL at %.0f%%. "+
				"The LLM likely guesses method-level FQNs that don't match JAR contents.",
			pkg.Value, mc.Value),
		Evidence: fmt.Sprintf("PACKAGE: %d/%d verified, METHOD_CALL: %d/%d verified",
			pkg.Good, pkg.Good+pkg.Bad, mc.Good, mc.Good+mc.Bad),
		Action: "For METHOD_CALL patterns where verification fails on the FQN, consider using short method name " +
			"patterns (e.g., 'closeExpiredConnections' instead of 'org.example.MyClass.closeExpiredConnections') " +
			"to handle type hierarchy and builder chain scenarios. See condition-types.md for the decision framework.",
	}}
}

func trendRecs(report *FeedbackReport) []Recommendation {
	if report.Overall.PassRateTrend != "declining" {
		return nil
	}

	return []Recommendation{{
		Severity:    "high",
		Category:    "pipeline",
		Title:       "Pass rate declining over recent runs",
		Description: "Test pass rate is trending downward across recent runs.",
		Evidence:    fmt.Sprintf("Trend: %s across %d runs", report.Overall.PassRateTrend, report.Overall.TotalRuns),
		Action:      "Review recent changes to agent prompts, reference docs, or pipeline code that may have caused regression.",
	}}
}
