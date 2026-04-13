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

	ruleList, ruleset, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "java-ee", Target: "quarkus", Language: "java",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ruleset.Name != "quarkus/java-ee" {
		t.Errorf("ruleset name: got %q", ruleset.Name)
	}

	if len(ruleList) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleList))
	}

	r := ruleList[0]
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

	ruleList, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "springboot", Target: "quarkus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ruleList) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleList))
	}
	if ruleList[0].When.BuiltinFilecontent == nil {
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

	ruleList, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "java-ee", Target: "quarkus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ruleList) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleList))
	}
	if len(ruleList[0].When.Or) != 2 {
		t.Errorf("expected or with 2 conditions, got %d", len(ruleList[0].When.Or))
	}
}

func TestGenerator_MultiplePatterns(t *testing.T) {
	mock := &mockCompleter{}
	gen := New(mock, nil)

	patterns := []extraction.MigrationPattern{
		{SourcePattern: "a", SourceFQN: "a", Rationale: "r", Complexity: "low", Category: "mandatory", ProviderType: "java", Concern: "security"},
		{SourcePattern: "b", SourceFQN: "b", Rationale: "r", Complexity: "low", Category: "mandatory", ProviderType: "java", Concern: "security"},
		{SourcePattern: "c", SourceFQN: "c", Rationale: "r", Complexity: "low", Category: "mandatory", ProviderType: "java", Concern: "web"},
	}

	ruleList, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "java-ee", Target: "quarkus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ruleList) != 3 {
		t.Errorf("expected 3 rules, got %d", len(ruleList))
	}
}

func TestGenerator_JavaDependency(t *testing.T) {
	mock := &mockCompleter{}
	gen := New(mock, nil)

	patterns := []extraction.MigrationPattern{{
		SourcePattern:  "spring-boot-starter-parent",
		DependencyName: "org.springframework.boot.spring-boot-starter-parent",
		Rationale:      "Upgrade to Spring Boot 3",
		Complexity:     "medium",
		Category:       "mandatory",
		ConditionType:  "java.dependency",
	}}

	ruleList, _, err := gen.Generate(context.Background(), patterns, GenerateInput{
		Source: "spring-boot-2", Target: "spring-boot-3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ruleList) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleList))
	}
	if ruleList[0].When.JavaDependency == nil {
		t.Fatal("expected java.dependency condition")
	}
	if ruleList[0].When.JavaDependency.Name != "org.springframework.boot.spring-boot-starter-parent" {
		t.Errorf("name: got %q", ruleList[0].When.JavaDependency.Name)
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
