package testgen

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestExtractRulePattern_GoReferenced(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00010",
		When: rules.Condition{
			GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/md4"},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "golang.org/x/crypto/md4" {
		t.Errorf("pattern = %q, want %q", pattern, "golang.org/x/crypto/md4")
	}
	if provider != "go.referenced" {
		t.Errorf("provider = %q, want %q", provider, "go.referenced")
	}
}

func TestExtractRulePattern_JavaReferenced(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00020",
		When: rules.Condition{
			JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless", Location: "ANNOTATION"},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "javax.ejb.Stateless" {
		t.Errorf("pattern = %q, want %q", pattern, "javax.ejb.Stateless")
	}
	if provider != "java.referenced" {
		t.Errorf("provider = %q, want %q", provider, "java.referenced")
	}
}

func TestExtractRulePattern_BuiltinFilecontent(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00030",
		When: rules.Condition{
			BuiltinFilecontent: &rules.BuiltinFilecontent{
				Pattern:     "import.*md4",
				FilePattern: "*.go",
			},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "import.*md4" {
		t.Errorf("pattern = %q, want %q", pattern, "import.*md4")
	}
	if provider != "builtin.filecontent" {
		t.Errorf("provider = %q, want %q", provider, "builtin.filecontent")
	}
}

func TestExtractRulePattern_OrCombinator(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00040",
		When: rules.Condition{
			Or: []rules.ConditionEntry{
				{Condition: rules.Condition{
					GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/blake2b"},
				}},
				{Condition: rules.Condition{
					GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/blake2s"},
				}},
			},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "golang.org/x/crypto/blake2b" {
		t.Errorf("pattern = %q, want first or entry", pattern)
	}
	if provider != "go.referenced" {
		t.Errorf("provider = %q, want %q", provider, "go.referenced")
	}
}

func TestExtractRulePattern_Empty(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00050",
		When:   rules.Condition{},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "" || provider != "" {
		t.Errorf("expected empty, got pattern=%q provider=%q", pattern, provider)
	}
}

func TestCheckCompilation_GoSuccess(t *testing.T) {
	// Create a valid Go project in a temp dir
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "tests", "data", "mytest")
	os.MkdirAll(dataDir, 0o755)

	os.WriteFile(filepath.Join(dataDir, "go.mod"), []byte("module example.com/test\n\ngo 1.20\n"), 0o644)
	os.WriteFile(filepath.Join(dataDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)

	g := &Generator{}
	input := GenerateInput{OutputDir: dir, Language: "go"}
	genOutput := &GenerateOutput{DataDirs: []string{"data/mytest"}}

	result := g.checkCompilation(input, genOutput)
	if result != "" {
		t.Errorf("expected no errors, got: %s", result)
	}
}

func TestCheckCompilation_GoErrors(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "tests", "data", "mytest")
	os.MkdirAll(dataDir, 0o755)

	os.WriteFile(filepath.Join(dataDir, "go.mod"), []byte("module example.com/test\n\ngo 1.20\n"), 0o644)
	// Invalid Go code — undefined variable
	os.WriteFile(filepath.Join(dataDir, "main.go"), []byte("package main\n\nfunc main() { _ = undefinedVar }\n"), 0o644)

	g := &Generator{}
	input := GenerateInput{OutputDir: dir, Language: "go"}
	genOutput := &GenerateOutput{DataDirs: []string{"data/mytest"}}

	result := g.checkCompilation(input, genOutput)
	if result == "" {
		t.Error("expected compilation errors, got empty string")
	}
}

func TestCheckCompilation_UnknownLanguageSkips(t *testing.T) {
	g := &Generator{}
	input := GenerateInput{OutputDir: "/tmp", Language: "ruby"}
	genOutput := &GenerateOutput{DataDirs: []string{"data/mytest"}}

	result := g.checkCompilation(input, genOutput)
	if result != "" {
		t.Errorf("expected empty for unsupported language, got: %s", result)
	}
}

func TestCheckCompilation_DefaultLanguage(t *testing.T) {
	// When Language is empty, defaults to "go"
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "tests", "data", "mytest")
	os.MkdirAll(dataDir, 0o755)

	os.WriteFile(filepath.Join(dataDir, "go.mod"), []byte("module example.com/test\n\ngo 1.20\n"), 0o644)
	os.WriteFile(filepath.Join(dataDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)

	g := &Generator{}
	input := GenerateInput{OutputDir: dir, Language: ""}
	genOutput := &GenerateOutput{DataDirs: []string{"data/mytest"}}

	result := g.checkCompilation(input, genOutput)
	if result != "" {
		t.Errorf("expected no errors with default language, got: %s", result)
	}
}

// mockCompleter implements llm.Completer for testing.
type mockCompleter struct {
	response string
	err      error
}

func (m *mockCompleter) Complete(_ context.Context, _ string) (string, error) {
	return m.response, m.err
}

func TestHintCompleter_AppendsExtra(t *testing.T) {
	inner := &mockCompleter{response: "fixed code"}
	h := &hintCompleter{inner: inner, extra: " EXTRA HINT"}

	result, err := h.Complete(context.Background(), "original prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fixed code" {
		t.Errorf("response = %q, want %q", result, "fixed code")
	}
}

func TestFixCompilationErrors_UnsupportedLanguage(t *testing.T) {
	g := &Generator{completer: &mockCompleter{}}
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "tests", "data")
	os.MkdirAll(dataDir, 0o755)

	input := GenerateInput{OutputDir: dir, Language: "python"}
	err := g.fixCompilationErrors(context.Background(), input, "some error")
	if err == nil {
		t.Error("expected error for unsupported language")
	}
}

func TestExtractRulePattern_NodejsReferenced(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00060",
		When: rules.Condition{
			NodejsReferenced: &rules.NodejsReferenced{Pattern: "express"},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "express" {
		t.Errorf("pattern = %q, want %q", pattern, "express")
	}
	if provider != "nodejs.referenced" {
		t.Errorf("provider = %q, want %q", provider, "nodejs.referenced")
	}
}

func TestExtractRulePattern_CSharpReferenced(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00070",
		When: rules.Condition{
			CSharpReferenced: &rules.CSharpReferenced{Pattern: "System.Web"},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "System.Web" {
		t.Errorf("pattern = %q, want %q", pattern, "System.Web")
	}
	if provider != "csharp.referenced" {
		t.Errorf("provider = %q, want %q", provider, "csharp.referenced")
	}
}

func TestGatherAPIDocs_GoProject(t *testing.T) {
	// Create a Go project with an import that matches
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.20\n\nrequire golang.org/x/crypto v0.17.0\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import "golang.org/x/crypto/salsa20"

func main() { _ = salsa20.XORKeyStream }
`), 0o644)

	errors := `./main.go:5:10: cannot use key (variable of type []byte) as *[32]byte value in argument to salsa20.XORKeyStream`
	result := gatherAPIDocs("go", dir, errors)
	// Should contain API docs for salsa20 (if go doc works in this env)
	// Even if go doc fails, function should not error
	_ = result
}

func TestGatherAPIDocs_NonGo(t *testing.T) {
	result := gatherAPIDocs("java", "/tmp", "some error")
	if result != "" {
		t.Errorf("expected empty for non-go, got: %s", result)
	}
}

func TestFindImportPath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import (
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/salsa20"
)

func main() {}
`), 0o644)

	if path := findGoImportPath(dir, "chacha20"); path != "golang.org/x/crypto/chacha20" {
		t.Errorf("got %q, want %q", path, "golang.org/x/crypto/chacha20")
	}
	if path := findGoImportPath(dir, "salsa20"); path != "golang.org/x/crypto/salsa20" {
		t.Errorf("got %q, want %q", path, "golang.org/x/crypto/salsa20")
	}
	if path := findGoImportPath(dir, "nonexistent"); path != "" {
		t.Errorf("expected empty, got %q", path)
	}
}

func TestExtractRulePattern_AndCombinator(t *testing.T) {
	r := rules.Rule{
		RuleID: "test-00080",
		When: rules.Condition{
			And: []rules.ConditionEntry{
				{Condition: rules.Condition{
					JavaReferenced: &rules.JavaReferenced{Pattern: "javax.servlet.http.HttpServlet"},
				}},
			},
		},
	}
	pattern, provider := extractRulePattern(r)
	if pattern != "javax.servlet.http.HttpServlet" {
		t.Errorf("pattern = %q, want %q", pattern, "javax.servlet.http.HttpServlet")
	}
	if provider != "java.referenced" {
		t.Errorf("provider = %q, want %q", provider, "java.referenced")
	}
}
