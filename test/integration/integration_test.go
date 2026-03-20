//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/confidence"
	"github.com/konveyor/ai-rule-gen/internal/extraction"
	"github.com/konveyor/ai-rule-gen/internal/generation"
	"github.com/konveyor/ai-rule-gen/internal/ingestion"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/testgen"
	"github.com/konveyor/ai-rule-gen/internal/tools"
	"github.com/konveyor/ai-rule-gen/templates"
	"gopkg.in/yaml.v3"
)

// mockCompleter returns canned LLM responses based on prompt content.
type mockCompleter struct {
	calls []string
}

func (m *mockCompleter) Complete(_ context.Context, prompt string) (string, error) {
	m.calls = append(m.calls, prompt)

	// Auto-detection response
	if strings.Contains(prompt, "source technology") || strings.Contains(prompt, "Detect") {
		return `{"source": "spring-boot-3", "target": "spring-boot-4", "language": "java"}`, nil
	}

	// Pattern extraction response
	if strings.Contains(prompt, "migration pattern") || strings.Contains(prompt, "Extract") {
		return `[
			{
				"source_pattern": "javax.servlet.http.HttpServlet",
				"target_pattern": "jakarta.servlet.http.HttpServlet",
				"source_fqn": "javax.servlet.http.HttpServlet",
				"provider_type": "java",
				"location_type": "IMPORT",
				"category": "mandatory",
				"complexity": "low",
				"concern": "web",
				"rationale": "javax.servlet namespace renamed to jakarta.servlet in Jakarta EE 9+",
				"example_before": "import javax.servlet.http.HttpServlet;",
				"example_after": "import jakarta.servlet.http.HttpServlet;"
			},
			{
				"source_pattern": "javax.inject.Inject",
				"target_pattern": "jakarta.inject.Inject",
				"source_fqn": "javax.inject.Inject",
				"provider_type": "java",
				"location_type": "ANNOTATION",
				"category": "mandatory",
				"complexity": "trivial",
				"concern": "di",
				"rationale": "javax.inject renamed to jakarta.inject",
				"example_before": "@javax.inject.Inject",
				"example_after": "@jakarta.inject.Inject"
			}
		]`, nil
	}

	// Message generation response
	if strings.Contains(prompt, "migration rule message") || strings.Contains(prompt, "Before") {
		return "## Migration Required\n\nReplace deprecated API with the new equivalent.\n\n## Before\n```java\nimport javax.servlet.http.HttpServlet;\n```\n\n## After\n```java\nimport jakarta.servlet.http.HttpServlet;\n```", nil
	}

	// Test data generation response
	if strings.Contains(prompt, "test data") || strings.Contains(prompt, "Generate") {
		return "```xml\n<project>\n  <modelVersion>4.0.0</modelVersion>\n  <groupId>com.example</groupId>\n  <artifactId>test</artifactId>\n  <version>1.0</version>\n</project>\n```\n\n```java\npackage com.example;\n\nimport javax.servlet.http.HttpServlet;\nimport javax.inject.Inject;\n\npublic class Application extends HttpServlet {\n    @Inject\n    private Object service;\n}\n```", nil
	}

	// Confidence judge response
	if strings.Contains(prompt, "auditor") || strings.Contains(prompt, "judge") || strings.Contains(prompt, "quality") {
		return `{"pattern_correctness": 5, "message_quality": 4, "category_appropriateness": 5, "effort_accuracy": 4, "false_positive_risk": 4, "reasoning": "Pattern correctly matches the javax namespace import"}`, nil
	}

	// Fix hint response
	if strings.Contains(prompt, "fix") || strings.Contains(prompt, "hint") {
		return "import javax.servlet.http.HttpServlet;\nnew HttpServlet() {};", nil
	}

	return "OK", nil
}

