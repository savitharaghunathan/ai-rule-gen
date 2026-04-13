package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// callTool invokes a ToolHandler with raw JSON arguments and returns the text
// of the first content item. It fails the test if the handler returns an error.
func callTool(t *testing.T, handler mcp.ToolHandler, argsJSON string) (text string, isError bool) {
	t.Helper()
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(argsJSON),
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned unexpected Go error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("handler returned empty content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] is not *mcp.TextContent")
	}
	return tc.Text, result.IsError
}

// ---------- buildConditionFromInput ----------

func TestBuildConditionFromInput_JavaReferenced(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "java.referenced",
		Pattern:       "javax.ejb.Stateless",
		Location:      "ANNOTATION",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.JavaReferenced == nil {
		t.Fatal("expected java.referenced condition")
	}
	if c.JavaReferenced.Pattern != "javax.ejb.Stateless" {
		t.Errorf("pattern = %q, want %q", c.JavaReferenced.Pattern, "javax.ejb.Stateless")
	}
	if c.JavaReferenced.Location != "ANNOTATION" {
		t.Errorf("location = %q, want %q", c.JavaReferenced.Location, "ANNOTATION")
	}
}

func TestBuildConditionFromInput_JavaReferenced_MissingPattern(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "java.referenced"})
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}

func TestBuildConditionFromInput_JavaDependency(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "java.dependency",
		Name:          "org.springframework.boot.spring-boot-starter-parent",
		Lowerbound:    "2.0.0",
		Upperbound:    "3.0.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.JavaDependency == nil {
		t.Fatal("expected java.dependency condition")
	}
	if c.JavaDependency.Name != "org.springframework.boot.spring-boot-starter-parent" {
		t.Errorf("name = %q", c.JavaDependency.Name)
	}
}

func TestBuildConditionFromInput_JavaDependency_NameRegex(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "java.dependency",
		NameRegex:     "org\\.springframework.*",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.JavaDependency == nil {
		t.Fatal("expected java.dependency condition")
	}
	if c.JavaDependency.NameRegex != "org\\.springframework.*" {
		t.Errorf("nameRegex = %q", c.JavaDependency.NameRegex)
	}
}

func TestBuildConditionFromInput_JavaDependency_MissingNameAndRegex(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "java.dependency"})
	if err == nil {
		t.Error("expected error when both name and nameRegex are empty")
	}
}

func TestBuildConditionFromInput_GoReferenced(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "go.referenced",
		Pattern:       "golang.org/x/crypto/md4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.GoReferenced == nil {
		t.Fatal("expected go.referenced condition")
	}
	if c.GoReferenced.Pattern != "golang.org/x/crypto/md4" {
		t.Errorf("pattern = %q", c.GoReferenced.Pattern)
	}
}

func TestBuildConditionFromInput_GoReferenced_MissingPattern(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "go.referenced"})
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}

func TestBuildConditionFromInput_GoDependency(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "go.dependency",
		Name:          "github.com/gin-gonic/gin",
		Upperbound:    "2.0.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.GoDependency == nil {
		t.Fatal("expected go.dependency condition")
	}
}

func TestBuildConditionFromInput_GoDependency_NameRegex(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "go.dependency",
		NameRegex:     "github\\.com/gin.*",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.GoDependency == nil || c.GoDependency.NameRegex != "github\\.com/gin.*" {
		t.Errorf("nameRegex not set correctly")
	}
}

func TestBuildConditionFromInput_NodejsReferenced(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "nodejs.referenced",
		Pattern:       "express",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.NodejsReferenced == nil || c.NodejsReferenced.Pattern != "express" {
		t.Error("expected nodejs.referenced condition with pattern 'express'")
	}
}

func TestBuildConditionFromInput_NodejsReferenced_MissingPattern(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "nodejs.referenced"})
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}

func TestBuildConditionFromInput_CSharpReferenced(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "csharp.referenced",
		Pattern:       "System.Web.HttpContext",
		Location:      "CLASS",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CSharpReferenced == nil {
		t.Fatal("expected csharp.referenced condition")
	}
	if c.CSharpReferenced.Pattern != "System.Web.HttpContext" {
		t.Errorf("pattern = %q", c.CSharpReferenced.Pattern)
	}
}

