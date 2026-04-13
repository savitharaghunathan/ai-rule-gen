package server

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
}

func TestNew_DoesNotPanic(t *testing.T) {
	// Verify New does not panic when given nil handlers.
	// The returned *mcp.Server is opaque; we just confirm it is non-nil.
	s := New(ToolHandlers{})
	if s == nil {
		t.Error("expected non-nil server")
	}
}

func TestConstructRuleTool_Fields(t *testing.T) {
	tool := constructRuleTool()
	if tool.Name != "construct_rule" {
		t.Errorf("Name = %q, want %q", tool.Name, "construct_rule")
	}
	if tool.Description == "" {
		t.Error("Description should not be empty")
	}
	schema, ok := tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatal("InputSchema is not map[string]any")
	}
	props, _ := schema["properties"].(map[string]any)
	for _, required := range []string{"ruleID", "condition_type", "message", "category", "effort"} {
		if _, found := props[required]; !found {
			t.Errorf("InputSchema missing property %q", required)
		}
	}
}

func TestConstructRulesetTool_Fields(t *testing.T) {
	tool := constructRulesetTool()
	if tool.Name != "construct_ruleset" {
		t.Errorf("Name = %q, want %q", tool.Name, "construct_ruleset")
	}
	if tool.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestValidateRulesTool_Fields(t *testing.T) {
	tool := validateRulesTool()
	if tool.Name != "validate_rules" {
		t.Errorf("Name = %q, want %q", tool.Name, "validate_rules")
	}
}

func TestGetHelpTool_Fields(t *testing.T) {
	tool := getHelpTool()
	if tool.Name != "get_help" {
		t.Errorf("Name = %q, want %q", tool.Name, "get_help")
	}
}
