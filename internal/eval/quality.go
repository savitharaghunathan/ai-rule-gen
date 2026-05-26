package eval

import (
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// ScoreRule checks a rule for quality signals and returns a detail record.
func ScoreRule(r rules.Rule) RuleDetail {
	var score int
	var missing []string
	const maxScore = 4

	if strings.TrimSpace(r.Message) != "" {
		score++
	} else {
		missing = append(missing, "message")
	}

	if len(r.Links) > 0 {
		score++
	} else {
		missing = append(missing, "links")
	}

	if r.Effort > 0 {
		score++
	} else {
		missing = append(missing, "effort")
	}

	msg := strings.ToLower(r.Message)
	hasGuidance := strings.Contains(msg, "replace ") ||
		strings.Contains(msg, "instead of") ||
		strings.Contains(msg, "renamed to") ||
		strings.Contains(msg, "use `") ||
		strings.Contains(msg, "before") ||
		strings.Contains(msg, "after")
	if hasGuidance {
		score++
	} else {
		missing = append(missing, "before_after_guidance")
	}

	return RuleDetail{
		RuleID:       r.RuleID,
		Description:  r.Description,
		QualityScore: score,
		QualityMax:   maxScore,
		HasGuidance:  hasGuidance,
		Missing:      missing,
	}
}

// ScoreAll scores all rules and produces a summary.
func ScoreAll(ruleList []rules.Rule) (QualitySummary, []RuleDetail) {
	var details []RuleDetail
	var totalScore int

	summary := QualitySummary{
		TotalRules: len(ruleList),
		MaxScore:   4,
	}

	for _, r := range ruleList {
		d := ScoreRule(r)
		details = append(details, d)
		totalScore += d.QualityScore

		if strings.TrimSpace(r.Message) != "" {
			summary.HasMessage++
		}
		if len(r.Links) > 0 {
			summary.HasLinks++
		}
		if r.Effort > 0 {
			summary.HasEffort++
		}
		if d.HasGuidance {
			summary.HasBeforeAfter++
		}
	}

	if len(ruleList) > 0 {
		summary.AvgScore = float64(totalScore) / float64(len(ruleList))
	}

	return summary, details
}
