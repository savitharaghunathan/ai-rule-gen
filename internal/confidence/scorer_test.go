package confidence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
)

func TestParseSummary(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantPassed int
		wantTotal  int
	}{
		{
			name:       "all passed",
			output:     "Rules Summary: 5/5 PASSED",
			wantPassed: 5,
			wantTotal:  5,
		},
		{
			name:       "some failed",
			output:     "some output\nRules Summary: 3/5 PASSED\nmore output",
			wantPassed: 3,
			wantTotal:  5,
		},
		{
			name:       "none passed",
			output:     "Rules Summary: 0/10 PASSED",
			wantPassed: 0,
			wantTotal:  10,
		},
		{
			name:       "no summary line",
			output:     "some random output with no summary",
			wantPassed: 0,
			wantTotal:  0,
		},
		{
			name:       "empty output",
			output:     "",
			wantPassed: 0,
			wantTotal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, total := kantraparser.ParseSummary(tt.output)
			if passed != tt.wantPassed {
				t.Errorf("passed = %d, want %d", passed, tt.wantPassed)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestParseFailedRules(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantFail []string
	}{
		{
			name:     "single failure",
			output:   "spring-boot-00010  0/1  PASSED  find debug data in /tmp/debug",
			wantFail: []string{"spring-boot-00010"},
		},
		{
			name: "multiple failures",
			output: `spring-boot-00010  1/1  PASSED
spring-boot-00020  0/1  PASSED  find debug data in /tmp/a
spring-boot-00030  0/1  PASSED  find debug data in /tmp/b`,
			wantFail: []string{"spring-boot-00020", "spring-boot-00030"},
		},
		{
			name:     "no failures",
			output:   "spring-boot-00010  1/1  PASSED",
			wantFail: nil,
		},
		{
			name:     "empty output",
			output:   "",
			wantFail: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failures := kantraparser.ParseFailures(tt.output)
			if tt.wantFail == nil && len(failures) != 0 {
				t.Errorf("expected no failures, got %v", failures)
				return
			}
			failedIDs := make(map[string]bool, len(failures))
			for _, f := range failures {
				failedIDs[f.RuleID] = true
			}
			for _, id := range tt.wantFail {
				if !failedIDs[id] {
					t.Errorf("expected %s in failed rules", id)
				}
			}
			if len(failures) != len(tt.wantFail) {
				t.Errorf("got %d failures, want %d", len(failures), len(tt.wantFail))
			}
		})
	}
}

func TestComputeSummary(t *testing.T) {
	tests := []struct {
		name    string
		scores  []Score
		want    Summary
	}{
		{
			name: "all passed",
			scores: []Score{
				{RuleID: "r1", TestPassed: true, Verdict: "accept"},
				{RuleID: "r2", TestPassed: true, Verdict: "accept"},
			},
			want: Summary{TotalRules: 2, Passed: 2, Failed: 0, PassRate: 100},
		},
		{
			name: "mixed results",
			scores: []Score{
				{RuleID: "r1", TestPassed: true, Verdict: "accept"},
				{RuleID: "r2", TestPassed: false, Verdict: "reject"},
				{RuleID: "r3", TestPassed: true, Verdict: "review"},
			},
			want: Summary{TotalRules: 3, Passed: 2, Failed: 1, PassRate: 200.0 / 3.0},
		},
		{
			name: "all failed",
			scores: []Score{
				{RuleID: "r1", TestPassed: false, Verdict: "reject"},
			},
			want: Summary{TotalRules: 1, Passed: 0, Failed: 1, PassRate: 0},
		},
		{
			name:   "empty",
			scores: []Score{},
			want:   Summary{TotalRules: 0, Passed: 0, Failed: 0, PassRate: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSummary(tt.scores)
			if got.TotalRules != tt.want.TotalRules {
				t.Errorf("TotalRules = %d, want %d", got.TotalRules, tt.want.TotalRules)
			}
			if got.Passed != tt.want.Passed {
				t.Errorf("Passed = %d, want %d", got.Passed, tt.want.Passed)
			}
			if got.Failed != tt.want.Failed {
				t.Errorf("Failed = %d, want %d", got.Failed, tt.want.Failed)
			}
			diff := got.PassRate - tt.want.PassRate
			if diff > 0.01 || diff < -0.01 {
				t.Errorf("PassRate = %f, want %f", got.PassRate, tt.want.PassRate)
			}
		})
	}
}

func TestParseJudgeResponse(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		wantScore   float64
		wantVerdict string
		wantErr     bool
	}{
		{
			name:        "high scores — accept",
			response:    `{"pattern_correctness": 5, "message_quality": 5, "category_appropriateness": 4, "effort_accuracy": 4, "false_positive_risk": 5, "reasoning": "looks good"}`,
			wantScore:   4.6,
			wantVerdict: "accept",
		},
		{
			name:        "low scores — reject",
			response:    `{"pattern_correctness": 1, "message_quality": 2, "category_appropriateness": 2, "effort_accuracy": 1, "false_positive_risk": 1, "reasoning": "bad rule"}`,
			wantScore:   1.4,
			wantVerdict: "reject",
		},
		{
			name:        "mid scores — review",
			response:    `{"pattern_correctness": 3, "message_quality": 3, "category_appropriateness": 3, "effort_accuracy": 3, "false_positive_risk": 3, "reasoning": "mediocre"}`,
			wantScore:   3.0,
			wantVerdict: "review",
		},
		{
			name:        "json embedded in text",
			response:    `Here is my assessment: {"pattern_correctness": 5, "message_quality": 5, "category_appropriateness": 5, "effort_accuracy": 5, "false_positive_risk": 5, "reasoning": "perfect"} end`,
			wantScore:   5.0,
			wantVerdict: "accept",
		},
		{
			name:     "no json",
			response: "this is not json at all",
			wantErr:  true,
		},
		{
			name:     "empty response",
			response: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, verdict, _, err := parseJudgeResponse(tt.response)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			diff := score - tt.wantScore
			if diff > 0.01 || diff < -0.01 {
				t.Errorf("score = %f, want %f", score, tt.wantScore)
			}
			if verdict != tt.wantVerdict {
				t.Errorf("verdict = %q, want %q", verdict, tt.wantVerdict)
			}
		})
	}
}

