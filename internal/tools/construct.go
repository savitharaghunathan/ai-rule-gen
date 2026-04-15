package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/extraction"
	"github.com/konveyor/ai-rule-gen/internal/generation"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

type constructRuleInput struct {
	RuleID        string            `json:"ruleID"`
	ConditionType string            `json:"condition_type"`
	Pattern       string            `json:"pattern,omitempty"`
	Location      string            `json:"location,omitempty"`
	Name          string            `json:"name,omitempty"`
	NameRegex     string            `json:"nameRegex,omitempty"`
	Lowerbound    string            `json:"lowerbound,omitempty"`
	Upperbound    string            `json:"upperbound,omitempty"`
	XPath         string            `json:"xpath,omitempty"`
	Namespaces    map[string]string `json:"namespaces,omitempty"`
	Filepaths     []string          `json:"filepaths,omitempty"`
	Regex         string            `json:"regex,omitempty"`
	FilePattern   string            `json:"filePattern,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Message       string            `json:"message"`
	Category      string            `json:"category"`
	Effort        int               `json:"effort"`
	Description   string            `json:"description,omitempty"`
	Labels        []string          `json:"labels,omitempty"`
	Links         []rules.Link      `json:"links,omitempty"`
}

type constructRuleOutput struct {
	YAML   string   `json:"yaml"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ConstructRuleHandler returns an MCP tool handler for construct_rule.
func ConstructRuleHandler() mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input constructRuleInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if input.RuleID == "" || input.Message == "" || input.Category == "" {
			return errorResult("ruleID, message, and category are required"), nil
		}
		if input.ConditionType == "" {
			return errorResult("condition_type is required"), nil
		}

		condition, err := buildConditionFromInput(input)
		if err != nil {
			out := constructRuleOutput{Valid: false, Errors: []string{err.Error()}}
			data, _ := json.Marshal(out)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
			}, nil
		}

		rule := rules.Rule{
			RuleID:      input.RuleID,
			Description: input.Description,
			Category:    rules.Category(input.Category),
			Effort:      input.Effort,
			Labels:      input.Labels,
			Message:     input.Message,
			Links:       input.Links,
			When:        condition,
		}

		// Validate the constructed rule
		result := rules.Validate([]rules.Rule{rule})
		if !result.Valid {
			var errStrs []string
			for _, e := range result.Errors {
				errStrs = append(errStrs, e)
			}
			out := constructRuleOutput{Valid: false, Errors: errStrs}
			data, _ := json.Marshal(out)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
			}, nil
		}

		// Marshal to YAML
		yamlBytes, err := yaml.Marshal([]rules.Rule{rule})
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal rule: %v", err)), nil
		}

		out := constructRuleOutput{YAML: string(yamlBytes), Valid: true}
		data, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil
	}
}

func buildConditionFromInput(input constructRuleInput) (rules.Condition, error) {
	switch input.ConditionType {
	case "java.referenced":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for java.referenced")
		}
		return rules.NewJavaReferenced(input.Pattern, input.Location), nil

	case "java.dependency":
		if input.Name == "" && input.NameRegex == "" {
			return rules.Condition{}, fmt.Errorf("name or nameRegex is required for java.dependency")
		}
		c := rules.NewJavaDependency(input.Name, input.Lowerbound, input.Upperbound)
		if input.NameRegex != "" {
			c.JavaDependency.NameRegex = input.NameRegex
		}
		return c, nil

	case "go.referenced":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for go.referenced")
		}
		return rules.NewGoReferenced(input.Pattern), nil

	case "go.dependency":
		if input.Name == "" && input.NameRegex == "" {
			return rules.Condition{}, fmt.Errorf("name or nameRegex is required for go.dependency")
		}
		c := rules.NewGoDependency(input.Name, input.Lowerbound, input.Upperbound)
		if input.NameRegex != "" {
			c.GoDependency.NameRegex = input.NameRegex
		}
		return c, nil

	case "nodejs.referenced":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for nodejs.referenced")
		}
		return rules.NewNodejsReferenced(input.Pattern), nil

	case "csharp.referenced":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for csharp.referenced")
		}
		return rules.NewCSharpReferenced(input.Pattern, input.Location), nil

	case "builtin.filecontent":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for builtin.filecontent")
		}
		c := rules.NewBuiltinFilecontent(input.Pattern, input.FilePattern)
		c.BuiltinFilecontent.Filepaths = input.Filepaths
		return c, nil

	case "builtin.file":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for builtin.file")
		}
		return rules.NewBuiltinFile(input.Pattern), nil

	case "builtin.xml":
		if input.XPath == "" {
			return rules.Condition{}, fmt.Errorf("xpath is required for builtin.xml")
		}
		c := rules.NewBuiltinXML(input.XPath, input.Namespaces)
		c.BuiltinXML.Filepaths = input.Filepaths
		return c, nil

	case "builtin.json":
		if input.XPath == "" {
			return rules.Condition{}, fmt.Errorf("xpath is required for builtin.json")
		}
		c := rules.NewBuiltinJSON(input.XPath)
		c.BuiltinJSON.Filepaths = input.Filepaths
		return c, nil

	case "builtin.hasTags":
		if len(input.Tags) == 0 {
			return rules.Condition{}, fmt.Errorf("tags is required for builtin.hasTags")
		}
		return rules.NewBuiltinHasTags(input.Tags), nil

	case "builtin.xmlPublicID":
		if input.Regex == "" {
			return rules.Condition{}, fmt.Errorf("regex is required for builtin.xmlPublicID")
		}
		c := rules.NewBuiltinXMLPublicID(input.Regex, input.Namespaces)
		c.BuiltinXMLPublicID.Filepaths = input.Filepaths
		return c, nil

	default:
		valid := []string{
			"java.referenced", "java.dependency", "go.referenced", "go.dependency",
			"nodejs.referenced", "csharp.referenced", "builtin.filecontent", "builtin.file",
			"builtin.xml", "builtin.json", "builtin.hasTags", "builtin.xmlPublicID",
		}
		return rules.Condition{}, fmt.Errorf("unsupported condition_type %q. Valid types: %s", input.ConditionType, strings.Join(valid, ", "))
	}
}

