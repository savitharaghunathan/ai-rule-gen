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
