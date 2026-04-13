package testgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// ---------- extractCodeBlocks ----------

func TestExtractCodeBlocks_Single(t *testing.T) {
	response := "```xml\n<project/>\n```"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Language != "xml" {
		t.Errorf("language = %q, want %q", blocks[0].Language, "xml")
	}
	if blocks[0].Content != "<project/>" {
		t.Errorf("content = %q, want %q", blocks[0].Content, "<project/>")
	}
}

func TestExtractCodeBlocks_Multiple(t *testing.T) {
	response := "```xml\n<project/>\n```\n\n```java\npublic class A {}\n```"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Language != "xml" {
		t.Errorf("block[0] language = %q, want xml", blocks[0].Language)
	}
	if blocks[1].Language != "java" {
		t.Errorf("block[1] language = %q, want java", blocks[1].Language)
	}
}

func TestExtractCodeBlocks_MultilineContent(t *testing.T) {
	response := "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println() }\n```"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if !strings.Contains(blocks[0].Content, "import") {
		t.Error("content should contain 'import'")
	}
	if !strings.Contains(blocks[0].Content, "func main") {
		t.Error("content should contain 'func main'")
	}
}

func TestExtractCodeBlocks_NoBlocks(t *testing.T) {
	response := "Here is some plain text without any code blocks."
	blocks := extractCodeBlocks(response)
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractCodeBlocks_NoLanguageTag(t *testing.T) {
	response := "```\nsome code\n```"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Language != "" {
		t.Errorf("language = %q, want empty", blocks[0].Language)
	}
}

func TestExtractCodeBlocks_Unclosed(t *testing.T) {
	// Unclosed code block — should not produce a block
	response := "```java\npublic class A {}\n"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for unclosed fence, got %d", len(blocks))
	}
}

func TestExtractCodeBlocks_EmptyBlock(t *testing.T) {
	response := "```java\n```"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Content != "" {
		t.Errorf("expected empty content, got %q", blocks[0].Content)
	}
}

func TestExtractCodeBlocks_ExtraTextBetween(t *testing.T) {
	response := "Here is the pom.xml:\n```xml\n<project/>\n```\n\nAnd the source:\n```java\nclass A {}\n```\n\nDone!"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Content != "<project/>" {
		t.Errorf("block[0] content = %q", blocks[0].Content)
	}
	if blocks[1].Content != "class A {}" {
		t.Errorf("block[1] content = %q", blocks[1].Content)
	}
}

func TestExtractCodeBlocks_IndentedFence(t *testing.T) {
	// Fences with leading whitespace should still be detected
	response := "  ```java\n  class A {}\n  ```"
	blocks := extractCodeBlocks(response)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block for indented fence, got %d", len(blocks))
	}
}

// ---------- detectLanguage ----------

func TestDetectLanguage_Java(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless"}}},
	}
	if lang := detectLanguage(ruleList); lang != "java" {
		t.Errorf("got %q, want java", lang)
	}
}

func TestDetectLanguage_Go(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/md4"}}},
	}
	if lang := detectLanguage(ruleList); lang != "go" {
		t.Errorf("got %q, want go", lang)
	}
}

func TestDetectLanguage_Nodejs(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{NodejsReferenced: &rules.NodejsReferenced{Pattern: "express"}}},
	}
	if lang := detectLanguage(ruleList); lang != "nodejs" {
		t.Errorf("got %q, want nodejs", lang)
	}
}

func TestDetectLanguage_CSharp(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{CSharpReferenced: &rules.CSharpReferenced{Pattern: "System.Web"}}},
	}
	if lang := detectLanguage(ruleList); lang != "csharp" {
		t.Errorf("got %q, want csharp", lang)
	}
}

func TestDetectLanguage_Empty(t *testing.T) {
	if lang := detectLanguage(nil); lang != "" {
		t.Errorf("got %q, want empty", lang)
	}
}

func TestDetectLanguage_JavaDependency(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.springframework.boot.spring-boot-starter-parent"}}},
	}
	if lang := detectLanguage(ruleList); lang != "java" {
		t.Errorf("got %q, want java", lang)
	}
}

// ---------- detectConditionLanguage ----------

func TestDetectConditionLanguage_BuiltinFilecontentJava(t *testing.T) {
	c := rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{FilePattern: "*.java"}}
	if lang := detectConditionLanguage(c); lang != "java" {
		t.Errorf("got %q, want java", lang)
	}
}

func TestDetectConditionLanguage_BuiltinFilecontentGo(t *testing.T) {
	c := rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{FilePattern: "*.go"}}
	if lang := detectConditionLanguage(c); lang != "go" {
		t.Errorf("got %q, want go", lang)
	}
}

