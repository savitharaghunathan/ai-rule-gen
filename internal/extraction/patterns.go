package extraction

// MigrationPattern is an intermediate type bridging ingested content and rule generation.
// Extracted by LLM from migration guides, changelogs, or code.
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
	ConditionType    string   `json:"condition_type,omitempty"`
	FilePattern      string   `json:"file_pattern,omitempty"`
	DependencyName   string   `json:"dependency_name,omitempty"`
	DepUpperbound    string   `json:"dep_upperbound,omitempty"`
	DepLowerbound    string   `json:"dep_lowerbound,omitempty"`
	ExampleBefore    string   `json:"example_before,omitempty"`
	ExampleAfter     string   `json:"example_after,omitempty"`
	DocumentationURL string   `json:"documentation_url,omitempty"`
}

// Deduplicate removes duplicate patterns based on source_fqn + source_pattern.
func Deduplicate(patterns []MigrationPattern) []MigrationPattern {
	seen := make(map[string]bool)
	var unique []MigrationPattern
	for _, p := range patterns {
		key := p.SourceFQN + "|" + p.SourcePattern
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, p)
	}
	return unique
}
