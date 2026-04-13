package rules

import "strings"

// Label key constants following the konveyor.io/ convention.
const (
	LabelGeneratedBy = "konveyor.io/generated-by"
	LabelTestResult  = "konveyor.io/test-result"
	LabelReview      = "konveyor.io/review"
)

// Label value constants.
const (
	GeneratedByValue = "ai-rule-gen"

	TestResultUntested = "untested"
	TestResultPassed   = "passed"
	TestResultFailed   = "failed"

	ReviewUnreviewed = "unreviewed"
	ReviewApproved   = "approved"
	ReviewRejected   = "rejected"
)

// InitialLabels returns the default labels for a newly generated rule,
// combined with any existing labels (e.g., source/target).
func InitialLabels(existing []string) []string {
	labels := make([]string, len(existing))
	copy(labels, existing)
	labels = SetLabel(labels, LabelGeneratedBy, GeneratedByValue)
	labels = SetLabel(labels, LabelTestResult, TestResultUntested)
	labels = SetLabel(labels, LabelReview, ReviewUnreviewed)
	return labels
}

// SetLabel sets a konveyor.io/ label to the given value, replacing any
// existing label with the same key. If not present, appends it.
func SetLabel(labels []string, key, value string) []string {
	prefix := key + "="
	for i, l := range labels {
		if strings.HasPrefix(l, prefix) {
			labels[i] = prefix + value
			return labels
		}
	}
	return append(labels, prefix+value)
}

// GetLabel returns the value for a konveyor.io/ label key, or "" if not found.
func GetLabel(labels []string, key string) string {
	prefix := key + "="
	for _, l := range labels {
		if v, ok := strings.CutPrefix(l, prefix); ok {
			return v
		}
	}
	return ""
}

// StampTestResults updates test-result labels on rules based on kantra results.
// passedIDs and failedIDs are sets of rule IDs that passed/failed.
func StampTestResults(ruleList []Rule, passedIDs, failedIDs map[string]bool) []Rule {
	for i := range ruleList {
		id := ruleList[i].RuleID
		if passedIDs[id] {
			ruleList[i].Labels = SetLabel(ruleList[i].Labels, LabelTestResult, TestResultPassed)
		} else if failedIDs[id] {
			ruleList[i].Labels = SetLabel(ruleList[i].Labels, LabelTestResult, TestResultFailed)
		}
	}
	return ruleList
}