func TestBuildConditionFromInput_BuiltinFilecontent(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "builtin.filecontent",
		Pattern:       "spring\\.datasource",
		FilePattern:   "application.*\\.properties",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BuiltinFilecontent == nil {
		t.Fatal("expected builtin.filecontent condition")
	}
	if c.BuiltinFilecontent.FilePattern != "application.*\\.properties" {
		t.Errorf("filePattern = %q", c.BuiltinFilecontent.FilePattern)
	}
}

func TestBuildConditionFromInput_BuiltinFile(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "builtin.file",
		Pattern:       "Dockerfile",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BuiltinFile == nil || c.BuiltinFile.Pattern != "Dockerfile" {
		t.Error("expected builtin.file condition with pattern 'Dockerfile'")
	}
}

func TestBuildConditionFromInput_BuiltinXML(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "builtin.xml",
		XPath:         "//dependencies/dependency/groupId",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BuiltinXML == nil {
		t.Fatal("expected builtin.xml condition")
	}
}

func TestBuildConditionFromInput_BuiltinXML_MissingXPath(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "builtin.xml"})
	if err == nil {
		t.Error("expected error for missing xpath")
	}
}

func TestBuildConditionFromInput_BuiltinJSON(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "builtin.json",
		XPath:         "//dependencies/express",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BuiltinJSON == nil {
		t.Fatal("expected builtin.json condition")
	}
}

func TestBuildConditionFromInput_BuiltinHasTags(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "builtin.hasTags",
		Tags:          []string{"konveyor.io/source=spring-boot-3"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.BuiltinHasTags) == 0 {
		t.Fatal("expected builtin.hasTags condition")
	}
}

func TestBuildConditionFromInput_BuiltinHasTags_Empty(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "builtin.hasTags"})
	if err == nil {
		t.Error("expected error for empty tags")
	}
}

func TestBuildConditionFromInput_BuiltinXMLPublicID(t *testing.T) {
	c, err := buildConditionFromInput(constructRuleInput{
		ConditionType: "builtin.xmlPublicID",
		Regex:         "-//Sun Microsystems.*//DTD.*",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BuiltinXMLPublicID == nil {
		t.Fatal("expected builtin.xmlPublicID condition")
	}
}

func TestBuildConditionFromInput_Unknown(t *testing.T) {
	_, err := buildConditionFromInput(constructRuleInput{ConditionType: "fake.condition"})
	if err == nil {
		t.Error("expected error for unknown condition type")
	}
	if !strings.Contains(err.Error(), "fake.condition") {
		t.Errorf("error should mention unknown type, got: %v", err)
	}
}

// ---------- getHelpContent ----------

func TestGetHelpContent_AllTopics(t *testing.T) {
	topics := []string{"condition_types", "locations", "labels", "categories", "rule_format", "ruleset_format", "examples"}
	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			content, err := getHelpContent(topic)
			if err != nil {
				t.Fatalf("topic %q: unexpected error: %v", topic, err)
			}
			if content == "" {
				t.Errorf("topic %q: got empty content", topic)
			}
		})
	}
}

func TestGetHelpContent_All(t *testing.T) {
	content, err := getHelpContent("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "all" should be a superset of individual topics
	if !strings.Contains(content, "Supported Condition Types") {
		t.Error("'all' content missing condition types section")
	}
	if !strings.Contains(content, "Valid Locations") {
		t.Error("'all' content missing locations section")
	}
}

func TestGetHelpContent_Unknown(t *testing.T) {
	_, err := getHelpContent("nonexistent-topic")
	if err == nil {
		t.Error("expected error for unknown topic")
	}
	if !strings.Contains(err.Error(), "nonexistent-topic") {
		t.Errorf("error should mention the unknown topic, got: %v", err)
	}
}

// ---------- loadRules ----------

func TestLoadRules_File(t *testing.T) {
	dir := t.TempDir()
	ruleList := []rules.Rule{{
		RuleID:   "test-00010",
		Category: "mandatory",
		Effort:   3,
		Labels:   []string{"konveyor.io/source=a", "konveyor.io/target=b"},
		Message:  "update code",
		When:     rules.NewGoReferenced("golang.org/x/crypto/md4"),
	}}
	data, _ := yaml.Marshal(ruleList)
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, data, 0o644)

	loaded, err := loadRules(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Errorf("got %d rules, want 1", len(loaded))
	}
}

