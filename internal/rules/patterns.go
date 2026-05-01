package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ExtractOutput is the intermediate format between agent pattern extraction and rule construction.
// The agent writes this as patterns.json; `rulegen construct` reads it.
type ExtractOutput struct {
	Source   string             `json:"source"`
	Target   string            `json:"target"`
	Language string             `json:"language,omitempty"`
	Patterns []MigrationPattern `json:"patterns"`
}

// MigrationPattern represents a single migration pattern extracted from a guide.
type MigrationPattern struct {
	SourcePattern    string   `json:"source_pattern"`
	TargetPattern    string   `json:"target_pattern,omitempty"`
	SourceFQN        string   `json:"source_fqn,omitempty"`
	LocationType     string   `json:"location_type,omitempty"`
	AlternativeFQNs  []string `json:"alternative_fqns,omitempty"`
	Rationale        string   `json:"rationale"`
	Complexity       string   `json:"complexity"`
	Category         string   `json:"category"`
	Concern          string   `json:"concern,omitempty"`
	ProviderType     string   `json:"provider_type,omitempty"`
	FilePattern      string   `json:"file_pattern,omitempty"`
	ExampleBefore    string   `json:"example_before,omitempty"`
	ExampleAfter     string   `json:"example_after,omitempty"`
	DocumentationURL string   `json:"documentation_url,omitempty"`
	Message          string   `json:"message,omitempty"`

	// Dependency condition fields (java.dependency, go.dependency).
	// When set, construct produces a dependency condition instead of a referenced condition.
	DependencyName string `json:"dependency_name,omitempty"`
	UpperBound     string `json:"upper_bound,omitempty"`
	LowerBound     string `json:"lower_bound,omitempty"`

	// XML condition fields (builtin.xml).
	// When set, construct produces a builtin.xml condition instead of builtin.filecontent.
	XPath          string            `json:"xpath,omitempty"`
	Namespaces     map[string]string `json:"namespaces,omitempty"`
	XPathFilepaths []string          `json:"xpath_filepaths,omitempty"`
}

// ReadPatternsFile reads an ExtractOutput from a JSON file.
func ReadPatternsFile(path string) (*ExtractOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading patterns file %s: %w", path, err)
	}
	var output ExtractOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parsing patterns file %s: %w", path, err)
	}
	return &output, nil
}

// MergePatterns combines multiple ExtractOutputs into one, deduplicating by
// source_fqn and dependency_name. The first occurrence wins.
func MergePatterns(parts []*ExtractOutput) *ExtractOutput {
	if len(parts) == 0 {
		return &ExtractOutput{}
	}
	merged := &ExtractOutput{
		Source:   parts[0].Source,
		Target:   parts[0].Target,
		Language: parts[0].Language,
	}

	seen := make(map[string]bool)
	for _, part := range parts {
		if merged.Language == "" && part.Language != "" {
			merged.Language = part.Language
		}
		for _, p := range part.Patterns {
			key := deduplicationKey(p)
			if key != "" && seen[key] {
				continue
			}
			if key != "" {
				seen[key] = true
			}
			merged.Patterns = append(merged.Patterns, p)
		}
	}
	return merged
}

func deduplicationKey(p MigrationPattern) string {
	if p.SourceFQN != "" {
		return "fqn:" + p.SourceFQN
	}
	if p.DependencyName != "" {
		return "dep:" + p.DependencyName
	}
	if p.XPath != "" {
		return "xpath:" + p.XPath
	}
	return ""
}

// WritePatternsFile writes an ExtractOutput to a JSON file.
func WritePatternsFile(path string, output *ExtractOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling patterns: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// ComplexityToEffort converts a human-readable complexity to a numeric effort value.
func ComplexityToEffort(complexity string) int {
	switch strings.ToLower(complexity) {
	case "trivial":
		return 1
	case "low":
		return 3
	case "medium":
		return 5
	case "high":
		return 7
	case "expert":
		return 9
	default:
		return 5
	}
}

// InitialLabels returns the default labels for a newly generated rule.
func InitialLabels(source, target string) []string {
	return []string{
		fmt.Sprintf("konveyor.io/source=%s", source),
		fmt.Sprintf("konveyor.io/target=%s", target),
		"konveyor.io/generated-by=ai-rule-gen",
		"konveyor.io/test-result=untested",
		"konveyor.io/review=unreviewed",
	}
}