func TestParseJudgeResponse_MissingFields(t *testing.T) {
	// Only some fields present — missing fields default to 0
	response := `{"pattern_correctness": 5, "reasoning": "partial"}`
	score, verdict, _, err := parseJudgeResponse(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only pattern_correctness=5, rest default to 0: average = 5/5 = 1.0
	if score >= 2.5 {
		t.Errorf("expected low score with missing fields, got %f", score)
	}
	if verdict != "reject" {
		t.Errorf("expected reject verdict, got %q", verdict)
	}
}

func TestParseJudgeResponse_MalformedJSON(t *testing.T) {
	_, _, _, err := parseJudgeResponse(`{"pattern_correctness": 5, broken`)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParseJudgeResponse_ExtremeScores(t *testing.T) {
	// All zeros
	response := `{"pattern_correctness": 0, "message_quality": 0, "category_appropriateness": 0, "effort_accuracy": 0, "false_positive_risk": 0, "reasoning": "all zero"}`
	score, verdict, _, err := parseJudgeResponse(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0 {
		t.Errorf("expected 0 score, got %f", score)
	}
	if verdict != "reject" {
		t.Errorf("expected reject, got %q", verdict)
	}
}

func TestParseJudgeResponse_MultipleJSONObjects(t *testing.T) {
	// Multiple JSON objects — parseJudgeResponse takes first { to last },
	// which captures invalid text between objects. This is expected to fail.
	response := `{"pattern_correctness": 3} and {"pattern_correctness": 5}`
	_, _, _, err := parseJudgeResponse(response)
	if err == nil {
		t.Error("expected error for multiple JSON objects with text between them")
	}
}

func TestFindTestFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "web.test.yaml"), []byte("test: true"), 0o644)
	os.WriteFile(filepath.Join(dir, "security.test.yml"), []byte("test: true"), 0o644)
	os.WriteFile(filepath.Join(dir, "not-a-test.yaml"), []byte("nope"), 0o644)
	os.MkdirAll(filepath.Join(dir, "data"), 0o755)

	files, err := kantraparser.FindTestFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("got %d test files, want 2: %v", len(files), files)
	}
}

func TestCollectRuleIDs(t *testing.T) {
	dir := t.TempDir()

	testYAML := `rulesPath: ../rules/web.yaml
tests:
  - ruleID: spring-boot-00010
    testCases:
      - name: tc-1
  - ruleID: spring-boot-00020
    testCases:
      - name: tc-1
`
	testFile := filepath.Join(dir, "web.test.yaml")
	os.WriteFile(testFile, []byte(testYAML), 0o644)

	ids, err := collectRuleIDs([]string{testFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d IDs, want 2", len(ids))
	}
	if ids[0] != "spring-boot-00010" || ids[1] != "spring-boot-00020" {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

func TestCollectRuleIDs_Deduplicates(t *testing.T) {
	dir := t.TempDir()

	// Same rule ID in two test files
	yaml1 := `tests:
  - ruleID: rule-00010
`
	yaml2 := `tests:
  - ruleID: rule-00010
  - ruleID: rule-00020
`
	f1 := filepath.Join(dir, "a.test.yaml")
	f2 := filepath.Join(dir, "b.test.yaml")
	os.WriteFile(f1, []byte(yaml1), 0o644)
	os.WriteFile(f2, []byte(yaml2), 0o644)

	ids, err := collectRuleIDs([]string{f1, f2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("got %d IDs (expected dedup), want 2: %v", len(ids), ids)
	}
}

func TestNew_Defaults(t *testing.T) {
	s := New("", 0, nil, nil)
	if s.kantraPath != "kantra" {
		t.Errorf("kantraPath = %q, want %q", s.kantraPath, "kantra")
	}
	if s.timeout.Seconds() != 900 {
		t.Errorf("timeout = %v, want 900s", s.timeout)
	}
}

func TestNew_CustomValues(t *testing.T) {
	s := New("/usr/bin/kantra", 60, nil, nil)
	if s.kantraPath != "/usr/bin/kantra" {
		t.Errorf("kantraPath = %q, want %q", s.kantraPath, "/usr/bin/kantra")
	}
	if s.timeout.Seconds() != 60 {
		t.Errorf("timeout = %v, want 60s", s.timeout)
	}
}
