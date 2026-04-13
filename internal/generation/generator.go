package generation

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
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

// Generate converts migration patterns into validated rules.
func (g *Generator) Generate(ctx context.Context, patterns []extraction.MigrationPattern, input GenerateInput) ([]rules.Rule, *rules.Ruleset, error) {
	prefix := rulePrefix(input.Source, input.Target)
	idGen := NewIDGenerator(prefix)

	var ruleList []rules.Rule

	for _, p := range patterns {
		rule, err := g.patternToRule(ctx, p, idGen, input)
		if err != nil {
			return nil, nil, fmt.Errorf("generating rule for %q: %w", p.SourcePattern, err)
		}
		ruleList = append(ruleList, rule)
	}

	ruleset := &rules.Ruleset{
		Name:        fmt.Sprintf("%s/%s", input.Target, input.Source),
		Description: fmt.Sprintf("Rules for migrating from %s to %s", input.Source, input.Target),
		Labels: []string{
			fmt.Sprintf("konveyor.io/source=%s", input.Source),
			fmt.Sprintf("konveyor.io/target=%s", input.Target),
		},
	}

	return ruleList, ruleset, nil
}

func (g *Generator) patternToRule(ctx context.Context, p extraction.MigrationPattern, idGen *IDGenerator, input GenerateInput) (rules.Rule, error) {
	condition := buildCondition(p)
	message, err := g.generateMessage(ctx, p, input)
	if err != nil {
		slog.Warn("LLM message generation failed, using fallback", "pattern", p.SourcePattern, "error", err)
		message = fmt.Sprintf("%s: %s", p.SourcePattern, p.Rationale)
	}

	return rules.Rule{
		RuleID:      idGen.Next(),
		Description: truncate(p.Rationale, 120),
		Category:    rules.Category(p.Category),
		Effort:      complexityToEffort(p.Complexity),
		Labels: rules.InitialLabels([]string{
			fmt.Sprintf("konveyor.io/source=%s", input.Source),
			fmt.Sprintf("konveyor.io/target=%s", input.Target),
		}),
		Message: message,
		Links:   buildLinks(p),
		When:    condition,
	}, nil
}

func buildCondition(p extraction.MigrationPattern) rules.Condition {
	// If there are alternative FQNs, create an or combinator
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

func buildSingleCondition(p extraction.MigrationPattern) rules.Condition {
	pattern := p.SourceFQN
	if pattern == "" {
		pattern = p.SourcePattern
	}

	// Use condition_type if specified (more precise than provider_type)
	condType := p.ConditionType
	if condType == "" {
		// Fall back to provider_type for backward compatibility
		condType = p.ProviderType + ".referenced"
		if p.ProviderType == "builtin" {
			condType = "builtin.filecontent"
		}
	}

	switch condType {
	case "java.referenced":
		pattern = ensureJavaPatternMatchable(pattern)
		return rules.NewJavaReferenced(pattern, p.LocationType)
	case "java.dependency":
		name := p.DependencyName
		if name == "" {
			name = pattern
		}
		return rules.NewJavaDependency(name, p.DepLowerbound, p.DepUpperbound)
	case "go.referenced":
		return rules.NewGoReferenced(pattern)
	case "go.dependency":
		name := p.DependencyName
		if name == "" {
			name = pattern
		}
		return rules.NewGoDependency(name, p.DepLowerbound, p.DepUpperbound)
	case "nodejs.referenced":
		return rules.NewNodejsReferenced(pattern)
	case "csharp.referenced":
		return rules.NewCSharpReferenced(pattern, p.LocationType)
	case "builtin.filecontent":
		return rules.NewBuiltinFilecontent(pattern, p.FilePattern)
	case "builtin.file":
		return rules.NewBuiltinFile(pattern)
	case "builtin.xml":
		xpath := p.XMLXPath
		if xpath == "" {
			xpath = pattern
		}
		cond := rules.NewBuiltinXML(xpath, p.XMLNamespaces)
		if len(p.XMLFilepaths) > 0 {
			cond.BuiltinXML.Filepaths = p.XMLFilepaths
		}
		return cond
	case "builtin.json":
		return rules.NewBuiltinJSON(pattern)
	default:
		// Best-effort fallback
		if p.LocationType != "" {
			return rules.NewJavaReferenced(pattern, p.LocationType)
		}
		return rules.NewBuiltinFilecontent(pattern, p.FilePattern)
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

// ensureJavaPatternMatchable appends a wildcard to package-level patterns
// that would otherwise not match any specific class. The analyzer requires
// "javax.xml.bind*" (with wildcard) to match classes like javax.xml.bind.JAXBContext.
// A bare "javax.xml.bind" only matches an exact reference to the package itself.
func ensureJavaPatternMatchable(pattern string) string {
	if pattern == "" || strings.HasSuffix(pattern, "*") {
		return pattern
	}
	// If any segment starts with uppercase, the pattern references a specific
	// class or method (e.g., org.junit.Assert.assertEquals) — leave it as-is.
	for _, part := range strings.Split(pattern, ".") {
		if len(part) > 0 && part[0] >= 'A' && part[0] <= 'Z' {
			return pattern
		}
	}
	// If it contains parens (method signature) or angle brackets, leave it.
	if strings.ContainsAny(pattern, "(){}[]<>") {
		return pattern
	}
	// All-lowercase segments: this is a package prefix — append wildcard.
	return pattern + "*"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
