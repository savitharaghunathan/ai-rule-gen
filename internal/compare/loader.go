package compare

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RawRule holds the fields compare needs (ruleID, description, when) as raw
// yaml.Node, sidestepping the strict types in internal/rules. Lets the diff
// read rules whose YAML uses template chains or field-name aliases the
// strict types would reject.
type RawRule struct {
	RuleID      string
	Description string
	When        *yaml.Node
}

// LoadRulesDirRaw reads every rule yaml in dir (skipping ruleset.yaml).
func LoadRulesDirRaw(dir string) ([]RawRule, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading rules directory %s: %w", dir, err)
	}

	var all []RawRule
	var warnings []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if name == "ruleset.yaml" || name == "ruleset.yml" {
			continue
		}
		fileRules, fileWarnings := loadFileRaw(filepath.Join(dir, name))
		all = append(all, fileRules...)
		warnings = append(warnings, fileWarnings...)
	}
	return all, warnings, nil
}

func loadFileRaw(path string) ([]RawRule, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: read failed: %v", path, err)}
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, []string{fmt.Sprintf("%s: parse failed: %v", path, err)}
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil
	}
	list := doc.Content[0]
	if list.Kind != yaml.SequenceNode {
		return nil, nil
	}

	var out []RawRule
	for _, n := range list.Content {
		if n.Kind != yaml.MappingNode {
			continue
		}
		r := RawRule{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i].Value
			v := n.Content[i+1]
			switch k {
			case "ruleID":
				r.RuleID = v.Value
			case "description":
				r.Description = v.Value
			case "when":
				r.When = v
			}
		}
		if r.RuleID == "" {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
