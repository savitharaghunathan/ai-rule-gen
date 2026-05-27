package eval

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestDetectPatternOverlaps(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "broad-rule",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse"}},
		},
		{
			RuleID: "narrow-rule",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse.getStatusLine"}},
		},
		{
			RuleID: "unrelated-rule",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "com.example.SomeClass"}},
		},
	}

	overlaps := DetectPatternOverlaps(ruleList)
	if len(overlaps) != 1 {
		t.Fatalf("got %d overlaps, want 1", len(overlaps))
	}
	if overlaps[0].RuleA != "broad-rule" || overlaps[0].RuleB != "narrow-rule" {
		t.Errorf("overlap = (%s, %s), want (broad-rule, narrow-rule)", overlaps[0].RuleA, overlaps[0].RuleB)
	}
	if overlaps[0].Type != "pattern_overlap" {
		t.Errorf("type = %q, want pattern_overlap", overlaps[0].Type)
	}
}

func TestDetectPatternOverlaps_DuplicatePattern(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-a",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse"}},
		},
		{
			RuleID: "rule-b",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse"}},
		},
	}

	overlaps := DetectPatternOverlaps(ruleList)
	if len(overlaps) != 1 {
		t.Fatalf("got %d overlaps, want 1", len(overlaps))
	}
	if overlaps[0].Type != "duplicate_pattern" {
		t.Errorf("type = %q, want duplicate_pattern", overlaps[0].Type)
	}
}

func TestDetectPatternOverlaps_DifferentProviders(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "java-rule",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.example.Class"}},
		},
		{
			RuleID: "go-rule",
			When:   rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "org.example.Class"}},
		},
	}

	overlaps := DetectPatternOverlaps(ruleList)
	if len(overlaps) != 0 {
		t.Errorf("got %d overlaps, want 0 (different providers)", len(overlaps))
	}
}

func TestDetectIncidentOverlaps(t *testing.T) {
	ruleList := []rules.Rule{
		{RuleID: "rule-a"},
		{RuleID: "rule-b"},
		{RuleID: "rule-c"},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-a": {Incidents: 2, Files: []string{"App.java", "Service.java"}},
			"rule-b": {Incidents: 1, Files: []string{"App.java"}},
			"rule-c": {Incidents: 1, Files: []string{"Other.java"}},
		},
	}

	overlaps := DetectIncidentOverlaps(ruleList, cov)
	if len(overlaps) != 1 {
		t.Fatalf("got %d overlaps, want 1", len(overlaps))
	}
	if overlaps[0].RuleA != "rule-a" || overlaps[0].RuleB != "rule-b" {
		t.Errorf("overlap = (%s, %s), want (rule-a, rule-b)", overlaps[0].RuleA, overlaps[0].RuleB)
	}
	if len(overlaps[0].SharedFiles) != 1 || overlaps[0].SharedFiles[0] != "App.java" {
		t.Errorf("SharedFiles = %v, want [App.java]", overlaps[0].SharedFiles)
	}
}

func TestDetectIncidentOverlaps_NilCoverage(t *testing.T) {
	overlaps := DetectIncidentOverlaps(nil, nil)
	if len(overlaps) != 0 {
		t.Errorf("got %d overlaps, want 0", len(overlaps))
	}
}

func TestDetectOverlaps_CombinesBoth(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-a",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse"}},
		},
		{
			RuleID: "rule-b",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse.getStatusLine"}},
		},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-a": {Incidents: 1, Files: []string{"App.java"}},
			"rule-b": {Incidents: 1, Files: []string{"App.java"}},
		},
	}

	overlaps := DetectOverlaps(ruleList, cov)
	if len(overlaps) != 2 {
		t.Fatalf("got %d overlaps, want 2 (1 pattern + 1 incident)", len(overlaps))
	}
}

func TestDetectOverlaps_NilCoverage(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-a",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse"}},
		},
		{
			RuleID: "rule-b",
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse.getStatusLine"}},
		},
	}

	overlaps := DetectOverlaps(ruleList, nil)
	if len(overlaps) != 1 {
		t.Fatalf("got %d overlaps, want 1 (pattern only)", len(overlaps))
	}
	if overlaps[0].Type != "pattern_overlap" {
		t.Errorf("type = %q, want pattern_overlap", overlaps[0].Type)
	}
}
