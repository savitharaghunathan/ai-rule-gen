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

// ---------- buildSingleCondition — all condition type branches ----------

func TestBuildSingleCondition_GoDependency(t *testing.T) {
	p := extraction.MigrationPattern{
		ConditionType:  "go.dependency",
		DependencyName: "github.com/gin-gonic/gin",
		DepLowerbound:  "1.0.0",
		DepUpperbound:  "2.0.0",
	}
	c := buildSingleCondition(p)
	if c.GoDependency == nil {
		t.Fatal("expected go.dependency condition")
	}
	if c.GoDependency.Name != "github.com/gin-gonic/gin" {
		t.Errorf("name = %q", c.GoDependency.Name)
	}
}

func TestBuildSingleCondition_NodejsReferenced(t *testing.T) {
	p := extraction.MigrationPattern{
		ConditionType: "nodejs.referenced",
		SourceFQN:     "express",
	}
	c := buildSingleCondition(p)
	if c.NodejsReferenced == nil || c.NodejsReferenced.Pattern != "express" {
		t.Error("expected nodejs.referenced with pattern 'express'")
	}
}

func TestBuildSingleCondition_CSharpReferenced(t *testing.T) {
	p := extraction.MigrationPattern{
		ConditionType: "csharp.referenced",
		SourceFQN:     "System.Web.HttpContext",
		LocationType:  "CLASS",
	}
	c := buildSingleCondition(p)
	if c.CSharpReferenced == nil {
		t.Fatal("expected csharp.referenced condition")
	}
	if c.CSharpReferenced.Pattern != "System.Web.HttpContext" {
		t.Errorf("pattern = %q", c.CSharpReferenced.Pattern)
	}
}

func TestBuildSingleCondition_BuiltinFile(t *testing.T) {
	p := extraction.MigrationPattern{
		ConditionType: "builtin.file",
		SourceFQN:     "Dockerfile",
	}
	c := buildSingleCondition(p)
	if c.BuiltinFile == nil || c.BuiltinFile.Pattern != "Dockerfile" {
		t.Error("expected builtin.file condition with pattern 'Dockerfile'")
	}
}

func TestBuildSingleCondition_BuiltinXML(t *testing.T) {
	p := extraction.MigrationPattern{
		ConditionType: "builtin.xml",
		SourceFQN:     "//dependencies/dependency/groupId",
	}
	c := buildSingleCondition(p)
	if c.BuiltinXML == nil {
		t.Fatal("expected builtin.xml condition")
	}
}

func TestBuildSingleCondition_BuiltinJSON(t *testing.T) {
	p := extraction.MigrationPattern{
		ConditionType: "builtin.json",
		SourceFQN:     "//dependencies/express",
	}
	c := buildSingleCondition(p)
	if c.BuiltinJSON == nil {
		t.Fatal("expected builtin.json condition")
	}
}

// Note: builtin.hasTags and builtin.xmlPublicID are not handled by buildSingleCondition
// (the LLM extraction pipeline doesn't emit them). They fall through to the default
// case which produces builtin.filecontent. They are only constructable via the MCP
// tool's buildConditionFromInput, which is tested in internal/tools.

func TestBuildSingleCondition_DefaultFallback_WithLocation(t *testing.T) {
	// Unknown condition type with a location → falls back to java.referenced
	p := extraction.MigrationPattern{
		ConditionType: "unknown.type",
		SourceFQN:     "some.Class",
		LocationType:  "IMPORT",
	}
	c := buildSingleCondition(p)
	if c.JavaReferenced == nil {
		t.Error("expected java.referenced fallback when location is set")
	}
}

func TestBuildSingleCondition_DefaultFallback_NoLocation(t *testing.T) {
	// Unknown condition type without location → falls back to builtin.filecontent
	p := extraction.MigrationPattern{
		ConditionType: "unknown.type",
		SourcePattern: "some-pattern",
	}
	c := buildSingleCondition(p)
	if c.BuiltinFilecontent == nil {
		t.Error("expected builtin.filecontent fallback when no location is set")
	}
}

