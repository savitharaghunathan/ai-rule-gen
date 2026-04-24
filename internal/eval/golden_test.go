// internal/eval/golden_test.go
package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGoldenSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "golden.yaml")
	content := []byte(`source: spring-boot-3
target: spring-boot-4
language: java
thresholds:
  pass_rate_post_fix: 95.0
  pre_fix_pass_rate: 80.0
  coverage_min: 70.0
patterns:
  - id: undertow-removal
    dependency_name: org.springframework.boot.spring-boot-starter-undertow
    condition_type: java.dependency
  - id: mockito-listener-removal
    source_fqn: org.springframework.boot.test.mock.mockito.MockitoTestExecutionListener
    condition_type: java.referenced
    location_type: IMPORT
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	gs, err := LoadGoldenSet(path)
	if err != nil {
		t.Fatalf("LoadGoldenSet: %v", err)
	}
	if gs.Source != "spring-boot-3" {
		t.Errorf("Source = %q, want spring-boot-3", gs.Source)
	}
	if gs.Target != "spring-boot-4" {
		t.Errorf("Target = %q, want spring-boot-4", gs.Target)
	}
	if gs.Thresholds.PassRatePostFix != 95.0 {
		t.Errorf("PassRatePostFix = %f, want 95.0", gs.Thresholds.PassRatePostFix)
	}
	if gs.Thresholds.PreFixPassRate != 80.0 {
		t.Errorf("PreFixPassRate = %f, want 80.0", gs.Thresholds.PreFixPassRate)
	}
	if gs.Thresholds.CoverageMin != 70.0 {
		t.Errorf("CoverageMin = %f, want 70.0", gs.Thresholds.CoverageMin)
	}
	if len(gs.Patterns) != 2 {
		t.Fatalf("len(Patterns) = %d, want 2", len(gs.Patterns))
	}
	if gs.Patterns[0].DependencyName != "org.springframework.boot.spring-boot-starter-undertow" {
		t.Errorf("Patterns[0].DependencyName = %q", gs.Patterns[0].DependencyName)
	}
	if gs.Patterns[1].ConditionType != "java.referenced" {
		t.Errorf("Patterns[1].ConditionType = %q", gs.Patterns[1].ConditionType)
	}
}

func TestLoadGoldenSet_FileNotFound(t *testing.T) {
	_, err := LoadGoldenSet("/nonexistent/golden.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
