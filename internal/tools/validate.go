package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type validateInput struct {
	RulesPath string `json:"rules_path"`
}

// ValidateRulesHandler returns an MCP tool handler for validate_rules.
func ValidateRulesHandler() mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input validateInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if input.RulesPath == "" {
			return errorResult("rules_path is required"), nil
		}
		rulesPath := input.RulesPath

		loaded, err := loadRules(rulesPath)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to load rules: %v", err)), nil
		}

		result := rules.Validate(loaded)
		data, err := json.Marshal(result)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal result: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(data)},
			},
		}, nil
	}
}

// loadRules loads rules from a file or directory path.
func loadRules(path string) ([]rules.Rule, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return rules.ReadRulesDir(path)
	}
	return rules.ReadRulesFile(path)
}

// PlaceholderHandler returns a handler that reports the tool is not yet implemented.
func PlaceholderHandler(name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return errorResult(fmt.Sprintf("%s is not yet implemented", name)), nil
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