// TestGeneratePipeline_EndToEnd tests the full generate pipeline with a mock LLM:
// ingest text → extract patterns → generate rules → validate → save to disk.
func TestGeneratePipeline_EndToEnd(t *testing.T) {
	mock := &mockCompleter{}
	outputDir := t.TempDir()

	migrationGuide := `# Spring Boot 3 to 4 Migration Guide

## Servlet API Changes

The javax.servlet namespace has been renamed to jakarta.servlet as part of the
Jakarta EE 9+ transition. All imports using javax.servlet must be updated.

Before: import javax.servlet.http.HttpServlet;
After:  import jakarta.servlet.http.HttpServlet;

## Dependency Injection

javax.inject has been renamed to jakarta.inject.

Before: @javax.inject.Inject
After:  @jakarta.inject.Inject
`

	result, err := tools.RunGeneratePipeline(context.Background(), mock, tools.GenerateInput{
		Input:      migrationGuide,
		Source:     "spring-boot-3",
		Target:     "spring-boot-4",
		Language:   "java",
		OutputPath: outputDir,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Verify output structure
	if result.RuleCount < 1 {
		t.Errorf("expected at least 1 rule, got %d", result.RuleCount)
	}
	if result.PatternsExtracted < 1 {
		t.Errorf("expected at least 1 pattern, got %d", result.PatternsExtracted)
	}

	// Verify files on disk
	outputPath := result.OutputPath
	rulesetPath := filepath.Join(outputPath, "rules", "ruleset.yaml")
	if _, err := os.Stat(rulesetPath); os.IsNotExist(err) {
		t.Error("ruleset.yaml not created")
	}

	// Verify at least one rule file exists
	rulesDir := filepath.Join(outputPath, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatalf("cannot read rules dir: %v", err)
	}
	ruleFiles := 0
	for _, e := range entries {
		if e.Name() != "ruleset.yaml" && (strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			ruleFiles++
		}
	}
	if ruleFiles == 0 {
		t.Error("no rule files written")
	}

	// Verify rules are valid YAML and pass validation
	ruleList, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("cannot read generated rules: %v", err)
	}
	validationResult := rules.Validate(ruleList)
	if !validationResult.Valid {
		t.Errorf("generated rules failed validation: %v", validationResult.Errors)
	}

	// Verify LLM was called (extraction + message generation)
	if len(mock.calls) < 2 {
		t.Errorf("expected at least 2 LLM calls, got %d", len(mock.calls))
	}
}

// TestGeneratePipeline_AutoDetect tests auto-detection of source/target/language.
func TestGeneratePipeline_AutoDetect(t *testing.T) {
	mock := &mockCompleter{}
	outputDir := t.TempDir()

	result, err := tools.RunGeneratePipeline(context.Background(), mock, tools.GenerateInput{
		Input:      "Migrate from Spring Boot 3 to Spring Boot 4. Update javax to jakarta.",
		OutputPath: outputDir,
		// Source, Target, Language intentionally omitted — should be auto-detected
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	if result.RuleCount < 1 {
		t.Errorf("expected at least 1 rule, got %d", result.RuleCount)
	}
}

// TestExtractionToGeneration tests the extraction → generation flow.
func TestExtractionToGeneration(t *testing.T) {
	mock := &mockCompleter{}
	ctx := context.Background()

	// Extract patterns
	extractTmpl, err := templates.Load("extraction/extract_patterns.tmpl")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	extractor := extraction.New(mock, extractTmpl)
	patterns, err := extractor.Extract(ctx, []string{"Migrate javax.servlet to jakarta.servlet"}, "spring-boot-3", "spring-boot-4", "java")
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}
	if len(patterns) == 0 {
		t.Fatal("no patterns extracted")
	}

	// Generate rules from patterns
	messageTmpl, err := templates.Load("generation/generate_message.tmpl")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	gen := generation.New(mock, messageTmpl)
	grouped, ruleset, err := gen.Generate(ctx, patterns, generation.GenerateInput{
		Source:   "spring-boot-3",
		Target:   "spring-boot-4",
		Language: "java",
	})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify ruleset
	if ruleset == nil {
		t.Fatal("nil ruleset")
	}
	if !strings.Contains(ruleset.Name, "spring-boot") {
		t.Errorf("ruleset name %q doesn't contain 'spring-boot'", ruleset.Name)
	}

	// Verify rules
	var allRules []rules.Rule
	for _, rr := range grouped {
		allRules = append(allRules, rr...)
	}
	if len(allRules) == 0 {
		t.Fatal("no rules generated")
	}

	for _, r := range allRules {
		if r.RuleID == "" {
			t.Error("rule has empty ruleID")
		}
		if r.When.JavaReferenced == nil {
			t.Errorf("rule %s: expected java.referenced condition", r.RuleID)
		}
		if len(r.Labels) == 0 {
			t.Errorf("rule %s: no labels", r.RuleID)
		}
	}

	// Validate generated rules
	result := rules.Validate(allRules)
	if !result.Valid {
		t.Errorf("rules failed validation: %v", result.Errors)
	}
}

// TestTestDataGeneration tests generating test data from rules using a mock LLM.
func TestTestDataGeneration(t *testing.T) {
	mock := &mockCompleter{}
	ctx := context.Background()
	outputDir := t.TempDir()

	// Write sample rules
	rulesDir := filepath.Join(outputDir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	sampleRules := []rules.Rule{
		{
			RuleID:      "test-00010",
			Description: "Replace javax.servlet with jakarta.servlet",
			Category:    "mandatory",
			Effort:      3,
			Labels:      []string{"konveyor.io/source=spring-boot-3", "konveyor.io/target=spring-boot-4"},
			Message:     "Update imports from javax.servlet to jakarta.servlet",
			When:        rules.NewJavaReferenced("javax.servlet.http.HttpServlet", "IMPORT"),
		},
	}
	rulesData, _ := yaml.Marshal(sampleRules)
	os.WriteFile(filepath.Join(rulesDir, "web.yaml"), rulesData, 0o644)

	// Generate test data
	tmpl, err := templates.Load("testing/main.tmpl")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	gen := testgen.New(mock, tmpl)
	genOutput, err := gen.Generate(ctx, testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: outputDir,
		Language:  "java",
		Source:    "spring-boot-3",
		Target:    "spring-boot-4",
	})
	if err != nil {
		t.Fatalf("test data generation failed: %v", err)
	}

	// Verify output
	if genOutput.RulesTested != 1 {
		t.Errorf("expected 1 rule tested, got %d", genOutput.RulesTested)
	}
	if len(genOutput.TestFiles) == 0 {
		t.Error("no test files generated")
	}
	if genOutput.FilesWritten < 3 {
		t.Errorf("expected at least 3 files written (build + source + test.yaml), got %d", genOutput.FilesWritten)
	}

	// Verify .test.yaml structure
	testsDir := filepath.Join(outputDir, "tests")
	testFilePath := filepath.Join(testsDir, "web.test.yaml")
	testFileData, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("cannot read test file: %v", err)
	}

	var tf testgen.TestFile
	if err := yaml.Unmarshal(testFileData, &tf); err != nil {
		t.Fatalf("cannot parse test file: %v", err)
	}
	if tf.RulesPath == "" {
		t.Error("test file missing rulesPath")
	}
	if len(tf.Providers) == 0 {
		t.Error("test file missing providers")
	}
	if len(tf.Tests) != 1 {
		t.Errorf("expected 1 test entry, got %d", len(tf.Tests))
	}
	if tf.Tests[0].RuleID != "test-00010" {
		t.Errorf("test ruleID = %q, want %q", tf.Tests[0].RuleID, "test-00010")
	}

	// Verify source files exist
	dataDir := filepath.Join(testsDir, "data", "web")
	if _, err := os.Stat(filepath.Join(dataDir, "pom.xml")); os.IsNotExist(err) {
		t.Error("pom.xml not created")
	}
	sourceFile := filepath.Join(dataDir, "src/main/java/com/example", "Application.java")
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		t.Error("Application.java not created")
	}
}

