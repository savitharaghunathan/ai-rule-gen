package contract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contract.json")
	content := `{
  "name": "demo-skill",
  "version": "1.0.0",
  "inputs": [
    {"name":"guide","type":"string","required":true},
    {"name":"sections","type":"array","required":false,"items_type":"object"}
  ],
  "returns": [
    {"name":"patterns_count","type":"number","required":true}
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if c.Name != "demo-skill" {
		t.Fatalf("Name = %q, want demo-skill", c.Name)
	}
}

func TestValidatePayload(t *testing.T) {
	fields := []Field{
		{Name: "guide", Type: "string", Required: true},
		{Name: "max_iterations", Type: "number", Required: false},
		{Name: "mode", Type: "string", Required: false, Enum: []string{"full", "chunk"}},
		{Name: "sections", Type: "array", Required: false, ItemsType: "object"},
		{Name: "result_types", Type: "array", Required: false, ItemsType: "string", Enum: []string{"fixed", "timeout"}},
	}

	okPayload := map[string]any{
		"guide":          "output/guide.md",
		"max_iterations": 2,
		"mode":           "full",
		"sections":       []any{map[string]any{"heading": "A"}},
		"result_types":   []any{"fixed"},
	}
	if errs := ValidatePayload(fields, okPayload, true); len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}

	badPayload := map[string]any{
		"mode":     "invalid",
		"sections": []any{"not-object"},
		"result_types": []any{
			"unknown",
		},
		"extra":    true,
	}
	errs := ValidatePayload(fields, badPayload, true)
	if len(errs) != 5 {
		t.Fatalf("got %d errors, want 5: %+v", len(errs), errs)
	}
}

func TestContractValidate_DuplicateField(t *testing.T) {
	c := SkillContract{
		Name:    "dup",
		Version: "1.0.0",
		Inputs: []Field{
			{Name: "guide", Type: "string", Required: true},
			{Name: "guide", Type: "string", Required: false},
		},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected duplicate field validation error")
	}
}
