package rules

import "gopkg.in/yaml.v3"

// Filepaths accepts a YAML sequence or a scalar. Hand-authored rulesets use a
// scalar for chained-variable templates like `'{{xmlfiles1.filepaths}}'`;
// kantra resolves the template at analysis time. Marshals back as a sequence.
type Filepaths []string

func (f *Filepaths) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		*f = Filepaths{node.Value}
		return nil
	case yaml.SequenceNode:
		var out []string
		if err := node.Decode(&out); err != nil {
			return err
		}
		*f = Filepaths(out)
		return nil
	default:
		return &yaml.TypeError{Errors: []string{"filepaths: expected scalar or sequence"}}
	}
}

// UnmarshalYAML accepts `name_regex` or its `nameregex` shorthand.
func (d *Dependency) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return &yaml.TypeError{Errors: []string{"dependency: expected mapping"}}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i].Value
		v := node.Content[i+1]
		switch k {
		case "name":
			if err := v.Decode(&d.Name); err != nil {
				return err
			}
		case "name_regex", "nameregex":
			if err := v.Decode(&d.NameRegex); err != nil {
				return err
			}
		case "upperbound":
			if err := v.Decode(&d.Upperbound); err != nil {
				return err
			}
		case "lowerbound":
			if err := v.Decode(&d.Lowerbound); err != nil {
				return err
			}
		}
	}
	return nil
}
