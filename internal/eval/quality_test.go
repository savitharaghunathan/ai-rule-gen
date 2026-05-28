package eval

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestScoreGuidanceDepth(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		lastSeg   string
		wantDepth int
	}{
		{"empty message", "", "getStatusLine", 0},
		{"whitespace only", "   \t\n  ", "getStatusLine", 0},
		{"plain text no actionable", "This API has been removed.", "getStatusLine", 1},
		{"actionable keyword replace", "Replace old usage with new.", "getStatusLine", 2},
		{"actionable keyword instead of", "Use the new API instead of the old one.", "foo", 2},
		{"backtick same as condition", "Use `getStatusLine` from the new client.", "getStatusLine", 2},
		{"backtick different from condition", "Use `getCode()` instead.", "getStatusLine", 3},
		{"backtick with FQN different", "Use `org.apache.hc.core5.http.ClassicHttpResponse` instead.", "getStatusLine", 3},
		{"backtick not an identifier", "Use `some random text with spaces` instead.", "getStatusLine", 2},
		{"no condition pattern allows tier 3", "Use `NewClass` for migration.", "", 3},
		{"renamed to keyword", "The class has been renamed to `NewName`.", "OldName", 3},
		{"bare before without after", "This was the default behavior before version 3.", "foo", 1},
		{"before and after together", "Change the behavior before migration and after.", "foo", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreGuidanceDepth(tt.message, tt.lastSeg)
			if got != tt.wantDepth {
				t.Errorf("scoreGuidanceDepth(%q, %q) = %d, want %d", tt.message, tt.lastSeg, got, tt.wantDepth)
			}
		})
	}
}