func TestBuildSingleCondition_FallsBackToSourcePattern(t *testing.T) {
	// When SourceFQN is empty, SourcePattern is used as the pattern.
	p := extraction.MigrationPattern{
		ConditionType: "go.referenced",
		SourceFQN:     "",
		SourcePattern: "golang.org/x/crypto/md4",
	}
	c := buildSingleCondition(p)
	if c.GoReferenced == nil || c.GoReferenced.Pattern != "golang.org/x/crypto/md4" {
		t.Errorf("expected pattern from SourcePattern, got: %+v", c.GoReferenced)
	}
}

// ---------- ensureJavaPatternMatchable ----------

func TestEnsureJavaPatternMatchable_PackageLevel(t *testing.T) {
	// All-lowercase package prefix → should get wildcard appended
	tests := []struct {
		input, want string
	}{
		{"javax.ejb", "javax.ejb*"},
		{"javax.xml.bind", "javax.xml.bind*"},
		{"com.example.service", "com.example.service*"},
	}
	for _, tt := range tests {
		got := ensureJavaPatternMatchable(tt.input)
		if got != tt.want {
			t.Errorf("ensureJavaPatternMatchable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEnsureJavaPatternMatchable_ClassLevel(t *testing.T) {
	// Has an uppercase segment → leave as-is
	tests := []string{
		"javax.ejb.Stateless",
		"org.springframework.web.bind.annotation.RequestMapping",
		"javax.xml.bind.JAXBContext",
	}
	for _, input := range tests {
		got := ensureJavaPatternMatchable(input)
		if got != input {
			t.Errorf("ensureJavaPatternMatchable(%q) = %q, want unchanged", input, got)
		}
	}
}

func TestEnsureJavaPatternMatchable_WildcardAlreadyPresent(t *testing.T) {
	input := "javax.ejb*"
	got := ensureJavaPatternMatchable(input)
	if got != input {
		t.Errorf("ensureJavaPatternMatchable(%q) = %q, want unchanged", input, got)
	}
}

func TestEnsureJavaPatternMatchable_Empty(t *testing.T) {
	got := ensureJavaPatternMatchable("")
	if got != "" {
		t.Errorf("ensureJavaPatternMatchable(%q) = %q, want empty", "", got)
	}
}

func TestEnsureJavaPatternMatchable_MethodSignature(t *testing.T) {
	// Contains parens → leave as-is
	input := "javax.ejb.Stateless.create()"
	got := ensureJavaPatternMatchable(input)
	if got != input {
		t.Errorf("ensureJavaPatternMatchable(%q) = %q, want unchanged", input, got)
	}
}

// ---------- truncate ----------

func TestTruncate_ShortString(t *testing.T) {
	got := truncate("hello", 10)
	if got != "hello" {
		t.Errorf("truncate(%q, 10) = %q, want %q", "hello", got, "hello")
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	got := truncate("hello", 5)
	if got != "hello" {
		t.Errorf("truncate(%q, 5) = %q, want %q", "hello", got, "hello")
	}
}

func TestTruncate_LongString(t *testing.T) {
	got := truncate("hello world", 8)
	if got != "hello..." {
		t.Errorf("truncate(%q, 8) = %q, want %q", "hello world", got, "hello...")
	}
}

// ---------- buildLinks ----------

func TestBuildLinks_WithURL(t *testing.T) {
	p := extraction.MigrationPattern{DocumentationURL: "https://spring.io/migration"}
	links := buildLinks(p)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].URL != "https://spring.io/migration" {
		t.Errorf("URL = %q", links[0].URL)
	}
}

func TestBuildLinks_NoURL(t *testing.T) {
	p := extraction.MigrationPattern{}
	links := buildLinks(p)
	if links != nil {
		t.Errorf("expected nil links for empty URL, got %v", links)
	}
}
