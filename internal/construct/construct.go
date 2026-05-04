package construct

import (
	"fmt"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// Result holds the output of a construct operation.
type Result struct {
	RulesWritten int                     `json:"rules_written"`
	FilesWritten int                     `json:"files_written"`
	OutputDir    string                  `json:"output_dir"`
	Errors       []string                `json:"errors,omitempty"`
	Grouped      map[string][]rules.Rule `json:"-"`
	Ruleset      *rules.Ruleset          `json:"-"`
}

// Run reads patterns from an ExtractOutput, converts them to rules, validates, and writes output.
func Run(extract *rules.ExtractOutput, outputDir string) (*Result, error) {
	if len(extract.Patterns) == 0 {
		return nil, fmt.Errorf("no patterns found in input")
	}

	prefix := rulePrefix(extract.Source, extract.Target)
	idGen := rules.NewIDGenerator(prefix)

	grouped := make(map[string][]rules.Rule)

	for _, p := range extract.Patterns {
		rule := patternToRule(p, idGen, extract.Source, extract.Target)
		concern := p.Concern
		if concern == "" {
			concern = "general"
		}
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
	ruleset := &rules.Ruleset{
		Name:        fmt.Sprintf("%s/%s", extract.Target, extract.Source),
		Description: fmt.Sprintf("Rules for migrating from %s to %s", extract.Source, extract.Target),
		Labels: []string{
			fmt.Sprintf("konveyor.io/source=%s", extract.Source),
			fmt.Sprintf("konveyor.io/target=%s", extract.Target),
		},
	}
	rulesetPath := fmt.Sprintf("%s/ruleset.yaml", outputDir)
	if err := rules.WriteRuleset(rulesetPath, ruleset); err != nil {
		return nil, fmt.Errorf("writing ruleset: %w", err)
	}

	filesWritten := len(grouped) + 1 // rule files + ruleset.yaml

	return &Result{
		RulesWritten: len(allRules),
		FilesWritten: filesWritten,
		OutputDir:    outputDir,
		Grouped:      grouped,
		Ruleset:      ruleset,
	}, nil
}

func patternToRule(p rules.MigrationPattern, idGen *rules.IDGenerator, source, target string) rules.Rule {
	condition := buildCondition(p)

	message := p.Message
	if message == "" {
		message = fmt.Sprintf("%s: %s", p.SourcePattern, p.Rationale)
	}

	return rules.Rule{
		RuleID:      idGen.Next(),
		Description: truncate(p.Rationale, 120),
		Category:    rules.Category(p.Category),
		Effort:      rules.ComplexityToEffort(p.Complexity),
		Labels:      rules.InitialLabels(source, target),
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

func rulePrefix(source, target string) string {
	return fmt.Sprintf("%s-to-%s", sanitize(source), sanitize(target))
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
