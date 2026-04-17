package kantraparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSummary(t *testing.T) {
	tests := []struct {
		name   string
		output string
		passed int
		total  int
	}{
		{
			name:   "standard output",
			output: "Rules Summary: 5/10 PASSED",
			passed: 5,
			total:  10,
		},
		{
			name:   "all passed",
			output: "some output\nRules Summary: 10/10 PASSED\nmore output",
			passed: 10,
			total:  10,
		},
		{
			name:   "no summary",
			output: "no summary here",
			passed: 0,
			total:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, total := ParseSummary(tt.output)
			if passed != tt.passed || total != tt.total {
				t.Errorf("ParseSummary() = (%d, %d), want (%d, %d)", passed, total, tt.passed, tt.total)
			}
		})
	}
}

func TestParseFailures(t *testing.T) {
	output := "java-ee-to-quarkus-00010    1/1  PASSED\njava-ee-to-quarkus-00020    0/1  PASSED\njava-ee-to-quarkus-00030    0/1  PASSED  find debug data in /tmp/debug/00030"

	failures := ParseFailures(output)
	if len(failures) != 2 {
		t.Fatalf("expected 2 failures, got %d", len(failures))
	}
	if failures[0].RuleID != "java-ee-to-quarkus-00020" {
		t.Errorf("first failure rule ID = %q, want %q", failures[0].RuleID, "java-ee-to-quarkus-00020")
	}
	if failures[1].DebugPath != "/tmp/debug/00030" {
		t.Errorf("second failure debug path = %q, want %q", failures[1].DebugPath, "/tmp/debug/00030")
	}
}

func TestPassedAndFailed_withErroredRules(t *testing.T) {
	// Simulate output where one rule fails per-rule and a group errors out entirely.
	output := "rule-00010    1/1  PASSED\nrule-00020    0/1  PASSED"
	allIDs := []string{"rule-00010", "rule-00020", "rule-00030", "rule-00040"}
	// rule-00030 and rule-00040 belong to an errored group.
	erroredIDs := []string{"rule-00030", "rule-00040"}

	passed, failed := PassedAndFailed(output, allIDs, erroredIDs...)
	if len(passed) != 1 || passed[0] != "rule-00010" {
		t.Errorf("passed = %v, want [rule-00010]", passed)
	}
	if len(failed) != 3 {
		t.Errorf("failed = %v, want 3 failures (rule-00020, rule-00030, rule-00040)", failed)
	}
}

func TestTestFileRuleIDs(t *testing.T) {
	dir := t.TempDir()
	content := `rulesPath: ../../rules/core.yaml
providers:
    - name: java
      dataPath: ./data/core
tests:
    - ruleID: rule-00010
      testCases:
        - name: tc-1
          hasIncidents:
            atLeast: 1
    - ruleID: rule-00020
      testCases:
        - name: tc-1
          hasIncidents:
            atLeast: 1
`
	path := filepath.Join(dir, "core.test.yaml")
	os.WriteFile(path, []byte(content), 0o644)

	ids, err := TestFileRuleIDs(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 rule IDs, got %d: %v", len(ids), ids)
	}
	if ids[0] != "rule-00010" || ids[1] != "rule-00020" {
		t.Errorf("rule IDs = %v, want [rule-00010, rule-00020]", ids)
	}
}

func TestPassedAndFailed_partialFailureNotOverCounted(t *testing.T) {
	// When kantra has partial failures (some rules pass, some fail), only the
	// actually-failing rules should be in the failed list. Previously, the runner
	// would add ALL rules from a group to erroredRuleIDs when kantra exited
	// with error, even if most rules passed. This test verifies the parser
	// correctly identifies only the actual failures.
	output := `data-1.test.yaml                       7/8 PASSED
  rule-00010                             1/1 PASSED
  rule-00020                             1/1 PASSED
  rule-00030                             0/1 PASSED
    tc-1                                 FAILED
    - expected rule to match but unmatched
  rule-00040                             1/1 PASSED
------------------------------------------------------------
  Rules Summary:      3/4 (75.00%) FAILED
------------------------------------------------------------`

	allIDs := []string{"rule-00010", "rule-00020", "rule-00030", "rule-00040"}

	// No erroredRuleIDs — the runner should NOT pass them when kantra
	// produced a Rules Summary (meaning it actually ran).
	passed, failed := PassedAndFailed(output, allIDs)

	if len(failed) != 1 || failed[0] != "rule-00030" {
		t.Errorf("failed = %v, want [rule-00030]", failed)
	}
	if len(passed) != 3 {
		t.Errorf("passed = %v, want 3 rules", passed)
	}
}

func TestPassedAndFailed_totalFailureUsesErroredIDs(t *testing.T) {
	// When kantra fails completely (no Rules Summary), ALL rules from
	// the group should be marked as failed via erroredRuleIDs.
	output := `time="..." level=error msg="unable to get build tool"`

	allIDs := []string{"rule-00010", "rule-00020"}
	erroredIDs := []string{"rule-00010", "rule-00020"}

	passed, failed := PassedAndFailed(output, allIDs, erroredIDs...)

	if len(passed) != 0 {
		t.Errorf("passed = %v, want empty", passed)
	}
	if len(failed) != 2 {
		t.Errorf("failed = %v, want 2 failures", failed)
	}
}

func TestParseSummary_nonZeroTotal(t *testing.T) {
	// Verify ParseSummary returns non-zero total when kantra ran.
	output := `Rules Summary:      7/8 (87.50%) FAILED`
	passed, total := ParseSummary(output)
	if passed != 7 || total != 8 {
		t.Errorf("ParseSummary() = (%d, %d), want (7, 8)", passed, total)
	}
}

func TestFindTestFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"rules.test.yaml", "other.test.yml", "not-a-test.yaml", "readme.md"} {
		os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644)
	}
	files, err := FindTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 test files, got %d: %v", len(files), files)
	}
}
