package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
		return rules.NewBuiltinFilecontent(input.Pattern, input.FilePattern), nil

	case "builtin.file":
		if input.Pattern == "" {
			return rules.Condition{}, fmt.Errorf("pattern is required for builtin.file")
		}
		return rules.NewBuiltinFile(input.Pattern), nil

	case "builtin.xml":
		if input.XPath == "" {
			return rules.Condition{}, fmt.Errorf("xpath is required for builtin.xml")
		}
		return rules.NewBuiltinXML(input.XPath, nil), nil

	case "builtin.json":
		if input.XPath == "" {
			return rules.Condition{}, fmt.Errorf("xpath is required for builtin.json")
		}
		return rules.NewBuiltinJSON(input.XPath), nil

	case "builtin.hasTags":
		if len(input.Tags) == 0 {
			return rules.Condition{}, fmt.Errorf("tags is required for builtin.hasTags")
		}
		return rules.NewBuiltinHasTags(input.Tags), nil

	case "builtin.xmlPublicID":
		if input.Regex == "" {
			return rules.Condition{}, fmt.Errorf("regex is required for builtin.xmlPublicID")
		}
		return rules.NewBuiltinXMLPublicID(input.Regex, nil), nil

	default:
		valid := []string{
			"java.referenced", "java.dependency", "go.referenced", "go.dependency",
			"nodejs.referenced", "csharp.referenced", "builtin.filecontent", "builtin.file",
			"builtin.xml", "builtin.json", "builtin.hasTags", "builtin.xmlPublicID",
		}
		return rules.Condition{}, fmt.Errorf("unsupported condition_type %q. Valid types: %s", input.ConditionType, strings.Join(valid, ", "))
	}
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
