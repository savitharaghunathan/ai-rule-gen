package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds MCP server configuration.
type Config struct {
	Host string
	Port int
}

// DefaultConfig returns the default server configuration.
func DefaultConfig() Config {
	return Config{
		Host: "localhost",
		Port: 8080,
	}
}

// ToolHandlers holds the handler functions for each MCP tool.
// Only deterministic tools are exposed via MCP (no server LLM needed).
// Pipeline tools (generate_rules, generate_test_data, run_tests, score_confidence)
// are CLI-only and call internal packages directly.
type ToolHandlers struct {
	ConstructRule    mcp.ToolHandler
	ConstructRuleset mcp.ToolHandler
	ValidateRules    mcp.ToolHandler
	GetHelp          mcp.ToolHandler
}

// New creates a new MCP server with all tools registered.
func New(handlers ToolHandlers) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ai-rule-gen",
			Version: "0.1.0",
		},
		&mcp.ServerOptions{
			Instructions: "MCP server for generating Konveyor analyzer migration rules. Provides deterministic tools for constructing, validating, and learning about Konveyor rule syntax. The MCP client's LLM reads migration guides and calls these tools to build valid rules.",
			Logger:       slog.Default(),
		},
	)

	server.AddTool(constructRuleTool(), handlers.ConstructRule)
	server.AddTool(constructRulesetTool(), handlers.ConstructRuleset)
	server.AddTool(validateRulesTool(), handlers.ValidateRules)
	server.AddTool(getHelpTool(), handlers.GetHelp)

	return server
}

// RunStdio starts the MCP server with stdio transport.
func RunStdio(ctx context.Context, s *mcp.Server) error {
	slog.Info("starting MCP server", "transport", "stdio")
	return s.Run(ctx, &mcp.StdioTransport{})
}

// ListenAndServe starts the MCP server with Streamable HTTP transport.
func ListenAndServe(cfg Config, s *mcp.Server) error {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	slog.Info("starting MCP server", "addr", addr, "transport", "streamable-http")
	return http.ListenAndServe(addr, handler)
}

func constructRuleTool() *mcp.Tool {
	return &mcp.Tool{
		Name: "construct_rule",
		Description: `Construct a single Konveyor analyzer rule. Takes all rule parameters, validates, and returns valid YAML.

Use this tool when you have identified a migration pattern and want to create a rule for it. You provide the condition type, pattern, location, message, category, effort, and labels. The tool validates everything and returns the complete rule YAML.

Workflow: 1) Use get_help to learn condition types and locations, 2) Call construct_rule for each migration pattern, 3) Call construct_ruleset to create the ruleset metadata, 4) Call validate_rules to verify the output.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ruleID": map[string]any{
					"type":        "string",
					"description": "Unique rule identifier (e.g., spring-boot-00010)",
				},
				"condition_type": map[string]any{
					"type":        "string",
					"description": "Condition type: java.referenced, java.dependency, go.referenced, go.dependency, nodejs.referenced, csharp.referenced, builtin.filecontent, builtin.file, builtin.xml, builtin.json, builtin.hasTags, builtin.xmlPublicID",
					"enum": []string{
						"java.referenced", "java.dependency", "go.referenced", "go.dependency",
						"nodejs.referenced", "csharp.referenced", "builtin.filecontent", "builtin.file",
						"builtin.xml", "builtin.json", "builtin.hasTags", "builtin.xmlPublicID",
					},
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "Match pattern (required for java.referenced, go.referenced, nodejs.referenced, csharp.referenced, builtin.filecontent, builtin.file)",
				},
				"location": map[string]any{
					"type":        "string",
					"description": "Code location for java.referenced (ANNOTATION, IMPORT, CLASS, METHOD_CALL, CONSTRUCTOR_CALL, FIELD, METHOD, INHERITANCE, IMPLEMENTS_TYPE, ENUM, RETURN_TYPE, VARIABLE_DECLARATION, TYPE, PACKAGE) or csharp.referenced (ALL, METHOD, FIELD, CLASS)",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Dependency name for java.dependency or go.dependency (e.g., groupId.artifactId or module path)",
				},
				"nameRegex": map[string]any{
					"type":        "string",
					"description": "Dependency name regex for java.dependency or go.dependency",
				},
				"lowerbound": map[string]any{
					"type":        "string",
					"description": "Version lower bound for dependency conditions",
				},
				"upperbound": map[string]any{
					"type":        "string",
					"description": "Version upper bound for dependency conditions",
				},
				"xpath": map[string]any{
					"type":        "string",
					"description": "XPath expression for builtin.xml or builtin.json",
				},
				"regex": map[string]any{
					"type":        "string",
					"description": "Regex pattern for builtin.xmlPublicID",
				},
				"filePattern": map[string]any{
					"type":        "string",
					"description": "File glob pattern for builtin.filecontent (e.g., *.properties)",
				},
				"tags": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Tags for builtin.hasTags condition",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "Migration guidance message (markdown with Before/After code examples)",
				},
				"category": map[string]any{
					"type":        "string",
					"description": "Rule category",
					"enum":        []string{"mandatory", "optional", "potential"},
				},
				"effort": map[string]any{
					"type":        "integer",
					"description": "Migration effort estimate (1-10)",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Short rule description",
				},
				"labels": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Labels in konveyor.io/<key>=<value> format (e.g., konveyor.io/source=spring-boot-3)",
				},
				"links": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"title": map[string]any{"type": "string"},
							"url":   map[string]any{"type": "string"},
						},
					},
					"description": "Documentation links",
				},
			},
			"required": []string{"ruleID", "condition_type", "message", "category", "effort"},
		},
	}
}

func constructRulesetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "construct_ruleset",
		Description: "Construct a Konveyor ruleset metadata YAML. A ruleset groups related rules and defines shared labels. Returns valid ruleset YAML.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Ruleset identifier (e.g., spring-boot-4-migration)",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Human-readable description of the ruleset",
				},
				"labels": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Labels in konveyor.io/<key>=<value> format",
				},
			},
			"required": []string{"name"},
		},
	}
}

func validateRulesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "validate_rules",
		Description: "Validate Konveyor analyzer rules for structural correctness. Checks required fields, valid categories, effort ranges, regex syntax, label format, and duplicate rule IDs.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rules_path": map[string]any{
					"type":        "string",
					"description": "Path to rules YAML file or directory",
				},
			},
			"required": []string{"rules_path"},
		},
	}
}

func getHelpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_help",
		Description: "Get documentation on Konveyor rule syntax, condition types, valid locations, label format, categories, and examples. Use this before constructing rules to understand available options.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topic": map[string]any{
					"type":        "string",
					"description": "Help topic: condition_types, locations, labels, categories, rule_format, ruleset_format, examples, all",
					"enum":        []string{"condition_types", "locations", "labels", "categories", "rule_format", "ruleset_format", "examples", "all"},
				},
			},
		},
	}
}
