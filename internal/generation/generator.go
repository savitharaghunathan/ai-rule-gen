package generation

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/extraction"
	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// Generator constructs rules from extracted migration patterns.
type Generator struct {
	completer  llm.Completer
	messageTmpl *template.Template
}

// New creates a Generator with an LLM completer for message generation.
func New(completer llm.Completer, messageTmpl *template.Template) *Generator {
	return &Generator{completer: completer, messageTmpl: messageTmpl}
}

// GenerateInput holds metadata for rule generation.
type GenerateInput struct {
	Source   string
	Target   string
	Language string
}

// Generate converts migration patterns into validated rules,
// grouped by concern.
func (g *Generator) Generate(ctx context.Context, patterns []extraction.MigrationPattern, input GenerateInput) (map[string][]rules.Rule, *rules.Ruleset, error) {
	prefix := rulePrefix(input.Source, input.Target)
	idGen := NewIDGenerator(prefix)

	grouped := make(map[string][]rules.Rule)

	for _, p := range patterns {
		rule, err := g.patternToRule(ctx, p, idGen, input)
		if err != nil {
			return nil, nil, fmt.Errorf("generating rule for %q: %w", p.SourcePattern, err)
		}
		concern := p.Concern
		if concern == "" {
			concern = "general"
		}
		grouped[concern] = append(grouped[concern], rule)
	}

	ruleset := &rules.Ruleset{
		Name:        fmt.Sprintf("%s/%s", input.Target, input.Source),
		Description: fmt.Sprintf("Rules for migrating from %s to %s", input.Source, input.Target),
		Labels: []string{
			fmt.Sprintf("konveyor.io/source=%s", input.Source),
			fmt.Sprintf("konveyor.io/target=%s", input.Target),
		},
	}

	return grouped, ruleset, nil
}

func (g *Generator) patternToRule(ctx context.Context, p extraction.MigrationPattern, idGen *IDGenerator, input GenerateInput) (rules.Rule, error) {
	condition := buildCondition(p)
	message, err := g.generateMessage(ctx, p, input)
	if err != nil {
		// Fall back to a simple message if LLM fails
		message = fmt.Sprintf("%s: %s", p.SourcePattern, p.Rationale)
	}

	return rules.Rule{
		RuleID:      idGen.Next(),
		Description: truncate(p.Rationale, 120),
		Category:    rules.Category(p.Category),
		Effort:      complexityToEffort(p.Complexity),
		Labels: []string{
			fmt.Sprintf("konveyor.io/source=%s", input.Source),
			fmt.Sprintf("konveyor.io/target=%s", input.Target),
		},
		Message: message,
		Links:   buildLinks(p),
		When:    condition,
	}, nil
}

func buildCondition(p extraction.MigrationPattern) rules.Condition {
	// If there are alternative FQNs, create an or combinator
	if len(p.AlternativeFQNs) > 0 {
		conditions := make([]rules.Condition, 0, len(p.AlternativeFQNs)+1)
		conditions = append(conditions, buildSingleCondition(p.ProviderType, p.SourceFQN, p.LocationType, p.FilePattern))
		for _, fqn := range p.AlternativeFQNs {
			conditions = append(conditions, buildSingleCondition(p.ProviderType, fqn, p.LocationType, p.FilePattern))
		}
		return rules.NewOr(conditions...)
	}

	return buildSingleCondition(p.ProviderType, p.SourceFQN, p.LocationType, p.FilePattern)
}

func buildSingleCondition(providerType, fqn, locationType, filePattern string) rules.Condition {
	// Use source_fqn if available, otherwise fall back to a generic pattern
	pattern := fqn
	if pattern == "" {
		pattern = "TODO-set-pattern"
	}

	switch providerType {
	case "java":
		return rules.NewJavaReferenced(pattern, locationType)
	case "go":
		return rules.NewGoReferenced(pattern)
	case "nodejs":
		return rules.NewNodejsReferenced(pattern)
	case "csharp":
		return rules.NewCSharpReferenced(pattern, locationType)
	case "builtin":
		return rules.NewBuiltinFilecontent(pattern, filePattern)
	default:
		// Default to java.referenced for Java-like patterns
		if locationType != "" {
			return rules.NewJavaReferenced(pattern, locationType)
		}
		return rules.NewBuiltinFilecontent(pattern, filePattern)
	}
}

func (g *Generator) generateMessage(ctx context.Context, p extraction.MigrationPattern, input GenerateInput) (string, error) {
	if g.messageTmpl == nil {
		return fmt.Sprintf("%s: %s", p.SourcePattern, p.Rationale), nil
	}

	var buf bytes.Buffer
	data := map[string]string{
		"Source":        input.Source,
		"Target":        input.Target,
		"SourcePattern": p.SourcePattern,
		"TargetPattern": p.TargetPattern,
		"Rationale":     p.Rationale,
		"ExampleBefore": p.ExampleBefore,
		"ExampleAfter":  p.ExampleAfter,
	}
	if err := g.messageTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering message template: %w", err)
	}

	return g.completer.Complete(ctx, buf.String())
}

func buildLinks(p extraction.MigrationPattern) []rules.Link {
	if p.DocumentationURL == "" {
		return nil
	}
	return []rules.Link{{
		URL:   p.DocumentationURL,
		Title: "Migration Documentation",
	}}
}

func complexityToEffort(complexity string) int {
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

func rulePrefix(source, target string) string {
	s := sanitize(source)
	t := sanitize(target)
	return fmt.Sprintf("%s-to-%s", s, t)
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
