package groundtruth

import "time"

// ConvertChanges converts japicmp APIChange entries into ground truth Entry entries.
func ConvertChanges(changes []APIChange) []Entry {
	today := time.Now().UTC().Format("2006-01-02")
	var entries []Entry

	for _, c := range changes {
		e := Entry{
			OldAPI:       c.OldAPI,
			NewAPI:       "",
			ActionType:   mapActionType(c.ChangeKind),
			Severity:     mapSeverity(c.ChangeKind),
			GuideSection: "",
			SourceQuote:  "",
			ReviewedBy:   "japicmp",
			ReviewedDate: today,
		}
		entries = append(entries, e)
	}
	return entries
}

func mapActionType(changeKind string) string {
	switch changeKind {
	case "class_removed":
		return "package_change"
	case "method_removed":
		return "method_removal"
	case "method_changed":
		return "signature_change"
	case "field_removed":
		return "config_change"
	default:
		return "behavioral_change"
	}
}

func mapSeverity(changeKind string) string {
	switch changeKind {
	case "class_removed", "method_removed", "method_changed":
		return "high"
	case "field_removed":
		return "medium"
	default:
		return "low"
	}
}
