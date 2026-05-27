package eval

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CIThresholds defines per-migration regression thresholds.
type CIThresholds struct {
	EffectiveCoverageDropPct int     `yaml:"effective_coverage_drop_pct" json:"effective_coverage_drop_pct"`
	QualityAvgDrop           float64 `yaml:"quality_avg_drop" json:"quality_avg_drop"`
}

// EvalConfig represents an eval_config.yaml file.
type EvalConfig struct {
	GuideURL     string        `yaml:"guide_url"`
	AppRepo      string        `yaml:"app_repo"`
	AppCommit    string        `yaml:"app_commit"`
	Source       string        `yaml:"source"`
	Target       string        `yaml:"target"`
	RulesDir     string        `yaml:"rules_dir"`
	CIThresholds *CIThresholds `yaml:"ci_thresholds"`
}

// DefaultThresholds returns the global default CI thresholds.
func DefaultThresholds() CIThresholds {
	return CIThresholds{
		EffectiveCoverageDropPct: 5,
		QualityAvgDrop:           0.0,
	}
}

// ResolvedThresholds returns custom thresholds if set, otherwise defaults.
func (ec *EvalConfig) ResolvedThresholds() CIThresholds {
	if ec.CIThresholds != nil {
		return *ec.CIThresholds
	}
	return DefaultThresholds()
}

// LoadEvalConfig reads an eval_config.yaml file.
func LoadEvalConfig(path string) (*EvalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading eval config %s: %w", path, err)
	}

	var cfg EvalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing eval config %s: %w", path, err)
	}

	return &cfg, nil
}