// ConstructInput is the JSON input format for the construct CLI command.
// The rules array uses the same schema as the MCP construct_rule tool.
type ConstructInput struct {
	Ruleset *constructRulesetInput `json:"ruleset,omitempty"`
	Rules   []constructRuleInput   `json:"rules"`
}

// ConstructResult holds the output of a construct operation.
type ConstructResult struct {
	OutputDir    string                 `json:"output_dir"`
	RuleCount    int                    `json:"rule_count"`
	FilesWritten []string               `json:"files_written"`
	Validation   rules.ValidationResult `json:"validation"`
}

// ExtractOutput is the JSON output of the extract pipeline.
// Contains raw MigrationPattern data + metadata for piping to construct.
type ExtractOutput struct {
	Source   string                        `json:"source"`
	Target   string                        `json:"target"`
	Language string                        `json:"language"`
	Patterns []extraction.MigrationPattern `json:"patterns"`
}

// ConstructRules builds rules from JSON input, validates them, and writes YAML files.
// Accepts two JSON formats:
//   - ConstructInput: {"rules": [...]} — pre-mapped fields from client LLM
//   - ExtractOutput: {"patterns": [...]} — raw MigrationPatterns from extract pipeline
func ConstructRules(jsonInput []byte, outputDir string) (*ConstructResult, error) {
	// Detect format by probing for "patterns" key
	var probe struct {
		Patterns json.RawMessage `json:"patterns"`
	}
	if err := json.Unmarshal(jsonInput, &probe); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	if len(probe.Patterns) > 0 && string(probe.Patterns) != "null" {
		// ExtractOutput format — convert patterns to ConstructInput first
		var ext ExtractOutput
		if err := json.Unmarshal(jsonInput, &ext); err != nil {
			return nil, fmt.Errorf("invalid ExtractOutput JSON: %w", err)
		}
		if len(ext.Patterns) == 0 {
			return nil, fmt.Errorf("at least one pattern is required in 'patterns' array")
		}
		ci := convertPatternsToRules(&ext)
		converted, err := json.Marshal(ci)
		if err != nil {
			return nil, fmt.Errorf("converting patterns: %w", err)
		}
		return constructFromInput(converted, outputDir)
	}

	return constructFromInput(jsonInput, outputDir)
}

func constructFromInput(jsonInput []byte, outputDir string) (*ConstructResult, error) {
	var input ConstructInput
	if err := json.Unmarshal(jsonInput, &input); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	if len(input.Rules) == 0 {
		return nil, fmt.Errorf("at least one rule is required in 'rules' array")
	}

	var ruleList []rules.Rule
	for i, ri := range input.Rules {
		if ri.RuleID == "" || ri.Message == "" || ri.Category == "" {
			return nil, fmt.Errorf("rules[%d]: ruleID, message, and category are required", i)
		}
		if ri.ConditionType == "" {
			return nil, fmt.Errorf("rules[%d] (%s): condition_type is required", i, ri.RuleID)
		}
		condition, err := buildConditionFromInput(ri)
		if err != nil {
			return nil, fmt.Errorf("rules[%d] (%s): %w", i, ri.RuleID, err)
		}
		ruleList = append(ruleList, rules.Rule{
			RuleID:      ri.RuleID,
			Description: ri.Description,
			Category:    rules.Category(ri.Category),
			Effort:      ri.Effort,
			Labels:      ri.Labels,
			Message:     ri.Message,
			Links:       ri.Links,
			When:        condition,
		})
	}

	result := rules.Validate(ruleList)
	if !result.Valid {
		return &ConstructResult{
			OutputDir:  outputDir,
			RuleCount:  len(ruleList),
			Validation: result,
		}, fmt.Errorf("validation failed: %d errors", len(result.Errors))
	}

	rulesDir := filepath.Join(outputDir, "rules")
	rulesPath := filepath.Join(rulesDir, "rules.yaml")
	if err := rules.WriteRulesFile(rulesPath, ruleList); err != nil {
		return nil, fmt.Errorf("writing rules: %w", err)
	}
	filesWritten := []string{"rules/rules.yaml"}

	if input.Ruleset != nil && input.Ruleset.Name != "" {
		rs := &rules.Ruleset{
			Name:        input.Ruleset.Name,
			Description: input.Ruleset.Description,
			Labels:      input.Ruleset.Labels,
		}
		rulesetPath := filepath.Join(rulesDir, "ruleset.yaml")
		if err := rules.WriteRuleset(rulesetPath, rs); err != nil {
			return nil, fmt.Errorf("writing ruleset: %w", err)
		}
		filesWritten = append(filesWritten, "rules/ruleset.yaml")
	}

	return &ConstructResult{
		OutputDir:    outputDir,
		RuleCount:    len(ruleList),
		FilesWritten: filesWritten,
		Validation:   result,
	}, nil
}

type constructRulesetInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Labels      []string `json:"labels,omitempty"`
}

type constructRulesetOutput struct {
	YAML  string `json:"yaml"`
	Valid bool   `json:"valid"`
}

// ConstructRulesetHandler returns an MCP tool handler for construct_ruleset.
func ConstructRulesetHandler() mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input constructRulesetInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if input.Name == "" {
			return errorResult("name is required"), nil
		}

		ruleset := rules.Ruleset{
			Name:        input.Name,
			Description: input.Description,
			Labels:      input.Labels,
		}

		yamlBytes, err := yaml.Marshal(ruleset)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal ruleset: %v", err)), nil
		}

		out := constructRulesetOutput{YAML: string(yamlBytes), Valid: true}
		data, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil
	}
}

// convertPatternsToRules converts ExtractOutput (raw MigrationPatterns) into
// ConstructInput with deterministic field mapping. No LLM, no roundtrip.
func convertPatternsToRules(ext *ExtractOutput) *ConstructInput {
	prefix := generation.RulePrefix(ext.Source, ext.Target)
	idGen := generation.NewIDGenerator(prefix)

	ci := &ConstructInput{
		Ruleset: &constructRulesetInput{
			Name:        fmt.Sprintf("%s/%s", ext.Target, ext.Source),
			Description: fmt.Sprintf("Rules for migrating from %s to %s", ext.Source, ext.Target),
			Labels: []string{
				fmt.Sprintf("konveyor.io/source=%s", ext.Source),
				fmt.Sprintf("konveyor.io/target=%s", ext.Target),
			},
		},
	}

	for _, p := range ext.Patterns {
		ci.Rules = append(ci.Rules, patternToConstructInput(p, idGen, ext.Source, ext.Target))
	}
	return ci
}

// patternToConstructInput maps a MigrationPattern to constructRuleInput fields directly.
func patternToConstructInput(p extraction.MigrationPattern, idGen *generation.IDGenerator, source, target string) constructRuleInput {
	pattern := p.SourceFQN
	if pattern == "" {
		pattern = p.SourcePattern
	}

	condType := p.ConditionType
	if condType == "" {
		condType = p.ProviderType + ".referenced"
		if p.ProviderType == "builtin" {
			condType = "builtin.filecontent"
		}
	}

	if condType == "java.referenced" {
		pattern = generation.EnsureJavaPatternMatchable(pattern)
	}

	message := buildDeterministicMessage(p)

	return constructRuleInput{
		RuleID:        idGen.Next(),
		ConditionType: condType,
		Pattern:       pattern,
		Location:      p.LocationType,
		Name:          p.DependencyName,
		Lowerbound:    p.DepLowerbound,
		Upperbound:    p.DepUpperbound,
		XPath:         p.XMLXPath,
		Namespaces:    p.XMLNamespaces,
		Filepaths:     p.XMLFilepaths,
		FilePattern:   p.FilePattern,
		Description:   generation.Truncate(p.Rationale, 120),
		Message:       message,
		Category:      p.Category,
		Effort:        generation.ComplexityToEffort(p.Complexity),
		Labels: rules.InitialLabels([]string{
			fmt.Sprintf("konveyor.io/source=%s", source),
			fmt.Sprintf("konveyor.io/target=%s", target),
		}),
		Links: generation.BuildLinks(p),
	}
}

// buildDeterministicMessage builds a rich migration message from extracted pattern
// fields (rationale, examples, target) without needing an LLM call.
func buildDeterministicMessage(p extraction.MigrationPattern) string {
	var b strings.Builder
	b.WriteString(p.Rationale)

	if p.TargetPattern != "" {
		fmt.Fprintf(&b, "\n\nReplace with `%s`.", p.TargetPattern)
	}

	if p.ExampleBefore != "" {
		b.WriteString("\n\nBefore:\n```\n")
		b.WriteString(p.ExampleBefore)
		b.WriteString("\n```")
	}
	if p.ExampleAfter != "" {
		b.WriteString("\n\nAfter:\n```\n")
		b.WriteString(p.ExampleAfter)
		b.WriteString("\n```")
	}

	return b.String()
}
