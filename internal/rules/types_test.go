package rules

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRule_YAMLRoundtrip(t *testing.T) {
	rule := Rule{
		RuleID:      "test-00010",
		Description: "Test rule",
		Category:    CategoryMandatory,
		Effort:      5,
		Labels:      []string{"konveyor.io/source=java-ee", "konveyor.io/target=quarkus"},
		Message:     "Replace deprecated API",
		Links: []Link{
			{URL: "https://example.com", Title: "Migration Guide"},
		},
		When: NewJavaReferenced("javax.ejb.Stateless", LocationAnnotation),
		CustomVariables: []CustomVariable{
			{Pattern: "javax\\.ejb\\.(?P<name>\\w+)", Name: "name"},
		},
	}

	data, err := yaml.Marshal([]Rule{rule})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got []Rule
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	r := got[0]
	if r.RuleID != rule.RuleID {
		t.Errorf("ruleID: got %q, want %q", r.RuleID, rule.RuleID)
	}
	if r.Category != rule.Category {
		t.Errorf("category: got %q, want %q", r.Category, rule.Category)
	}
	if r.When.JavaReferenced == nil {
		t.Fatal("when.java.referenced is nil after roundtrip")
	}
	if r.When.JavaReferenced.Pattern != "javax.ejb.Stateless" {
		t.Errorf("pattern: got %q, want %q", r.When.JavaReferenced.Pattern, "javax.ejb.Stateless")
	}
	if r.When.JavaReferenced.Location != LocationAnnotation {
		t.Errorf("location: got %q, want %q", r.When.JavaReferenced.Location, LocationAnnotation)
	}
}

func TestRule_TagOnly(t *testing.T) {
	rule := Rule{
		RuleID: "tag-only-00010",
		Tag:    []string{"EJB"},
		When:   NewJavaReferenced("javax.ejb.*", ""),
	}

	data, err := yaml.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Rule
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Message != "" {
		t.Errorf("expected no message, got %q", got.Message)
	}
	if len(got.Tag) != 1 || got.Tag[0] != "EJB" {
		t.Errorf("tag: got %v, want [EJB]", got.Tag)
	}
}

func TestCondition_OrCombinator_YAML(t *testing.T) {
	cond := NewOr(
		NewJavaDependency("javax.jms:javax.jms-api", "0.0.0", ""),
		NewJavaDependency("jakarta.jms:jakarta.jms-api", "0.0.0", ""),
	)

	data, err := yaml.Marshal(cond)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Condition
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Or) != 2 {
		t.Fatalf("expected 2 or entries, got %d", len(got.Or))
	}
	if got.Or[0].JavaDependency == nil {
		t.Error("or[0].java.dependency is nil")
	}
}

func TestCondition_BuiltinFilecontent_YAML(t *testing.T) {
	cond := NewBuiltinFilecontent("spring\\.datasource", `application.*\.(properties|yml|yaml)`)

	data, err := yaml.Marshal(cond)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Condition
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.BuiltinFilecontent == nil {
		t.Fatal("builtin.filecontent is nil")
	}
	if got.BuiltinFilecontent.Pattern != "spring\\.datasource" {
		t.Errorf("pattern: got %q", got.BuiltinFilecontent.Pattern)
	}
	if got.BuiltinFilecontent.FilePattern != `application.*\.(properties|yml|yaml)` {
		t.Errorf("filePattern: got %q", got.BuiltinFilecontent.FilePattern)
	}
}

func TestRuleset_YAMLRoundtrip(t *testing.T) {
	rs := Ruleset{
		Name:        "quarkus/springboot",
		Description: "Rules for migrating from Spring Boot to Quarkus",
		Labels:      []string{"konveyor.io/target=quarkus"},
	}

	data, err := yaml.Marshal(rs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Ruleset
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != rs.Name {
		t.Errorf("name: got %q, want %q", got.Name, rs.Name)
	}
}