func TestLoadRules_Directory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		ruleList := []rules.Rule{{
			RuleID:   "rule-" + name,
			Category: "mandatory",
			Effort:   1,
			Labels:   []string{"konveyor.io/source=a", "konveyor.io/target=b"},
			Message:  "msg",
			When:     rules.NewGoReferenced("golang.org/x/crypto/md4"),
		}}
		data, _ := yaml.Marshal(ruleList)
		os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}

	loaded, err := loadRules(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("got %d rules, want 2", len(loaded))
	}
}

func TestLoadRules_NotFound(t *testing.T) {
	_, err := loadRules("/tmp/nonexistent-path-for-test")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

// ---------- ValidateRulesHandler ----------

func TestValidateRulesHandler_ValidFile(t *testing.T) {
	dir := t.TempDir()
	ruleList := []rules.Rule{{
		RuleID:   "test-00010",
		Category: "mandatory",
		Effort:   3,
		Labels:   []string{"konveyor.io/source=a", "konveyor.io/target=b"},
		Message:  "update code",
		When:     rules.NewGoReferenced("golang.org/x/crypto/md4"),
	}}
	data, _ := yaml.Marshal(ruleList)
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, data, 0o644)

	handler := ValidateRulesHandler()
	text, isError := callTool(t, handler, `{"rules_path":"`+path+`"}`)
	if isError {
		t.Errorf("expected success, got error: %s", text)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("cannot parse result: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid=true, got false. Response: %s", text)
	}
}

func TestValidateRulesHandler_InvalidRules(t *testing.T) {
	dir := t.TempDir()
	invalidRules := []rules.Rule{{RuleID: "", Category: "bad-category"}}
	data, _ := yaml.Marshal(invalidRules)
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, data, 0o644)

	handler := ValidateRulesHandler()
	text, isError := callTool(t, handler, `{"rules_path":"`+path+`"}`)
	if isError {
		t.Errorf("handler itself should not error; got: %s", text)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("cannot parse result: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false for invalid rules")
	}
}

func TestValidateRulesHandler_MissingPath(t *testing.T) {
	handler := ValidateRulesHandler()
	_, isError := callTool(t, handler, `{}`)
	if !isError {
		t.Error("expected error result for missing rules_path")
	}
}

func TestValidateRulesHandler_NonexistentPath(t *testing.T) {
	handler := ValidateRulesHandler()
	_, isError := callTool(t, handler, `{"rules_path":"/tmp/no-such-path-for-test"}`)
	if !isError {
		t.Error("expected error result for nonexistent path")
	}
}

// ---------- ConstructRuleHandler ----------