func TestDetectConditionLanguage_BuiltinFilecontentNodejs(t *testing.T) {
	c := rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{FilePattern: "*.ts"}}
	if lang := detectConditionLanguage(c); lang != "nodejs" {
		t.Errorf("got %q, want nodejs", lang)
	}
}

func TestDetectConditionLanguage_BuiltinFilecontentCSharp(t *testing.T) {
	c := rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{FilePattern: "*.cs"}}
	if lang := detectConditionLanguage(c); lang != "csharp" {
		t.Errorf("got %q, want csharp", lang)
	}
}

func TestDetectConditionLanguage_BuiltinFilecontentUnknown(t *testing.T) {
	c := rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{FilePattern: "*.yaml"}}
	if lang := detectConditionLanguage(c); lang != "" {
		t.Errorf("got %q, want empty for unknown file extension", lang)
	}
}

func TestDetectConditionLanguage_OrCombinator(t *testing.T) {
	c := rules.Condition{
		Or: []rules.ConditionEntry{
			{Condition: rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/md4"}}},
		},
	}
	if lang := detectConditionLanguage(c); lang != "go" {
		t.Errorf("got %q, want go", lang)
	}
}

func TestDetectConditionLanguage_AndCombinator(t *testing.T) {
	c := rules.Condition{
		And: []rules.ConditionEntry{
			{Condition: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless"}}},
		},
	}
	if lang := detectConditionLanguage(c); lang != "java" {
		t.Errorf("got %q, want java", lang)
	}
}

func TestDetectConditionLanguage_Empty(t *testing.T) {
	if lang := detectConditionLanguage(rules.Condition{}); lang != "" {
		t.Errorf("got %q, want empty for empty condition", lang)
	}
}

// ---------- detectProviders ----------

func TestDetectProviders_Java(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless"}}},
	}
	providers := detectProviders(ruleList)
	if len(providers) != 1 || providers[0] != "java" {
		t.Errorf("got %v, want [java]", providers)
	}
}

func TestDetectProviders_Go(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "golang.org/x/crypto/md4"}}},
	}
	providers := detectProviders(ruleList)
	if len(providers) != 1 || providers[0] != "go" {
		t.Errorf("got %v, want [go]", providers)
	}
}

func TestDetectProviders_DefaultBuiltin(t *testing.T) {
	// Empty condition → falls back to "builtin"
	providers := detectProviders([]rules.Rule{{When: rules.Condition{}}})
	if len(providers) != 1 || providers[0] != "builtin" {
		t.Errorf("got %v, want [builtin]", providers)
	}
}

func TestDetectProviders_Deduplicates(t *testing.T) {
	ruleList := []rules.Rule{
		{When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "a"}}},
		{When: rules.Condition{JavaDependency: &rules.Dependency{Name: "b"}}},
	}
	providers := detectProviders(ruleList)
	if len(providers) != 1 || providers[0] != "java" {
		t.Errorf("expected dedup to 1 java provider, got %v", providers)
	}
}

// ---------- conditionProviders ----------

func TestConditionProviders_Builtin(t *testing.T) {
	c := rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{Pattern: "x"}}
	providers := conditionProviders(c)
	if len(providers) != 1 || providers[0] != "builtin" {
		t.Errorf("got %v, want [builtin]", providers)
	}
}

func TestConditionProviders_Nodejs(t *testing.T) {
	c := rules.Condition{NodejsReferenced: &rules.NodejsReferenced{Pattern: "express"}}
	providers := conditionProviders(c)
	if len(providers) != 1 || providers[0] != "nodejs" {
		t.Errorf("got %v, want [nodejs]", providers)
	}
}

func TestConditionProviders_Dotnet(t *testing.T) {
	c := rules.Condition{CSharpReferenced: &rules.CSharpReferenced{Pattern: "System.Web"}}
	providers := conditionProviders(c)
	if len(providers) != 1 || providers[0] != "dotnet" {
		t.Errorf("got %v, want [dotnet]", providers)
	}
}

func TestConditionProviders_OrCombinator(t *testing.T) {
	c := rules.Condition{
		Or: []rules.ConditionEntry{
			{Condition: rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "p"}}},
			{Condition: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "q"}}},
		},
	}
	providers := conditionProviders(c)
	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[p] = true
	}
	if !providerSet["go"] {
		t.Error("expected 'go' in providers")
	}
	if !providerSet["java"] {
		t.Error("expected 'java' in providers")
	}
}

// ---------- sanitizeXMLComments ----------

