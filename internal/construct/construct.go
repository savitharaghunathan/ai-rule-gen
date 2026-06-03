package construct

import (
	"fmt"
	"sort"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// GroupCount holds the rule count for a single group file.
type GroupCount struct {
	File  string `json:"file"`
	Rules int    `json:"rules"`
}

// Result holds the output of a construct operation.
type Result struct {
	RulesWritten     int                     `json:"rules_written"`
	FilesWritten     int                     `json:"files_written"`
	OutputDir        string                  `json:"output_dir"`
	Groups           []GroupCount            `json:"groups"`
	Errors           []string                `json:"errors,omitempty"`
	PatternRuleMap   map[int]string          `json:"pattern_rule_map,omitempty"`
	Grouped          map[string][]rules.Rule `json:"-"`
	Ruleset          *rules.Ruleset          `json:"-"`
}

// Run reads patterns from an ExtractOutput, converts them to rules, validates, and writes output.
func Run(extract *rules.ExtractOutput, outputDir string) (*Result, error) {
	if len(extract.Patterns) == 0 {
		return nil, fmt.Errorf("no patterns found in input")
	}
	if len(extract.Targets) == 0 {
		return nil, fmt.Errorf("targets are required in patterns.json")
	}

	sources := extract.Sources
	targets := extract.Targets

	prefixGens := make(map[string]*rules.IDGenerator)
	grouped := make(map[string][]rules.Rule)
	patternRuleMap := make(map[int]string)

	for i, p := range extract.Patterns {
		concern := p.Concern
		if concern == "" {
			concern = "general"
		}
		changeType := rules.ChangeType(p.LocationType, p.ProviderType, p.DependencyName, p.XPath)
		prefix := rules.RuleIDPrefix(concern, changeType)
		if _, ok := prefixGens[prefix]; !ok {
			prefixGens[prefix] = rules.NewIDGenerator()
		}
		rule := patternToRule(p, prefixGens[prefix], prefix, sources, targets)
		patternRuleMap[i] = rule.RuleID
		grouped[concern] = append(grouped[concern], rule)
	}

	// Validate all rules
	var allRules []rules.Rule
	for _, rr := range grouped {
		allRules = append(allRules, rr...)
	}
	validationResult := rules.Validate(allRules)
	if !validationResult.Valid {
		return &Result{
			Errors: validationResult.Errors,
		}, fmt.Errorf("validation failed: %s", strings.Join(validationResult.Errors, "; "))
	}

	// Write grouped rule files
	if err := rules.WriteRulesGrouped(outputDir, grouped); err != nil {
		return nil, fmt.Errorf("writing rules: %w", err)
	}

	// Write ruleset.yaml
	var rulesetLabels []string
	for _, s := range sources {
		rulesetLabels = append(rulesetLabels, fmt.Sprintf("konveyor.io/source=%s", s))
	}
	for _, t := range targets {
		rulesetLabels = append(rulesetLabels, fmt.Sprintf("konveyor.io/target=%s", t))
	}
	var rulesetName, rulesetDesc string
	if len(sources) > 0 {
		rulesetName = fmt.Sprintf("%s/%s", targets[0], sources[0])
		rulesetDesc = fmt.Sprintf("Rules for migrating from %s to %s", sources[0], targets[0])
	} else {
		rulesetName = targets[0]
		rulesetDesc = fmt.Sprintf("Rules for migrating to %s", targets[0])
	}
	ruleset := &rules.Ruleset{
		Name:        rulesetName,
		Description: rulesetDesc,
		Labels:      rulesetLabels,
	}
	rulesetPath := fmt.Sprintf("%s/ruleset.yaml", outputDir)
	if err := rules.WriteRuleset(rulesetPath, ruleset); err != nil {
		return nil, fmt.Errorf("writing ruleset: %w", err)
	}

	filesWritten := len(grouped) + 1 // rule files + ruleset.yaml

	groups := make([]GroupCount, 0, len(grouped))
	for name, rr := range grouped {
		groups = append(groups, GroupCount{File: name + ".yaml", Rules: len(rr)})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].File < groups[j].File })

	return &Result{
		RulesWritten:   len(allRules),
		FilesWritten:   filesWritten,
		OutputDir:      outputDir,
		Groups:         groups,
		PatternRuleMap: patternRuleMap,
		Grouped:        grouped,
		Ruleset:        ruleset,
	}, nil
}

func patternToRule(p rules.MigrationPattern, idGen *rules.IDGenerator, prefix string, sources, targets []string) rules.Rule {
	condition := buildCondition(p)

	message := p.Message
	if message == "" {
		message = fmt.Sprintf("%s: %s", p.SourcePattern, p.Rationale)
	}

	return rules.Rule{
		RuleID:      idGen.Next(prefix),
		Description: p.Rationale,
		Category:    rules.Category(p.Category),
		Effort:      rules.ComplexityToEffort(p.Complexity),
		Labels:      rules.InitialLabels(sources, targets),
		Message:     message,
		Links:       buildLinks(p),
		When:        condition,
	}
}

func buildCondition(p rules.MigrationPattern) rules.Condition {
	if len(p.AlternativeFQNs) > 0 {
		conditions := make([]rules.Condition, 0, len(p.AlternativeFQNs)+1)
		conditions = append(conditions, buildSingleCondition(p))
		for _, fqn := range p.AlternativeFQNs {
			alt := p
			alt.SourceFQN = fqn
			conditions = append(conditions, buildSingleCondition(alt))
		}
		return rules.NewOr(conditions...)
	}
	return buildSingleCondition(p)
}

func buildSingleCondition(p rules.MigrationPattern) rules.Condition {
	// Dependency conditions (java.dependency, go.dependency)
	if p.DependencyName != "" {
		switch p.ProviderType {
		case "go":
			return rules.NewGoDependency(p.DependencyName, p.LowerBound, p.UpperBound)
		default:
			return rules.NewJavaDependency(p.DependencyName, p.LowerBound, p.UpperBound)
		}
	}

	// XML conditions (builtin.xml)
	if p.XPath != "" {
		c := rules.NewBuiltinXML(p.XPath, p.Namespaces)
		if len(p.XPathFilepaths) > 0 {
			c.BuiltinXML.Filepaths = p.XPathFilepaths
		}
		return c
	}

	// Referenced and filecontent conditions
	pattern := p.SourceFQN
	if pattern == "" {
		pattern = p.SourcePattern
	}
	switch p.ProviderType {
	case "java":
		return rules.NewJavaReferenced(pattern, p.LocationType)
	case "go":
		return rules.NewGoReferenced(pattern)
	case "nodejs":
		return rules.NewNodejsReferenced(pattern)
	case "csharp":
		return rules.NewCSharpReferenced(pattern, p.LocationType)
	case "python":
		return rules.NewPythonReferenced(pattern)
	case "builtin":
		return rules.NewBuiltinFilecontent(pattern, p.FilePattern)
	default:
		if p.LocationType != "" {
			return rules.NewJavaReferenced(pattern, p.LocationType)
		}
		return rules.NewBuiltinFilecontent(pattern, p.FilePattern)
	}
}

func buildLinks(p rules.MigrationPattern) []rules.Link {
	if p.DocumentationURL == "" {
		return nil
	}
	return []rules.Link{{
		URL:   p.DocumentationURL,
		Title: "Migration Documentation",
	}}
}


