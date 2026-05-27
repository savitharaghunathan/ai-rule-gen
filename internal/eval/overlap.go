package eval

import (
	"sort"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// Overlap describes a detected overlap between two rules.
type Overlap struct {
	RuleA       string   `json:"rule_a"`
	RuleB       string   `json:"rule_b"`
	Type        string   `json:"type"`
	SharedFiles []string `json:"shared_files,omitempty"`
	Reason      string   `json:"reason"`
}

func extractPatternAndProvider(r rules.Rule) (pattern, provider string) {
	c := r.When
	if c.JavaReferenced != nil {
		return c.JavaReferenced.Pattern, "java.referenced"
	}
	if c.GoReferenced != nil {
		return c.GoReferenced.Pattern, "go.referenced"
	}
	if c.NodejsReferenced != nil {
		return c.NodejsReferenced.Pattern, "nodejs.referenced"
	}
	if c.CSharpReferenced != nil {
		return c.CSharpReferenced.Pattern, "csharp.referenced"
	}
	if c.PythonReferenced != nil {
		return c.PythonReferenced.Pattern, "python.referenced"
	}
	if c.BuiltinFilecontent != nil {
		return c.BuiltinFilecontent.Pattern, "builtin.filecontent"
	}
	if c.BuiltinXML != nil {
		return c.BuiltinXML.XPath, "builtin.xml"
	}
	if c.JavaDependency != nil {
		return c.JavaDependency.Name, "java.dependency"
	}
	if c.GoDependency != nil {
		return c.GoDependency.Name, "go.dependency"
	}
	return "", ""
}

// DetectPatternOverlaps finds rule pairs where one pattern is a substring of the other.
func DetectPatternOverlaps(ruleList []rules.Rule) []Overlap {
	type entry struct {
		ruleID   string
		pattern  string
		provider string
	}

	var entries []entry
	for _, r := range ruleList {
		pat, prov := extractPatternAndProvider(r)
		if pat != "" && prov != "" {
			entries = append(entries, entry{r.RuleID, pat, prov})
		}
	}

	var overlaps []Overlap
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			a, b := entries[i], entries[j]
			if a.provider != b.provider {
				continue
			}
			if a.pattern == b.pattern {
				overlaps = append(overlaps, Overlap{
					RuleA:  a.ruleID,
					RuleB:  b.ruleID,
					Type:   "duplicate_pattern",
					Reason: "identical pattern: " + a.pattern,
				})
			} else if strings.Contains(a.pattern, b.pattern) {
				overlaps = append(overlaps, Overlap{
					RuleA:  a.ruleID,
					RuleB:  b.ruleID,
					Type:   "pattern_overlap",
					Reason: b.ruleID + " pattern is substring of " + a.ruleID,
				})
			} else if strings.Contains(b.pattern, a.pattern) {
				overlaps = append(overlaps, Overlap{
					RuleA:  a.ruleID,
					RuleB:  b.ruleID,
					Type:   "pattern_overlap",
					Reason: a.ruleID + " pattern is substring of " + b.ruleID,
				})
			}
		}
	}
	return overlaps
}

// DetectIncidentOverlaps finds rule pairs that fire on the same files.
func DetectIncidentOverlaps(ruleList []rules.Rule, cov *AppCoverage) []Overlap {
	if cov == nil || len(cov.Violations) == 0 {
		return nil
	}

	type ruleFiles struct {
		ruleID string
		files  map[string]bool
	}
	var fired []ruleFiles
	for _, r := range ruleList {
		v, ok := cov.Violations[r.RuleID]
		if !ok || len(v.Files) == 0 {
			continue
		}
		fm := make(map[string]bool, len(v.Files))
		for _, f := range v.Files {
			fm[f] = true
		}
		fired = append(fired, ruleFiles{r.RuleID, fm})
	}

	var overlaps []Overlap
	for i := 0; i < len(fired); i++ {
		for j := i + 1; j < len(fired); j++ {
			a, b := fired[i], fired[j]
			var shared []string
			for f := range a.files {
				if b.files[f] {
					shared = append(shared, f)
				}
			}
			if len(shared) == 0 {
				continue
			}
			sort.Strings(shared)
			overlaps = append(overlaps, Overlap{
				RuleA:       a.ruleID,
				RuleB:       b.ruleID,
				Type:        "incident_overlap",
				SharedFiles: shared,
				Reason:      "both rules fire on same file(s)",
			})
		}
	}
	return overlaps
}

// DetectOverlaps runs both pattern and incident overlap detection.
func DetectOverlaps(ruleList []rules.Rule, cov *AppCoverage) []Overlap {
	overlaps := DetectPatternOverlaps(ruleList)
	if cov != nil {
		overlaps = append(overlaps, DetectIncidentOverlaps(ruleList, cov)...)
	}
	return overlaps
}
