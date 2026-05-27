package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEvalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eval_config.yaml")

	content := `guide_url: https://example.com/guide
app_repo: https://github.com/example/app
app_commit: abc1234
source: old-lib
target: new-lib
ci_thresholds:
  effective_coverage_drop_pct: 10
  quality_avg_drop: 0.5
`
	os.WriteFile(path, []byte(content), 0o644)

	cfg, err := LoadEvalConfig(path)
	if err != nil {
		t.Fatalf("LoadEvalConfig: %v", err)
	}

	if cfg.GuideURL != "https://example.com/guide" {
		t.Errorf("GuideURL = %q", cfg.GuideURL)
	}
	if cfg.AppCommit != "abc1234" {
		t.Errorf("AppCommit = %q", cfg.AppCommit)
	}
	if cfg.Source != "old-lib" {
		t.Errorf("Source = %q", cfg.Source)
	}
	if cfg.CIThresholds == nil {
		t.Fatal("CIThresholds is nil")
	}
	if cfg.CIThresholds.EffectiveCoverageDropPct != 10 {
		t.Errorf("EffectiveCoverageDropPct = %d, want 10", cfg.CIThresholds.EffectiveCoverageDropPct)
	}
	if cfg.CIThresholds.QualityAvgDrop != 0.5 {
		t.Errorf("QualityAvgDrop = %f, want 0.5", cfg.CIThresholds.QualityAvgDrop)
	}
}

func TestLoadEvalConfig_NoThresholds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eval_config.yaml")

	content := `guide_url: https://example.com/guide
source: old-lib
target: new-lib
`
	os.WriteFile(path, []byte(content), 0o644)

	cfg, err := LoadEvalConfig(path)
	if err != nil {
		t.Fatalf("LoadEvalConfig: %v", err)
	}

	if cfg.CIThresholds != nil {
		t.Error("CIThresholds should be nil when not specified")
	}

	resolved := cfg.ResolvedThresholds()
	defaults := DefaultThresholds()
	if resolved.EffectiveCoverageDropPct != defaults.EffectiveCoverageDropPct {
		t.Errorf("resolved coverage threshold = %d, want default %d",
			resolved.EffectiveCoverageDropPct, defaults.EffectiveCoverageDropPct)
	}
}

func TestLoadEvalConfig_NotFound(t *testing.T) {
	_, err := LoadEvalConfig("/nonexistent/eval_config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestResolvedThresholds_Custom(t *testing.T) {
	cfg := &EvalConfig{
		CIThresholds: &CIThresholds{
			EffectiveCoverageDropPct: 3,
			QualityAvgDrop:           0.2,
		},
	}

	resolved := cfg.ResolvedThresholds()
	if resolved.EffectiveCoverageDropPct != 3 {
		t.Errorf("EffectiveCoverageDropPct = %d, want 3", resolved.EffectiveCoverageDropPct)
	}
	if resolved.QualityAvgDrop != 0.2 {
		t.Errorf("QualityAvgDrop = %f, want 0.2", resolved.QualityAvgDrop)
	}
}
