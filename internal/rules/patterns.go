package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// ArtifactCoordinates identifies a published library artifact in a package registry.
type ArtifactCoordinates struct {
	GroupID    string `json:"group_id"`
	ArtifactID string `json:"artifact_id"`
	Version    string `json:"version"`
}

// ExtractOutput is the intermediate format between agent pattern extraction and rule construction.
// The agent writes this as patterns.json; `rulegen construct` reads it.
type ExtractOutput struct {
	Sources  []string           `json:"sources"`
	Targets  []string           `json:"targets"`
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

	// Source artifact coordinates for deterministic verification.
	// When set, the verifier downloads this artifact and checks that SourceFQN exists in it.
	SourceArtifact *ArtifactCoordinates `json:"source_artifact,omitempty"`

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

// MergeResult contains the merged output plus statistics.
type MergeResult struct {
	Output     *ExtractOutput
	Duplicates int
	Absorbed   int // patterns folded into package-level rules
}

// MergePatterns combines multiple ExtractOutputs into one, deduplicating by
// source_fqn and dependency_name (first occurrence wins), then consolidating
// IMPORT/TYPE patterns into matching PACKAGE patterns.
func MergePatterns(parts []*ExtractOutput) *MergeResult {
	if len(parts) == 0 {
		return &MergeResult{Output: &ExtractOutput{}}
	}
	merged := &ExtractOutput{
		Sources:  append([]string{}, parts[0].Sources...),
		Targets:  append([]string{}, parts[0].Targets...),
		Language: parts[0].Language,
	}

	totalPatterns := 0
	seen := make(map[string]bool)
	for _, part := range parts {
		if merged.Language == "" && part.Language != "" {
			merged.Language = part.Language
		}
		merged.Sources = unionStrings(merged.Sources, part.Sources)
		merged.Targets = unionStrings(merged.Targets, part.Targets)
		for _, p := range part.Patterns {
			totalPatterns++
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

	duplicates := totalPatterns - len(merged.Patterns)
	consolidated, absorbed := consolidatePackages(merged.Patterns)
	merged.Patterns = consolidated

	return &MergeResult{
		Output:     merged,
		Duplicates: duplicates,
		Absorbed:   absorbed,
	}
}

// consolidatePackages folds IMPORT and TYPE patterns into any PACKAGE pattern
// whose source_fqn is a prefix of theirs. Absorbed patterns contribute a row
// to the PACKAGE pattern's message table. METHOD_CALL and other location types
// are kept as separate rules because they provide detection beyond the package
// import itself.
func consolidatePackages(patterns []MigrationPattern) ([]MigrationPattern, int) {
	pkgIndices := map[int]string{}
	for i, p := range patterns {
		if strings.EqualFold(p.LocationType, "PACKAGE") && p.SourceFQN != "" {
			pkgIndices[i] = p.SourceFQN
		}
	}
	if len(pkgIndices) == 0 {
		return patterns, 0
	}

	absorbable := func(lt string) bool {
		up := strings.ToUpper(lt)
		return up == "IMPORT" || up == "TYPE"
	}

	// Map each absorbable pattern to its parent PACKAGE index.
	// Longest prefix wins when multiple PACKAGE patterns could match.
	childToParent := map[int]int{}
	for i, p := range patterns {
		if !absorbable(p.LocationType) || p.SourceFQN == "" {
			continue
		}
		bestPkg := -1
		bestLen := 0
		for pi, pfqn := range pkgIndices {
			if pi == i {
				continue
			}
			prefix := pfqn + "."
			if strings.HasPrefix(p.SourceFQN, prefix) && len(pfqn) > bestLen {
				bestPkg = pi
				bestLen = len(pfqn)
			}
		}
		if bestPkg >= 0 && !isClassReplacement(p.SourceFQN, p.TargetPattern) {
			childToParent[i] = bestPkg
		}
	}

	if len(childToParent) == 0 {
		return patterns, 0
	}

	// Group absorbed patterns by parent and build the replacement table.
	parentChildren := map[int][]MigrationPattern{}
	for ci, pi := range childToParent {
		parentChildren[pi] = append(parentChildren[pi], patterns[ci])
	}

	absorbed := map[int]bool{}
	for ci := range childToParent {
		absorbed[ci] = true
	}

	for pi, children := range parentChildren {
		pkg := &patterns[pi]

		// Build a markdown table of specific replacements.
		var table strings.Builder
		table.WriteString("\n\n### Specific replacements\n\n")
		table.WriteString("| Old class | Replacement |\n|---|---|\n")
		for _, c := range children {
			replacement := c.TargetPattern
			if replacement == "" {
				replacement = c.Rationale
			}
			table.WriteString(fmt.Sprintf("| `%s` | %s |\n", c.SourceFQN, replacement))
		}
		pkg.Message += table.String()

		// Inherit the highest complexity.
		for _, c := range children {
			if complexityRank(c.Complexity) > complexityRank(pkg.Complexity) {
				pkg.Complexity = c.Complexity
			}
		}
	}

	// Rebuild the slice, skipping absorbed patterns.
	result := make([]MigrationPattern, 0, len(patterns)-len(absorbed))
	for i, p := range patterns {
		if !absorbed[i] {
			result = append(result, p)
		}
	}

	return result, len(absorbed)
}

func complexityRank(c string) int {
	switch strings.ToLower(c) {
	case "trivial":
		return 1
	case "low":
		return 2
	case "medium":
		return 3
	case "high":
		return 4
	case "expert":
		return 5
	default:
		return 0
	}
}

// isClassReplacement returns true when the migration replaces a class with a
// differently-named class (not just a package move). Such patterns should NOT
// be absorbed into a parent PACKAGE rule.
func isClassReplacement(sourceFQN, targetPattern string) bool {
	if targetPattern == "" {
		return false
	}
	className := sourceFQN
	if idx := strings.LastIndex(sourceFQN, "."); idx >= 0 {
		className = sourceFQN[idx+1:]
	}
	if className == "" {
		return false
	}
	return !containsWord(targetPattern, className)
}

// containsWord returns true if text contains word as a standalone token,
// bounded on both sides by a non-alphanumeric character or string edge.
func containsWord(text, word string) bool {
	start := 0
	for {
		idx := strings.Index(text[start:], word)
		if idx < 0 {
			return false
		}
		abs := start + idx
		leftOK := abs == 0 || !isAlphanumeric(rune(text[abs-1]))
		rightOK := abs+len(word) == len(text) || !isAlphanumeric(rune(text[abs+len(word)]))
		if leftOK && rightOK {
			return true
		}
		start = abs + 1
	}
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
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
// Each source and target gets its own konveyor.io label.
func InitialLabels(sources, targets []string) []string {
	var labels []string
	for _, s := range sources {
		labels = append(labels, fmt.Sprintf("konveyor.io/source=%s", s))
	}
	for _, t := range targets {
		labels = append(labels, fmt.Sprintf("konveyor.io/target=%s", t))
	}
	labels = append(labels, "konveyor.io/generated-by=ai-rule-gen")
	return labels
}

// unionStrings returns the union of two string slices, preserving order and deduplicating.
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
