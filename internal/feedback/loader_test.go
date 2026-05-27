package feedback

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/verify"
)

func TestDiscoverRuns(t *testing.T) {
	base := t.TempDir()

	os.MkdirAll(filepath.Join(base, "httpclient-4-to-5-20260511-120000"), 0o755)
	os.MkdirAll(filepath.Join(base, "httpclient-4-to-5-20260512-130000"), 0o755)
	os.MkdirAll(filepath.Join(base, "spring-boot-3-to-4-20260511-100000"), 0o755)
	os.MkdirAll(filepath.Join(base, "not-a-run"), 0o755)
	os.WriteFile(filepath.Join(base, "file.txt"), []byte("x"), 0o644)

	t.Run("all", func(t *testing.T) {
		dirs, err := DiscoverRuns(base, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(dirs) != 3 {
			t.Errorf("got %d dirs, want 3", len(dirs))
		}
	})

	t.Run("filtered", func(t *testing.T) {
		dirs, err := DiscoverRuns(base, "httpclient")
		if err != nil {
			t.Fatal(err)
		}
		if len(dirs) != 2 {
			t.Errorf("got %d dirs, want 2", len(dirs))
		}
	})
}

func TestLoadRun(t *testing.T) {
	dir := t.TempDir()

	reportYAML := `generated_at: "2026-05-12T19:00:00Z"
sources:
  - httpclient-4
targets:
  - httpclient-5
rules_total: 3
tests_passed: 2
tests_failed: 1
kantra_limitation: 0
pass_rate: 66.67
rules:
  - rule_id: httpclient-4-to-httpclient-5-00010
    test_status: passed
    source_verified: "true"
  - rule_id: httpclient-4-to-httpclient-5-00020
    test_status: passed
    source_verified: "false"
  - rule_id: httpclient-4-to-httpclient-5-00030
    test_status: failed
    source_verified: "true"
`
	os.WriteFile(filepath.Join(dir, "report.yaml"), []byte(reportYAML), 0o644)

	patternsJSON := `{
  "sources": ["httpclient-4"],
  "targets": ["httpclient-5"],
  "language": "java",
  "patterns": [
    {"source_fqn": "org.apache.http.HttpClient", "location_type": "PACKAGE", "category": "mandatory", "complexity": "low"},
    {"source_fqn": "org.apache.http.HttpResponse.getStatusLine", "location_type": "METHOD_CALL", "category": "mandatory", "complexity": "medium"},
    {"source_fqn": "org.apache.http.conn.ssl.SSLConnectionSocketFactory", "location_type": "TYPE", "category": "optional", "complexity": "high", "source_artifact": {"group_id": "org.apache.httpcomponents", "artifact_id": "httpclient", "version": "4.5.14"}}
  ]
}`
	os.WriteFile(filepath.Join(dir, "patterns.json"), []byte(patternsJSON), 0o644)

	verifyResults := map[string]any{
		"results": []verify.Result{
			{PatternIndex: 0, SourceFQN: "org.apache.http.HttpClient", Status: verify.StatusVerified},
			{PatternIndex: 1, SourceFQN: "org.apache.http.HttpResponse.getStatusLine", Status: verify.StatusNotFound, Reason: "not found in jar"},
			{PatternIndex: 2, SourceFQN: "org.apache.http.conn.ssl.SSLConnectionSocketFactory", Status: verify.StatusVerified},
		},
	}
	data, _ := json.Marshal(verifyResults)
	os.WriteFile(filepath.Join(dir, "verify-results.json"), data, 0o644)

	run, err := LoadRun(dir)
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}

	if run.RulesTotal != 3 {
		t.Errorf("RulesTotal: got %d, want 3", run.RulesTotal)
	}
	if run.Language != "java" {
		t.Errorf("Language: got %q, want java", run.Language)
	}
	if len(run.Patterns) != 3 {
		t.Fatalf("Patterns: got %d, want 3", len(run.Patterns))
	}

	p0 := run.Patterns[0]
	if p0.VerifyStatus != "verified" {
		t.Errorf("pattern 0 verify: got %q, want verified", p0.VerifyStatus)
	}
	if p0.TestStatus != "passed" {
		t.Errorf("pattern 0 test: got %q, want passed", p0.TestStatus)
	}

	p1 := run.Patterns[1]
	if p1.VerifyStatus != "not_found" {
		t.Errorf("pattern 1 verify: got %q, want not_found", p1.VerifyStatus)
	}
	if p1.TestStatus != "passed" {
		t.Errorf("pattern 1 test: got %q, want passed", p1.TestStatus)
	}

	p2 := run.Patterns[2]
	if !p2.HasArtifact {
		t.Error("pattern 2 should have artifact")
	}
	if p2.TestStatus != "failed" {
		t.Errorf("pattern 2 test: got %q, want failed", p2.TestStatus)
	}
}