func TestConstructRuleHandler_JavaReferenced(t *testing.T) {
	handler := ConstructRuleHandler()
	args := `{
		"ruleID": "spring-boot-00010",
		"condition_type": "java.referenced",
		"pattern": "javax.ejb.Stateless",
		"location": "ANNOTATION",
		"message": "Replace javax.ejb.Stateless with CDI bean",
		"category": "mandatory",
		"effort": 3,
		"labels": ["konveyor.io/source=java-ee", "konveyor.io/target=quarkus"]
	}`
	text, isError := callTool(t, handler, args)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}

	var out struct {
		YAML  string `json:"yaml"`
		Valid bool   `json:"valid"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("cannot parse result: %v", err)
	}
	if !out.Valid {
		t.Errorf("expected valid=true, got false. Response: %s", text)
	}
	if !strings.Contains(out.YAML, "javax.ejb.Stateless") {
		t.Errorf("YAML should contain the pattern, got: %s", out.YAML)
	}
}

func TestConstructRuleHandler_GoReferenced(t *testing.T) {
	handler := ConstructRuleHandler()
	args := `{
		"ruleID": "fips-00010",
		"condition_type": "go.referenced",
		"pattern": "golang.org/x/crypto/md4",
		"message": "Replace golang.org/x/crypto/md4 with a FIPS-compliant alternative",
		"category": "mandatory",
		"effort": 3,
		"labels": ["konveyor.io/source=go-non-fips", "konveyor.io/target=go-fips"]
	}`
	text, isError := callTool(t, handler, args)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}

	var out struct {
		Valid bool `json:"valid"`
	}
	json.Unmarshal([]byte(text), &out)
	if !out.Valid {
		t.Errorf("expected valid=true. Response: %s", text)
	}
}

func TestConstructRuleHandler_MissingRequired(t *testing.T) {
	handler := ConstructRuleHandler()
	// Missing ruleID
	_, isError := callTool(t, handler, `{"message":"m","category":"mandatory","effort":3,"condition_type":"go.referenced","pattern":"x"}`)
	if !isError {
		t.Error("expected error for missing ruleID")
	}
}

func TestConstructRuleHandler_InvalidCondition(t *testing.T) {
	handler := ConstructRuleHandler()
	args := `{
		"ruleID": "test-00010",
		"condition_type": "java.referenced",
		"message": "msg",
		"category": "mandatory",
		"effort": 3
	}`
	// pattern is required for java.referenced but not provided
	text, isError := callTool(t, handler, args)
	// Should return a non-error result with valid=false (handler catches it)
	_ = isError
	var out struct {
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("cannot parse result: %v", err)
	}
	if out.Valid {
		t.Error("expected valid=false for missing pattern")
	}
}

func TestConstructRuleHandler_InvalidJSON(t *testing.T) {
	handler := ConstructRuleHandler()
	_, isError := callTool(t, handler, `{not valid json`)
	if !isError {
		t.Error("expected error for invalid JSON input")
	}
}

// ---------- ConstructRulesetHandler ----------

func TestConstructRulesetHandler_Valid(t *testing.T) {
	handler := ConstructRulesetHandler()
	args := `{
		"name": "spring-boot-4-migration",
		"description": "Rules for migrating Spring Boot 3 to 4",
		"labels": ["konveyor.io/source=spring-boot-3", "konveyor.io/target=spring-boot-4"]
	}`
	text, isError := callTool(t, handler, args)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}

	var out struct {
		YAML  string `json:"yaml"`
		Valid bool   `json:"valid"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("cannot parse result: %v", err)
	}
	if !out.Valid {
		t.Errorf("expected valid=true. Response: %s", text)
	}
	if !strings.Contains(out.YAML, "spring-boot-4-migration") {
		t.Errorf("YAML should contain ruleset name, got: %s", out.YAML)
	}
}

func TestConstructRulesetHandler_MissingName(t *testing.T) {
	handler := ConstructRulesetHandler()
	_, isError := callTool(t, handler, `{"description":"desc"}`)
	if !isError {
		t.Error("expected error for missing name")
	}
}

// ---------- GetHelpHandler ----------

func TestGetHelpHandler_ValidTopic(t *testing.T) {
	handler := GetHelpHandler()
	text, isError := callTool(t, handler, `{"topic":"condition_types"}`)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}

	var out struct {
		Topic   string `json:"topic"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("cannot parse result: %v", err)
	}
	if out.Topic != "condition_types" {
		t.Errorf("topic = %q, want %q", out.Topic, "condition_types")
	}
	if out.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestGetHelpHandler_EmptyTopicDefaultsToAll(t *testing.T) {
	handler := GetHelpHandler()
	text, isError := callTool(t, handler, `{}`)
	if isError {
		t.Fatalf("expected success, got error: %s", text)
	}

	var out struct {
		Topic string `json:"topic"`
	}
	json.Unmarshal([]byte(text), &out)
	if out.Topic != "all" {
		t.Errorf("topic = %q, want %q", out.Topic, "all")
	}
}

func TestGetHelpHandler_InvalidTopic(t *testing.T) {
	handler := GetHelpHandler()
	_, isError := callTool(t, handler, `{"topic":"nonexistent"}`)
	if !isError {
		t.Error("expected error for unknown topic")
	}
}

// ---------- PlaceholderHandler ----------

func TestPlaceholderHandler(t *testing.T) {
	handler := PlaceholderHandler("my_tool")
	_, isError := callTool(t, handler, `{}`)
	if !isError {
		t.Error("expected placeholder to return an error result")
	}
}

// ---------- errorResult ----------

func TestErrorResult(t *testing.T) {
	result := errorResult("something went wrong")
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "something went wrong" {
		t.Errorf("text = %q, want %q", tc.Text, "something went wrong")
	}
}
