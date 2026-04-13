package testgen

import (
	"context"
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