func TestLoadRunOldFormat(t *testing.T) {
	dir := t.TempDir()

	reportYAML := `generated_at: "2026-05-11T10:00:00Z"
source: httpclient-4
target: httpclient-5
rules_total: 1
tests_passed: 1
tests_failed: 0
pass_rate: 100
rules:
  - rule_id: httpclient-4-to-httpclient-5-00010
    test_status: passed
`
	os.WriteFile(filepath.Join(dir, "report.yaml"), []byte(reportYAML), 0o644)

	run, err := LoadRun(dir)
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if len(run.Sources) != 1 || run.Sources[0] != "httpclient-4" {
		t.Errorf("Sources: got %v, want [httpclient-4]", run.Sources)
	}
	if len(run.Targets) != 1 || run.Targets[0] != "httpclient-5" {
		t.Errorf("Targets: got %v, want [httpclient-5]", run.Targets)
	}
}

func TestLoadRunNoReport(t *testing.T) {
	dir := t.TempDir()
	run, err := LoadRun(dir)
	if err != nil {
		t.Fatal(err)
	}
	if run != nil {
		t.Error("expected nil for directory without report.yaml")
	}
}

func TestLoadRunReportOnlyVerification(t *testing.T) {
	dir := t.TempDir()

	reportYAML := `generated_at: "2026-05-12T19:00:00Z"
sources:
  - httpclient-4
targets:
  - httpclient-5
rules_total: 2
tests_passed: 2
tests_failed: 0
pass_rate: 100
rules:
  - rule_id: httpclient-4-to-httpclient-5-00010
    test_status: passed
    source_verified: "true"
  - rule_id: httpclient-4-to-httpclient-5-00020
    test_status: passed
    source_verified: "false"
`
	os.WriteFile(filepath.Join(dir, "report.yaml"), []byte(reportYAML), 0o644)

	patternsJSON := `{
  "sources": ["httpclient-4"],
  "targets": ["httpclient-5"],
  "language": "java",
  "patterns": [
    {"source_fqn": "org.apache.http.HttpClient", "location_type": "PACKAGE", "category": "mandatory", "complexity": "low"},
    {"source_fqn": "org.apache.http.HttpResponse.getStatusLine", "location_type": "METHOD_CALL", "category": "mandatory", "complexity": "medium"}
  ]
}`
	os.WriteFile(filepath.Join(dir, "patterns.json"), []byte(patternsJSON), 0o644)

	// No verify-results.json — verification comes only from report.yaml

	run, err := LoadRun(dir)
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if len(run.Patterns) != 2 {
		t.Fatalf("Patterns: got %d, want 2", len(run.Patterns))
	}

	p0 := run.Patterns[0]
	if p0.VerifyStatus != "true" {
		t.Errorf("pattern 0 verify: got %q, want %q", p0.VerifyStatus, "true")
	}

	p1 := run.Patterns[1]
	if p1.VerifyStatus != "false" {
		t.Errorf("pattern 1 verify: got %q, want %q", p1.VerifyStatus, "false")
	}
}

func TestParseTimestamp(t *testing.T) {
	ts := parseTimestamp("httpclient-4-to-5-20260512-153000")
	if ts.Year() != 2026 || ts.Month() != 5 || ts.Day() != 12 {
		t.Errorf("unexpected date: %v", ts)
	}
	if ts.Hour() != 15 || ts.Minute() != 30 {
		t.Errorf("unexpected time: %v", ts)
	}
}