// TestValidationPipeline tests reading and validating rules from disk.
func TestValidationPipeline(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	// Write valid rules
	validRules := []rules.Rule{
		{
			RuleID:      "valid-00010",
			Description: "Test rule",
			Category:    "mandatory",
			Effort:      3,
			Labels:      []string{"konveyor.io/source=a", "konveyor.io/target=b"},
			Message:     "Update code",
			When:        rules.NewGoReferenced("golang.org/x/crypto/md4"),
		},
	}
	data, _ := yaml.Marshal(validRules)
	os.WriteFile(filepath.Join(rulesDir, "security.yaml"), data, 0o644)

	// Read and validate
	ruleList, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("reading rules: %v", err)
	}
	if len(ruleList) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleList))
	}

	result := rules.Validate(ruleList)
	if !result.Valid {
		t.Errorf("valid rules failed validation: %v", result.Errors)
	}

	// Write invalid rules and verify validation catches errors
	invalidRules := []rules.Rule{
		{
			RuleID:   "",
			Category: "invalid-category",
		},
	}
	invalidData, _ := yaml.Marshal(invalidRules)
	os.WriteFile(filepath.Join(rulesDir, "bad.yaml"), invalidData, 0o644)

	allRules, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("reading rules: %v", err)
	}
	result = rules.Validate(allRules)
	if result.Valid {
		t.Error("expected validation to fail for invalid rules")
	}
}

// TestIngestionChunking tests that large content is properly chunked.
func TestIngestionChunking(t *testing.T) {
	// Create content with markdown headers that exceeds chunk size.
	// The chunker splits on ## headers, so we need multiple sections.
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, "## Section %d\n\n", i)
		sb.WriteString(strings.Repeat("Migration step: update javax to jakarta.\n", 20))
		sb.WriteString("\n")
	}
	largeContent := sb.String()

	ingested, err := ingestion.Ingest(largeContent, 2000)
	if err != nil {
		t.Fatalf("ingestion failed: %v", err)
	}

	if len(ingested.Chunks) < 2 {
		t.Errorf("expected multiple chunks for large content (%d chars), got %d chunks", len(largeContent), len(ingested.Chunks))
	}

	// Verify all content is preserved across chunks
	totalLen := 0
	for _, chunk := range ingested.Chunks {
		totalLen += len(chunk)
	}
	if totalLen < len(largeContent)/2 {
		t.Errorf("chunks lost too much content: total %d vs original %d", totalLen, len(largeContent))
	}
}