func TestExtractConditionPattern(t *testing.T) {
	tests := []struct {
		name string
		rule rules.Rule
		want string
	}{
		{
			name: "java.referenced",
			rule: rules.Rule{When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse"}}},
			want: "org.apache.http.HttpResponse",
		},
		{
			name: "go.referenced",
			rule: rules.Rule{When: rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "github.com/old/pkg.Function"}}},
			want: "github.com/old/pkg.Function",
		},
		{
			name: "nodejs.referenced",
			rule: rules.Rule{When: rules.Condition{NodejsReferenced: &rules.NodejsReferenced{Pattern: "express.Router"}}},
			want: "express.Router",
		},
		{
			name: "empty condition",
			rule: rules.Rule{},
			want: "",
		},
		{
			name: "java.dependency",
			rule: rules.Rule{When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.springframework.boot"}}},
			want: "org.springframework.boot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractConditionPattern(tt.rule)
			if got != tt.want {
				t.Errorf("extractConditionPattern() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConditionLastSegment(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"org.apache.http.HttpResponse.getStatusLine", "getStatusLine"},
		{"org.apache.http.HttpResponse", "HttpResponse"},
		{"org.apache.http*", "http*"},
		{"org.apache.http.*", ""},
		{"", ""},
		{"  ", ""},
		{"SingleName", "SingleName"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := conditionLastSegment(tt.pattern)
			if got != tt.want {
				t.Errorf("conditionLastSegment(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"getStatusLine", true},
		{"ClassicHttpResponse", true},
		{"org.apache.http.ClassicHttpResponse", true},
		{"new_method", true},
		{"some random text", false},
		{"", false},
		{"123abc", false},
		{"abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := isValidIdentifier(tt.s)
			if got != tt.want {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestScoreRule(t *testing.T) {
	tests := []struct {
		name        string
		rule        rules.Rule
		wantScore   int
		wantDepth   int
		wantGuidance bool
		wantMissing []string
	}{
		{
			name: "full quality rule with specific replacement",
			rule: rules.Rule{
				RuleID: "test-001",
				When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse.getStatusLine"}},
				Message: "Replace `getStatusLine()` with `getCode()`. Use `org.apache.hc.core5.http.ClassicHttpResponse` instead.",
				Links:   []rules.Link{{URL: "https://example.com", Title: "Migration guide"}},
				Effort:  3,
			},
			wantScore:    6,
			wantDepth:    3,
			wantGuidance: true,
			wantMissing:  nil,
		},
		{
			name: "empty rule",
			rule: rules.Rule{
				RuleID: "test-002",
			},
			wantScore:    0,
			wantDepth:    0,
			wantGuidance: false,
			wantMissing:  []string{"message", "links", "effort", "before_after_guidance"},
		},
		{
			name: "message only plain text",
			rule: rules.Rule{
				RuleID:  "test-003",
				Message: "This API has been removed.",
			},
			wantScore:    2,
			wantDepth:    1,
			wantGuidance: false,
			wantMissing:  []string{"links", "effort", "before_after_guidance"},
		},
		{
			name: "message with specific replacement but no links or effort",
			rule: rules.Rule{
				RuleID:  "test-004",
				When:    rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.servlet"}},
				Message: "Replace `javax.servlet` with `jakarta.servlet`.",
			},
			wantScore:    4,
			wantDepth:    3,
			wantGuidance: true,
			wantMissing:  []string{"links", "effort"},
		},
		{
			name: "whitespace-only message treated as missing",
			rule: rules.Rule{
				RuleID:  "test-005",
				Message: "   \t\n  ",
				Links:   []rules.Link{{URL: "https://example.com"}},
				Effort:  1,
			},
			wantScore:    2,
			wantDepth:    0,
			wantGuidance: false,
			wantMissing:  []string{"message", "before_after_guidance"},
		},
		{
			name: "renamed to keyword triggers guidance",
			rule: rules.Rule{
				RuleID:  "test-006",
				When:    rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.old.OldName"}},
				Message: "The class has been renamed to `NewName`.",
				Links:   []rules.Link{{URL: "https://example.com"}},
				Effort:  1,
			},
			wantScore:    6,
			wantDepth:    3,
			wantGuidance: true,
			wantMissing:  nil,
		},
		{
			name: "bare before without after does not trigger actionable",
			rule: rules.Rule{
				RuleID:  "test-007",
				Message: "This was the default behavior before version 3.",
				Links:   []rules.Link{{URL: "https://example.com"}},
				Effort:  1,
			},
			wantScore:    4,
			wantDepth:    1,
			wantGuidance: false,
			wantMissing:  []string{"before_after_guidance"},
		},
		{
			name: "backtick code reference same as condition",
			rule: rules.Rule{
				RuleID:  "test-008",
				When:    rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.hc.core5.http.ClassicHttpResponse"}},
				Message: "Use `ClassicHttpResponse` instead.",
				Links:   []rules.Link{{URL: "https://example.com"}},
				Effort:  1,
			},
			wantScore:    5,
			wantDepth:    2,
			wantGuidance: true,
			wantMissing:  nil,
		},
		{
			name: "backtick code reference different from condition",
			rule: rules.Rule{
				RuleID:  "test-009",
				When:    rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.apache.http.HttpResponse.getStatusLine"}},
				Message: "Use `org.apache.hc.core5.http.ClassicHttpResponse` instead.",
				Links:   []rules.Link{{URL: "https://example.com"}},
				Effort:  1,
			},
			wantScore:    6,
			wantDepth:    3,
			wantGuidance: true,
			wantMissing:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := ScoreRule(tt.rule)
			if d.QualityScore != tt.wantScore {
				t.Errorf("QualityScore = %d, want %d", d.QualityScore, tt.wantScore)
			}
			if d.GuidanceDepth != tt.wantDepth {
				t.Errorf("GuidanceDepth = %d, want %d", d.GuidanceDepth, tt.wantDepth)
			}
			if d.HasGuidance != tt.wantGuidance {
				t.Errorf("HasGuidance = %v, want %v", d.HasGuidance, tt.wantGuidance)
			}
			if d.QualityMax != 6 {
				t.Errorf("QualityMax = %d, want 6", d.QualityMax)
			}
			if len(d.Missing) != len(tt.wantMissing) {
				t.Errorf("Missing = %v, want %v", d.Missing, tt.wantMissing)
			} else {
				for i, m := range d.Missing {
					if m != tt.wantMissing[i] {
						t.Errorf("Missing[%d] = %q, want %q", i, m, tt.wantMissing[i])
					}
				}
			}
		})
	}
}

func TestScoreAll(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID:  "full-001",
			When:    rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.old.OldClass.oldMethod"}},
			Message: "Replace `oldMethod()` with `newAPI()`. Use `NewClass` instead of `OldClass`.",
			Links:   []rules.Link{{URL: "https://example.com"}},
			Effort:  3,
		},
		{
			RuleID:  "partial-001",
			Message: "This API changed.",
			Links:   []rules.Link{{URL: "https://example.com"}},
			Effort:  1,
		},
		{
			RuleID: "empty-001",
		},
	}

	summary, details := ScoreAll(ruleList)

	if summary.TotalRules != 3 {
		t.Errorf("TotalRules = %d, want 3", summary.TotalRules)
	}
	if summary.HasMessage != 2 {
		t.Errorf("HasMessage = %d, want 2", summary.HasMessage)
	}
	if summary.HasLinks != 2 {
		t.Errorf("HasLinks = %d, want 2", summary.HasLinks)
	}
	if summary.HasEffort != 2 {
		t.Errorf("HasEffort = %d, want 2", summary.HasEffort)
	}
	if summary.HasBeforeAfter != 1 {
		t.Errorf("HasBeforeAfter = %d, want 1", summary.HasBeforeAfter)
	}
	if summary.MaxScore != 6 {
		t.Errorf("MaxScore = %d, want 6", summary.MaxScore)
	}

	if len(details) != 3 {
		t.Fatalf("details length = %d, want 3", len(details))
	}
	// full-001: message(1) + links(1) + effort(1) + depth 3 = 6
	if details[0].QualityScore != 6 {
		t.Errorf("details[0].QualityScore = %d, want 6", details[0].QualityScore)
	}
	if details[0].GuidanceDepth != 3 {
		t.Errorf("details[0].GuidanceDepth = %d, want 3", details[0].GuidanceDepth)
	}
	// partial-001: message(1) + links(1) + effort(1) + depth 1 = 4
	if details[1].QualityScore != 4 {
		t.Errorf("details[1].QualityScore = %d, want 4", details[1].QualityScore)
	}
	if details[1].GuidanceDepth != 1 {
		t.Errorf("details[1].GuidanceDepth = %d, want 1", details[1].GuidanceDepth)
	}
	// empty-001: 0
	if details[2].QualityScore != 0 {
		t.Errorf("details[2].QualityScore = %d, want 0", details[2].QualityScore)
	}

	// AvgScore: (6+4+0)/3 = 3.333...
	wantAvg := float64(6+4+0) / 3.0
	if summary.AvgScore != wantAvg {
		t.Errorf("AvgScore = %f, want %f", summary.AvgScore, wantAvg)
	}

	// GuidanceDepthAvg: (3+1+0)/3 = 1.333...
	wantDepthAvg := float64(3+1+0) / 3.0
	if summary.GuidanceDepthAvg != wantDepthAvg {
		t.Errorf("GuidanceDepthAvg = %f, want %f", summary.GuidanceDepthAvg, wantDepthAvg)
	}

	t.Run("empty input", func(t *testing.T) {
		s, d := ScoreAll(nil)
		if s.TotalRules != 0 || s.AvgScore != 0 || s.GuidanceDepthAvg != 0 || len(d) != 0 {
			t.Errorf("ScoreAll(nil) = (%+v, %v), want zero values", s, d)
		}
	})
}
