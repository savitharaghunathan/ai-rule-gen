// internal/eval/golden.go
package eval

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type GoldenSet struct {
	Source     string          `yaml:"source"`
	Target     string          `yaml:"target"`
	Language   string          `yaml:"language"`
	Patterns   []GoldenPattern `yaml:"patterns"`
	Thresholds Thresholds      `yaml:"thresholds"`
}

type GoldenPattern struct {
	ID             string `yaml:"id"`
	SourceFQN      string `yaml:"source_fqn,omitempty"`
	DependencyName string `yaml:"dependency_name,omitempty"`
	XPath          string `yaml:"xpath,omitempty"`
	ConditionType  string `yaml:"condition_type"`
	LocationType   string `yaml:"location_type,omitempty"`
}

type Thresholds struct {
	PassRatePostFix float64 `yaml:"pass_rate_post_fix"`
	PreFixPassRate  float64 `yaml:"pre_fix_pass_rate"`
	CoverageMin     float64 `yaml:"coverage_min"`
}

func LoadGoldenSet(path string) (*GoldenSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading golden set %s: %w", path, err)
	}
	var gs GoldenSet
	if err := yaml.Unmarshal(data, &gs); err != nil {
		return nil, fmt.Errorf("parsing golden set %s: %w", path, err)
	}
	return &gs, nil
}