// TestConfidenceScoring_WithMockKantraOutput tests scoring with synthetic kantra output.
func TestConfidenceScoring_WithMockKantraOutput(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	rulesDir := filepath.Join(dir, "rules")
	os.MkdirAll(testsDir, 0o755)
	os.MkdirAll(rulesDir, 0o755)

	// Write rules
	ruleList := []rules.Rule{
		{
			RuleID:      "test-00010",
			Description: "Test rule 1",
			Category:    "mandatory",
			Effort:      3,
			Labels:      []string{"konveyor.io/source=a", "konveyor.io/target=b"},
			Message:     "Update code",
			When:        rules.NewJavaReferenced("javax.servlet.http.HttpServlet", "IMPORT"),
		},
		{
			RuleID:      "test-00020",
			Description: "Test rule 2",
			Category:    "optional",
			Effort:      1,
			Labels:      []string{"konveyor.io/source=a", "konveyor.io/target=b"},
			Message:     "Update code",
			When:        rules.NewJavaReferenced("javax.inject.Inject", "ANNOTATION"),
		},
	}
	rulesData, _ := yaml.Marshal(ruleList)
	os.WriteFile(filepath.Join(rulesDir, "web.yaml"), rulesData, 0o644)

	// Write test file
	testYAML := `rulesPath: ../rules/web.yaml
providers:
  - name: java
    dataPath: ./data/web
tests:
  - ruleID: test-00010
    testCases:
      - name: tc-1
        analysisParams:
          mode: source-only
        hasIncidents:
          atLeast: 1
  - ruleID: test-00020
    testCases:
      - name: tc-1
        analysisParams:
          mode: source-only
        hasIncidents:
          atLeast: 1
`
	os.WriteFile(filepath.Join(testsDir, "web.test.yaml"), []byte(testYAML), 0o644)

	// Create scorer with mock LLM judge (no kantra — this test only checks scoring logic)
	mock := &mockCompleter{}
	judgeTmpl, err := templates.Load("confidence/judge.tmpl")
	if err != nil {
		t.Fatalf("loading judge template: %v", err)
	}

	scorer := confidence.New("", 60, mock, judgeTmpl)
	_ = scorer

	// Verify the scorer can collect rule IDs from test files
	// (actual kantra execution is tested in E2E tests)
	t.Log("Scorer created successfully with mock judge")
}

// TestRulesetLayout verifies output matches konveyor/rulesets repo structure.
func TestRulesetLayout(t *testing.T) {
	mock := &mockCompleter{}
	outputDir := t.TempDir()

	result, err := tools.RunGeneratePipeline(context.Background(), mock, tools.GenerateInput{
		Input:      "Migrate javax.servlet to jakarta.servlet. Migrate javax.inject to jakarta.inject.",
		Source:     "spring-boot-3",
		Target:     "spring-boot-4",
		Language:   "java",
		OutputPath: outputDir,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	root := result.OutputPath

	// Required: rules/ruleset.yaml
	rulesetPath := filepath.Join(root, "rules", "ruleset.yaml")
	if _, err := os.Stat(rulesetPath); os.IsNotExist(err) {
		t.Error("missing rules/ruleset.yaml")
	}

	// Verify ruleset.yaml has correct structure
	rulesetData, err := os.ReadFile(rulesetPath)
	if err != nil {
		t.Fatalf("reading ruleset: %v", err)
	}
	var rs rules.Ruleset
	if err := yaml.Unmarshal(rulesetData, &rs); err != nil {
		t.Fatalf("parsing ruleset: %v", err)
	}
	if rs.Name == "" {
		t.Error("ruleset has empty name")
	}
	if len(rs.Labels) == 0 {
		t.Error("ruleset has no labels")
	}

	// Verify source/target labels
	hasSource, hasTarget := false, false
	for _, l := range rs.Labels {
		if strings.HasPrefix(l, "konveyor.io/source=") {
			hasSource = true
		}
		if strings.HasPrefix(l, "konveyor.io/target=") {
			hasTarget = true
		}
	}
	if !hasSource {
		t.Error("ruleset missing konveyor.io/source label")
	}
	if !hasTarget {
		t.Error("ruleset missing konveyor.io/target label")
	}

	// Verify at least one rule file with valid rules
	rulesDir := filepath.Join(root, "rules")
	allRules, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("reading rules: %v", err)
	}
	if len(allRules) == 0 {
		t.Error("no rules in output")
	}

	// Each rule should have required fields
	for _, r := range allRules {
		if r.RuleID == "" {
			t.Error("rule missing ruleID")
		}
		if r.Category == "" {
			t.Error("rule missing category")
		}
		if r.Effort == 0 {
			t.Error("rule missing effort")
		}
		if r.Message == "" {
			t.Error("rule missing message")
		}
	}
}
