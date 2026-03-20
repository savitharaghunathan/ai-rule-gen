package generation

import (
	"context"
	"testing"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/extraction"
)

type mockCompleter struct {
	responses []string
	calls     []string
	index     int
}

func (m *mockCompleter) Complete(_ context.Context, prompt string) (string, error) {
	m.calls = append(m.calls, prompt)
	if m.index >= len(m.responses) {
		return "Generated migration message.", nil
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func TestIDGenerator(t *testing.T) {
	gen := NewIDGenerator("java-ee-to-quarkus")

	expected := []string{
		"java-ee-to-quarkus-00010",
		"java-ee-to-quarkus-00020",
		"java-ee-to-quarkus-00030",
	}

	for _, want := range expected {
		got := gen.Next()
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestComplexityToEffort(t *testing.T) {
	tests := []struct {
		complexity string
		want       int
	}{
		{"trivial", 1},
		{"low", 3},
		{"medium", 5},
		{"high", 7},
		{"expert", 9},
		{"MEDIUM", 5},  // case insensitive
		{"unknown", 5}, // default
	}

	for _, tt := range tests {
		got := complexityToEffort(tt.complexity)
		if got != tt.want {
			t.Errorf("complexityToEffort(%q): got %d, want %d", tt.complexity, got, tt.want)
		}
		if got < 1 || got > 10 {
			t.Errorf("complexityToEffort(%q): %d outside range 1-10", tt.complexity, got)
		}
	}
}

func TestGenerator_JavaReferenced(t *testing.T) {
	mock := &mockCompleter{}
	tmpl := template.Must(template.New("msg").Parse("migrate {{.SourcePattern}}"))

	gen := New(mock, tmpl)
	patterns := []extraction.MigrationPattern{{
		SourcePattern: "@Stateless",
		SourceFQN:     "javax.ejb.Stateless",
		LocationType:  "ANNOTATION",
		Rationale:     "EJBs are not supported",
		Complexity:    "medium",
		Category:      "mandatory",
		ProviderType:  "java",
		Concern:       "ejb",
	}}

	grouped, ruleset, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "java-ee", Target: "quarkus", Language: "java",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ruleset.Name != "quarkus/java-ee" {
		t.Errorf("ruleset name: got %q", ruleset.Name)
	}

	ejbRules := grouped["ejb"]
	if len(ejbRules) != 1 {
		t.Fatalf("expected 1 ejb rule, got %d", len(ejbRules))
	}

	r := ejbRules[0]
	if r.When.JavaReferenced == nil {
		t.Fatal("expected java.referenced condition")
	}
	if r.When.JavaReferenced.Pattern != "javax.ejb.Stateless" {
		t.Errorf("pattern: got %q", r.When.JavaReferenced.Pattern)
	}
	if r.When.JavaReferenced.Location != "ANNOTATION" {
		t.Errorf("location: got %q", r.When.JavaReferenced.Location)
	}
	if r.Effort != 5 {
		t.Errorf("effort: got %d, want 5", r.Effort)
	}
}

func TestGenerator_BuiltinFilecontent(t *testing.T) {
	mock := &mockCompleter{}
	gen := New(mock, nil)

	patterns := []extraction.MigrationPattern{{
		SourcePattern: "spring.datasource",
		Rationale:     "Spring datasource config not supported",
		Complexity:    "low",
		Category:      "mandatory",
		ProviderType:  "builtin",
		FilePattern:   `application.*\.properties`,
	}}

	grouped, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "springboot", Target: "quarkus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rules := grouped["general"]
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].When.BuiltinFilecontent == nil {
		t.Fatal("expected builtin.filecontent condition")
	}
}

func TestGenerator_OrCombinator(t *testing.T) {
	mock := &mockCompleter{}
	gen := New(mock, nil)

	patterns := []extraction.MigrationPattern{{
		SourcePattern:   "JMS API",
		SourceFQN:       "javax.jms.MessageListener",
		AlternativeFQNs: []string{"jakarta.jms.MessageListener"},
		Rationale:       "JMS to reactive messaging",
		Complexity:      "high",
		Category:        "mandatory",
		ProviderType:    "java",
		LocationType:    "IMPLEMENTS_TYPE",
		Concern:         "messaging",
	}}

	grouped, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "java-ee", Target: "quarkus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rules := grouped["messaging"]
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if len(rules[0].When.Or) != 2 {
		t.Errorf("expected or with 2 conditions, got %d", len(rules[0].When.Or))
	}
}

func TestGenerator_ConcernGrouping(t *testing.T) {
	mock := &mockCompleter{}
	gen := New(mock, nil)

	patterns := []extraction.MigrationPattern{
		{SourcePattern: "a", SourceFQN: "a", Rationale: "r", Complexity: "low", Category: "mandatory", ProviderType: "java", Concern: "security"},
		{SourcePattern: "b", SourceFQN: "b", Rationale: "r", Complexity: "low", Category: "mandatory", ProviderType: "java", Concern: "security"},
		{SourcePattern: "c", SourceFQN: "c", Rationale: "r", Complexity: "low", Category: "mandatory", ProviderType: "java", Concern: "web"},
	}

	grouped, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "java-ee", Target: "quarkus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(grouped["security"]) != 2 {
		t.Errorf("security: expected 2, got %d", len(grouped["security"]))
	}
	if len(grouped["web"]) != 1 {
		t.Errorf("web: expected 1, got %d", len(grouped["web"]))
	}
}

func TestRulePrefix(t *testing.T) {
	tests := []struct {
		source, target, want string
	}{
		{"java-ee", "quarkus", "java-ee-to-quarkus"},
		{"Spring Boot", "Quarkus", "spring-boot-to-quarkus"},
	}
	for _, tt := range tests {
		got := rulePrefix(tt.source, tt.target)
		if got != tt.want {
			t.Errorf("rulePrefix(%q, %q): got %q, want %q", tt.source, tt.target, got, tt.want)
		}
	}
}
