package eval

import (
	"regexp"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

var backtickRe = regexp.MustCompile("`([^`]+)`")

var actionablePattern = regexp.MustCompile(`(?i)\breplace\b|\binstead of\b|\brenamed to\b|(?i)\bbefore\b.*\bafter\b`)

func extractConditionPattern(r rules.Rule) string {
	c := r.When
	if c.JavaReferenced != nil {
		return c.JavaReferenced.Pattern
	}
	if c.GoReferenced != nil {
		return c.GoReferenced.Pattern
	}
	if c.NodejsReferenced != nil {
		return c.NodejsReferenced.Pattern
	}
	if c.CSharpReferenced != nil {
		return c.CSharpReferenced.Pattern
	}
	if c.PythonReferenced != nil {
		return c.PythonReferenced.Pattern
	}
	if c.JavaDependency != nil {
		return c.JavaDependency.Name
	}
	if c.GoDependency != nil {
		return c.GoDependency.Name
	}
	if c.BuiltinFilecontent != nil {
		return c.BuiltinFilecontent.Pattern
	}
	if c.BuiltinXML != nil {
		return c.BuiltinXML.XPath
	}
	return ""
}

func conditionLastSegment(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	parts := strings.Split(pattern, ".")
	last := parts[len(parts)-1]
	if last == "" || last == "*" {
		return ""
	}
	return last
}

func scoreGuidanceDepth(message string, conditionLastSegment string) int {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return 0
	}

	backticks := backtickRe.FindAllStringSubmatch(msg, -1)
	hasBacktick := len(backticks) > 0
	hasActionable := actionablePattern.MatchString(msg)

	if hasBacktick {
		for _, m := range backticks {
			quoted := stripTrailingParens(m[1])
			if isValidIdentifier(quoted) {
				if conditionLastSegment == "" || !strings.EqualFold(quoted, conditionLastSegment) {
					return 3
				}
			}
		}
	}

	if hasBacktick || hasActionable {
		return 2
	}

	return 1
}

func stripTrailingParens(s string) string {
	if strings.HasSuffix(s, "()") {
		return s[:len(s)-2]
	}
	return s
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if r == '_' || r == '.' || r == '$' {
			continue
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// ScoreRule checks a rule for quality signals and returns a detail record.
func ScoreRule(r rules.Rule) RuleDetail {
	var score int
	var missing []string
	const maxScore = 6

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

	pattern := extractConditionPattern(r)
	lastSeg := conditionLastSegment(pattern)
	depth := scoreGuidanceDepth(r.Message, lastSeg)
	score += depth

	hasGuidance := depth >= 2
	if !hasGuidance {
		missing = append(missing, "before_after_guidance")
	}

	return RuleDetail{
		RuleID:        r.RuleID,
		Description:   r.Description,
		QualityScore:  score,
		QualityMax:    maxScore,
		HasGuidance:   hasGuidance,
		GuidanceDepth: depth,
		Missing:       missing,
	}
}

// ScoreAll scores all rules and produces a summary.
func ScoreAll(ruleList []rules.Rule) (QualitySummary, []RuleDetail) {
	var details []RuleDetail
	var totalScore int
	var totalDepth int

	summary := QualitySummary{
		TotalRules: len(ruleList),
		MaxScore:   6,
	}

	for _, r := range ruleList {
		d := ScoreRule(r)
		details = append(details, d)
		totalScore += d.QualityScore
		totalDepth += d.GuidanceDepth

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
		summary.GuidanceDepthAvg = float64(totalDepth) / float64(len(ruleList))
	}

	return summary, details
}
