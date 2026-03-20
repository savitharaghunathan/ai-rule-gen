package rules

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReadRulesFile reads rules from a single YAML file.
func ReadRulesFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading rules file %s: %w", path, err)
	}
	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parsing rules file %s: %w", path, err)
	}
	return rules, nil
}

// ReadRulesDir reads all rule YAML files from a directory.
// It skips ruleset.yaml files.
func ReadRulesDir(dir string) ([]Rule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading rules directory %s: %w", dir, err)
	}
	var allRules []Rule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}
		if name == "ruleset.yaml" || name == "ruleset.yml" {
			continue
		}
		rules, err := ReadRulesFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, rules...)
	}
	return allRules, nil
}

// ReadRuleset reads a ruleset.yaml file.
func ReadRuleset(path string) (*Ruleset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading ruleset file %s: %w", path, err)
	}
	var rs Ruleset
	if err := yaml.Unmarshal(data, &rs); err != nil {
		return nil, fmt.Errorf("parsing ruleset file %s: %w", path, err)
	}
	return &rs, nil
}

// WriteRulesFile writes rules to a YAML file.
func WriteRulesFile(path string, rules []Rule) error {
	data, err := yaml.Marshal(rules)
	if err != nil {
		return fmt.Errorf("marshaling rules: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing rules file %s: %w", path, err)
	}
	return nil
}

// WriteRuleset writes a ruleset.yaml file.
func WriteRuleset(path string, rs *Ruleset) error {
	data, err := yaml.Marshal(rs)
	if err != nil {
		return fmt.Errorf("marshaling ruleset: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing ruleset file %s: %w", path, err)
	}
	return nil
}

// WriteRulesGrouped writes rules to separate files grouped by concern.
// The concern key is used as the filename (e.g., "security" → "security.yaml").
// Rules without a concern are written to "general.yaml".
func WriteRulesGrouped(dir string, grouped map[string][]Rule) error {
	for concern, rules := range grouped {
		if concern == "" {
			concern = "general"
		}
		path := filepath.Join(dir, concern+".yaml")
		if err := WriteRulesFile(path, rules); err != nil {
			return err
		}
	}
	return nil
}
