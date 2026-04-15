package testgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
)

func TestParseSummary(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantPass  int
		wantTotal int
	}{
		{"all passed", "Rules Summary: 5/5 PASSED", 5, 5},
		{"some failed", "output\nRules Summary: 3/5 PASSED\nmore", 3, 5},
		{"none passed", "Rules Summary: 0/10 PASSED", 0, 10},
		{"no summary", "random output", 0, 0},
		{"empty", "", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, total := kantraparser.ParseSummary(tt.output)
			if passed != tt.wantPass {
				t.Errorf("passed = %d, want %d", passed, tt.wantPass)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestParseFailures(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantIDs  []string
	}{
		{
			name:    "single failure with debug path",
			output:  "spring-boot-00010  0/1  PASSED  find debug data in /tmp/debug-123",
			wantIDs: []string{"spring-boot-00010"},
		},
		{
			name: "multiple failures",
			output: `spring-boot-00010  1/1  PASSED
spring-boot-00020  0/1  PASSED  find debug data in /tmp/a
spring-boot-00030  0/1  PASSED  find debug data in /tmp/b`,
			wantIDs: []string{"spring-boot-00020", "spring-boot-00030"},
		},
		{
			name:    "no failures",
			output:  "spring-boot-00010  1/1  PASSED",
			wantIDs: nil,
		},
		{
			name:    "empty",
			output:  "",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failures := kantraparser.ParseFailures(tt.output)
			if len(failures) != len(tt.wantIDs) {
				t.Fatalf("got %d failures, want %d", len(failures), len(tt.wantIDs))
			}
			for i, id := range tt.wantIDs {
				if failures[i].RuleID != id {
					t.Errorf("failure[%d].RuleID = %q, want %q", i, failures[i].RuleID, id)
				}
			}
		})
	}
}

func TestParseFailures_DebugPath(t *testing.T) {
	output := "golang-fips-00010  0/1  PASSED  find debug data in /tmp/kantra-debug-abc123"
	failures := kantraparser.ParseFailures(output)
	if len(failures) != 1 {
		t.Fatalf("got %d failures, want 1", len(failures))
	}
	if failures[0].DebugPath != "/tmp/kantra-debug-abc123" {
		t.Errorf("DebugPath = %q, want %q", failures[0].DebugPath, "/tmp/kantra-debug-abc123")
	}
}

func TestFindTestFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "web.test.yaml"), []byte("test: true"), 0o644)
	os.WriteFile(filepath.Join(dir, "sec.test.yml"), []byte("test: true"), 0o644)
	os.WriteFile(filepath.Join(dir, "not-test.yaml"), []byte("nope"), 0o644)
	os.MkdirAll(filepath.Join(dir, "data"), 0o755)

	files, err := kantraparser.FindTestFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2: %v", len(files), files)
	}
}

func TestExtractPatternFromDebug(t *testing.T) {
	rulesYAML := `- ruleID: spring-boot-00010
  when:
    java.referenced:
      pattern: javax.ejb.Stateless
      location: ANNOTATION
- ruleID: fips-00020
  when:
    go.referenced:
      pattern: golang.org/x/crypto/md4
`
	t.Run("java pattern", func(t *testing.T) {
		pattern, provider := extractPatternFromDebug([]byte(rulesYAML), "spring-boot-00010")
		if pattern != "javax.ejb.Stateless" {
			t.Errorf("pattern = %q, want %q", pattern, "javax.ejb.Stateless")
		}
		if provider != "java.referenced" {
			t.Errorf("provider = %q, want %q", provider, "java.referenced")
		}
	})

	t.Run("go pattern", func(t *testing.T) {
		pattern, provider := extractPatternFromDebug([]byte(rulesYAML), "fips-00020")
		if pattern != "golang.org/x/crypto/md4" {
			t.Errorf("pattern = %q, want %q", pattern, "golang.org/x/crypto/md4")
		}
		if provider != "go.referenced" {
			t.Errorf("provider = %q, want %q", provider, "go.referenced")
		}
	})

	t.Run("unknown rule", func(t *testing.T) {
		pattern, provider := extractPatternFromDebug([]byte(rulesYAML), "nonexistent-00099")
		if pattern != "" || provider != "" {
			t.Errorf("expected empty, got pattern=%q provider=%q", pattern, provider)
		}
	})
}
