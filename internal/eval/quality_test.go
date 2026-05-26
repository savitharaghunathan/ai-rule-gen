package eval

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestScoreRule(t *testing.T) {
	tests := []struct {
		name        string
		rule        rules.Rule
		wantScore   int
		wantGuidance bool
		wantMissing []string
	}{
		{
			name: "full quality rule",
			rule: rules.Rule{
				RuleID:  "test-001",
				Message: "Replace `OldClass` with `NewClass`. Use `new.api()` instead of `old.api()`.",
				Links:   []rules.Link{{URL: "https://example.com", Title: "Migration guide"}},
				Effort:  3,
			},
			wantScore:    4,
			wantGuidance: true,
			wantMissing:  nil,
		},
		{
			name: "empty rule",
			rule: rules.Rule{
				RuleID: "test-002",
			},
			wantScore:    0,
			wantGuidance: false,
			wantMissing:  []string{"message", "links", "effort", "before_after_guidance"},
		},
		{
			name: "message only",
			rule: rules.Rule{
				RuleID:  "test-003",
				Message: "This API has been removed.",
			},
			wantScore:    1,
			wantGuidance: false,
			wantMissing:  []string{"links", "effort", "before_after_guidance"},
		},
		{
			name: "message with guidance keywords but no links or effort",
			rule: rules.Rule{
				RuleID:  "test-004",
				Message: "Replace `javax.servlet` with `jakarta.servlet`.",
			},
			wantScore:    2,
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
			wantGuidance: false,
			wantMissing:  []string{"message", "before_after_guidance"},
		},
		{
			name: "renamed to keyword triggers guidance",
			rule: rules.Rule{
				RuleID:  "test-006",
				Message: "The class has been renamed to `NewName`.",
				Links:   []rules.Link{{URL: "https://example.com"}},
				Effort:  1,
			},
			wantScore:    4,
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
			if d.HasGuidance != tt.wantGuidance {
				t.Errorf("HasGuidance = %v, want %v", d.HasGuidance, tt.wantGuidance)
			}
			if d.QualityMax != 4 {
				t.Errorf("QualityMax = %d, want 4", d.QualityMax)
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
			Message: "Replace `old` with `new`. Use `newAPI()` instead of `oldAPI()`.",
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
	if summary.MaxScore != 4 {
		t.Errorf("MaxScore = %d, want 4", summary.MaxScore)
	}
	wantAvg := float64(4+3+0) / 3.0
	if summary.AvgScore != wantAvg {
		t.Errorf("AvgScore = %f, want %f", summary.AvgScore, wantAvg)
	}
	if len(details) != 3 {
		t.Fatalf("details length = %d, want 3", len(details))
	}
	if details[0].QualityScore != 4 {
		t.Errorf("details[0].QualityScore = %d, want 4", details[0].QualityScore)
	}
	if details[1].QualityScore != 3 {
		t.Errorf("details[1].QualityScore = %d, want 3", details[1].QualityScore)
	}
	if details[2].QualityScore != 0 {
		t.Errorf("details[2].QualityScore = %d, want 0", details[2].QualityScore)
	}

	t.Run("empty input", func(t *testing.T) {
		s, d := ScoreAll(nil)
		if s.TotalRules != 0 || s.AvgScore != 0 || len(d) != 0 {
			t.Errorf("ScoreAll(nil) = (%+v, %v), want zero values", s, d)
		}
	})
}