func TestSanitizeXMLComments_ReplacesDoubleDash(t *testing.T) {
	input := "<!-- --add-opens flag -->"
	got := sanitizeXMLComments(input)
	if strings.Contains(got, "--") && strings.Contains(got, "add-opens") {
		// The double dash inside the comment content should be replaced
		inner := got[4 : len(got)-3] // strip <!-- and -->
		if strings.Contains(inner, "--") {
			t.Errorf("double dash not sanitized inside comment, got: %s", got)
		}
	}
}

func TestSanitizeXMLComments_NoComments(t *testing.T) {
	input := "<dependency><groupId>org.example</groupId></dependency>"
	got := sanitizeXMLComments(input)
	if got != input {
		t.Errorf("content without comments should be unchanged, got: %s", got)
	}
}

func TestSanitizeXMLComments_CleanComment(t *testing.T) {
	input := "<!-- normal comment -->"
	got := sanitizeXMLComments(input)
	// No double dash in "normal comment", so should be unchanged (or at least valid)
	if !strings.HasPrefix(got, "<!--") || !strings.HasSuffix(got, "-->") {
		t.Errorf("comment structure broken: %s", got)
	}
}

func TestSanitizeXMLComments_MultipleComments(t *testing.T) {
	input := "<!-- first -- comment -->\n<tag/>\n<!-- second -- comment -->"
	got := sanitizeXMLComments(input)
	// Count remaining double dashes not inside <!-- -->: none should remain inside comments
	// Simple check: result should still have two comment markers
	if strings.Count(got, "<!--") != 2 {
		t.Errorf("expected 2 comments, got: %s", got)
	}
}

// ---------- buildTestFile ----------

func TestBuildTestFile_Structure(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	testsDir := filepath.Join(dir, "tests")
	dataDir := filepath.Join(dir, "tests", "data", "web")
	os.MkdirAll(rulesDir, 0o755)
	os.MkdirAll(testsDir, 0o755)
	os.MkdirAll(dataDir, 0o755)

	ruleFilePath := filepath.Join(rulesDir, "web.yaml")
	ruleList := []rules.Rule{
		{RuleID: "test-00010", When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless"}}},
		{RuleID: "test-00020", When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.inject.Inject"}}},
	}

	tf := buildTestFile(ruleList, ruleFilePath, dataDir, testsDir, []string{"java"})

	if tf.RulesPath == "" {
		t.Error("RulesPath should not be empty")
	}
	if len(tf.Providers) != 1 || tf.Providers[0].Name != "java" {
		t.Errorf("expected 1 java provider, got %v", tf.Providers)
	}
	if len(tf.Tests) != 2 {
		t.Errorf("expected 2 test entries, got %d", len(tf.Tests))
	}
	if tf.Tests[0].RuleID != "test-00010" {
		t.Errorf("first test ruleID = %q, want %q", tf.Tests[0].RuleID, "test-00010")
	}
	if tf.Tests[1].RuleID != "test-00020" {
		t.Errorf("second test ruleID = %q, want %q", tf.Tests[1].RuleID, "test-00020")
	}
	// Each test case should have tc-1 with source-only mode
	for i, test := range tf.Tests {
		if len(test.TestCases) != 1 {
			t.Errorf("test[%d]: expected 1 test case, got %d", i, len(test.TestCases))
		}
		if test.TestCases[0].Name != "tc-1" {
			t.Errorf("test[%d]: test case name = %q, want tc-1", i, test.TestCases[0].Name)
		}
		if test.TestCases[0].AnalysisParams.Mode != "source-only" {
			t.Errorf("test[%d]: mode = %q, want source-only", i, test.TestCases[0].AnalysisParams.Mode)
		}
		if test.TestCases[0].HasIncidents.AtLeast != 1 {
			t.Errorf("test[%d]: hasIncidents.atLeast = %d, want 1", i, test.TestCases[0].HasIncidents.AtLeast)
		}
	}
}

func TestBuildTestFile_MultipleProviders(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	testsDir := filepath.Join(dir, "tests")
	dataDir := filepath.Join(dir, "tests", "data", "mixed")
	os.MkdirAll(rulesDir, 0o755)
	os.MkdirAll(testsDir, 0o755)
	os.MkdirAll(dataDir, 0o755)

	ruleFilePath := filepath.Join(rulesDir, "mixed.yaml")
	ruleList := []rules.Rule{
		{RuleID: "test-00010"},
	}

	tf := buildTestFile(ruleList, ruleFilePath, dataDir, testsDir, []string{"java", "builtin"})

	if len(tf.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(tf.Providers))
	}
}
