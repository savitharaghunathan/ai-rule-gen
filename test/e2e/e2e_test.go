//go:build e2e

// Package e2e contains end-to-end tests that require a real LLM provider and kantra.
//
// Prerequisites:
//   - LLM API key: set RULEGEN_LLM_PROVIDER and corresponding API key env var
//   - kantra: must be on PATH (required for test and score commands)
//
// Run: go test -tags=e2e -timeout 600s ./test/e2e/...
package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/confidence"
	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/testgen"
	"github.com/konveyor/ai-rule-gen/internal/tools"
	"github.com/konveyor/ai-rule-gen/templates"
	"gopkg.in/yaml.v3"
)

func skipIfNoLLM(t *testing.T) llm.Completer {
	t.Helper()
	provider := os.Getenv("RULEGEN_LLM_PROVIDER")
	if provider == "" {
		t.Skip("RULEGEN_LLM_PROVIDER not set — skipping E2E test")
	}
	c, err := llm.NewCompleterFromEnv()
	if err != nil {
		t.Fatalf("LLM configuration error: %v", err)
	}
	if c == nil {
		t.Skip("LLM provider not configured — skipping E2E test")
	}
	return c
}

func skipIfNoKantra(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kantra"); err != nil {
		t.Skip("kantra not on PATH — skipping E2E test")
	}
}

// TestE2E_GenerateFromText generates rules from inline migration text using a real LLM.
func TestE2E_GenerateFromText(t *testing.T) {
	completer := skipIfNoLLM(t)
	outputDir := t.TempDir()

	migrationGuide := `# Go FIPS Compliance Migration

Replace non-FIPS compliant crypto packages from golang.org/x/crypto with
standard library crypto packages that are FIPS 140-2 compliant.

## Changes Required

1. Replace golang.org/x/crypto/md4 with crypto/md5 (MD4 is not FIPS compliant)
2. Replace golang.org/x/crypto/bcrypt with crypto/sha256 based hashing
3. Replace golang.org/x/crypto/chacha20 with crypto/aes (use AES-GCM instead)
`

	result, err := tools.RunGeneratePipeline(context.Background(), completer, tools.GenerateInput{
		Input:      migrationGuide,
		Source:     "go-non-fips-crypto",
		Target:     "go-fips-140-compliance",
		Language:   "go",
		OutputPath: outputDir,
	})
	if err != nil {
		t.Fatalf("generate pipeline failed: %v", err)
	}

	t.Logf("Generated %d rules from %d patterns", result.RuleCount, result.PatternsExtracted)

	if result.RuleCount == 0 {
		t.Fatal("no rules generated")
	}

	// Verify rules are valid
	rulesDir := filepath.Join(result.OutputPath, "rules")
	ruleList, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("reading rules: %v", err)
	}

	validationResult := rules.Validate(ruleList)
	if !validationResult.Valid {
		t.Errorf("validation failed: %v", validationResult.Errors)
	}

	// Verify conditions use go.referenced
	for _, r := range ruleList {
		if r.When.GoReferenced == nil && len(r.When.Or) == 0 {
			t.Errorf("rule %s: expected go.referenced condition, got something else", r.RuleID)
		}
	}
}

// TestE2E_GenerateAutoDetect generates rules with auto-detected source/target/language.
func TestE2E_GenerateAutoDetect(t *testing.T) {
	completer := skipIfNoLLM(t)
	outputDir := t.TempDir()

	result, err := tools.RunGeneratePipeline(context.Background(), completer, tools.GenerateInput{
		Input:      "Migrate from Spring Boot 3 to Spring Boot 4. Replace javax.servlet with jakarta.servlet. Replace javax.inject with jakarta.inject.",
		OutputPath: outputDir,
	})
	if err != nil {
		t.Fatalf("generate pipeline failed: %v", err)
	}

	t.Logf("Auto-detect: generated %d rules, output at %s", result.RuleCount, result.OutputPath)

	if result.RuleCount == 0 {
		t.Fatal("no rules generated")
	}

	// Verify output has source/target in the path
	if !strings.Contains(result.OutputPath, "spring") {
		t.Logf("Warning: output path %q may not reflect auto-detected source/target", result.OutputPath)
	}
}

// TestE2E_TestDataGeneration generates test data and verifies compilation.
func TestE2E_TestDataGeneration(t *testing.T) {
	completer := skipIfNoLLM(t)
	outputDir := t.TempDir()

	// Write a simple Go rule
	rulesDir := filepath.Join(outputDir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	goRules := []rules.Rule{
		{
			RuleID:      "e2e-test-00010",
			Description: "Replace golang.org/x/crypto/md4",
			Category:    "mandatory",
			Effort:      3,
			Labels:      []string{"konveyor.io/source=go-non-fips", "konveyor.io/target=go-fips"},
			Message:     "Replace md4 with a FIPS-compliant hash",
			When:        rules.NewGoReferenced("golang.org/x/crypto/md4"),
		},
	}
	data, _ := yaml.Marshal(goRules)
	os.WriteFile(filepath.Join(rulesDir, "security.yaml"), data, 0o644)

	// Also write a ruleset
	rs := rules.Ruleset{
		Name:        "go-fips/go-non-fips",
		Description: "E2E test ruleset",
		Labels:      []string{"konveyor.io/source=go-non-fips", "konveyor.io/target=go-fips"},
	}
	rsData, _ := yaml.Marshal(rs)
	os.WriteFile(filepath.Join(rulesDir, "ruleset.yaml"), rsData, 0o644)

	// Generate test data
	tmpl, err := templates.Load("testing/main.tmpl")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	gen := testgen.New(completer, tmpl)
	genOutput, err := gen.Generate(context.Background(), testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: outputDir,
		Language:  "go",
		Source:    "go-non-fips",
		Target:    "go-fips",
	})
	if err != nil {
		t.Fatalf("test data generation failed: %v", err)
	}

	t.Logf("Generated test data: %d files written, %d rules tested", genOutput.FilesWritten, genOutput.RulesTested)

	if genOutput.RulesTested != 1 {
		t.Errorf("expected 1 rule tested, got %d", genOutput.RulesTested)
	}

	// Verify .test.yaml exists
	testsDir := filepath.Join(outputDir, "tests")
	testFiles, err := filepath.Glob(filepath.Join(testsDir, "*.test.yaml"))
	if err != nil || len(testFiles) == 0 {
		t.Fatal("no .test.yaml files generated")
	}

	// Verify source file exists
	dataDir := filepath.Join(testsDir, "data", "security")
	mainGo := filepath.Join(dataDir, "main.go")
	if _, err := os.Stat(mainGo); os.IsNotExist(err) {
		t.Error("main.go not created")
	}

	// Verify go.mod exists
	goMod := filepath.Join(dataDir, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		t.Error("go.mod not created")
	}
}

// TestE2E_KantraTest runs kantra tests on generated test data.
// Requires both an LLM provider and kantra on PATH.
func TestE2E_KantraTest(t *testing.T) {
	completer := skipIfNoLLM(t)
	skipIfNoKantra(t)
	outputDir := t.TempDir()

	// Write a simple Go rule
	rulesDir := filepath.Join(outputDir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	goRules := []rules.Rule{
		{
			RuleID:      "e2e-kantra-00010",
			Description: "Replace golang.org/x/crypto/md4",
			Category:    "mandatory",
			Effort:      3,
			Labels:      []string{"konveyor.io/source=go-non-fips", "konveyor.io/target=go-fips"},
			Message:     "Replace md4 with a FIPS-compliant hash",
			When:        rules.NewGoReferenced("golang.org/x/crypto/md4"),
		},
	}
	data, _ := yaml.Marshal(goRules)
	os.WriteFile(filepath.Join(rulesDir, "security.yaml"), data, 0o644)

	rs := rules.Ruleset{
		Name:        "go-fips/go-non-fips",
		Description: "E2E test ruleset",
		Labels:      []string{"konveyor.io/source=go-non-fips", "konveyor.io/target=go-fips"},
	}
	rsData, _ := yaml.Marshal(rs)
	os.WriteFile(filepath.Join(rulesDir, "ruleset.yaml"), rsData, 0o644)

	// Generate test data
	tmpl, err := templates.Load("testing/main.tmpl")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	fixTmpl, err := templates.Load("testing/fix_hint.tmpl")
	if err != nil {
		t.Fatalf("loading fix_hint template: %v", err)
	}

	gen := testgen.New(completer, tmpl)
	result, err := gen.RunWithTests(context.Background(), testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: outputDir,
		Language:  "go",
		Source:    "go-non-fips",
		Target:    "go-fips",
	}, fixTmpl, 2)
	if err != nil {
		t.Fatalf("RunWithTests failed: %v", err)
	}

	t.Logf("Kantra result: %d/%d passed (%.0f%%) in %d iterations",
		result.TestResult.Passed, result.TestResult.Total,
		result.TestResult.PassRate, result.Iterations)

	if result.TestResult.Total == 0 {
		t.Error("kantra reported 0 total rules")
	}

	// We expect at least some rules to pass after fix loop
	if result.TestResult.Passed == 0 {
		t.Errorf("no rules passed after %d fix iterations", result.Iterations)
	}
}

// TestE2E_ConfidenceScore runs confidence scoring (functional + optional LLM judge).
// Requires kantra on PATH. LLM judge runs only if RULEGEN_LLM_PROVIDER is set.
func TestE2E_ConfidenceScore(t *testing.T) {
	skipIfNoKantra(t)

	// Use the existing output if available, otherwise skip
	existingTests := "../../output/go-non-fips-crypto-to-go-fips-140-compliance/tests"
	existingRules := "../../output/go-non-fips-crypto-to-go-fips-140-compliance/rules"

	if _, err := os.Stat(existingTests); os.IsNotExist(err) {
		t.Skip("no existing test output found — run 'rulegen test' first")
	}

	// Functional scoring (no LLM needed)
	scorer := confidence.New("", 900, nil, nil)
	report, err := scorer.ScoreRules(context.Background(), existingTests, "")
	if err != nil {
		t.Fatalf("scoring failed: %v", err)
	}

	t.Logf("Functional: %d/%d passed (%.0f%%)", report.Summary.Passed, report.Summary.TotalRules, report.Summary.PassRate)

	if report.Summary.TotalRules == 0 {
		t.Error("no rules scored")
	}

	// If LLM provider is available, also test with judge
	if os.Getenv("RULEGEN_LLM_PROVIDER") != "" {
		completer, err := llm.NewCompleterFromEnv()
		if err != nil {
			t.Fatalf("LLM config error: %v", err)
		}
		judgeTmpl, err := templates.Load("confidence/judge.tmpl")
		if err != nil {
			t.Fatalf("loading judge template: %v", err)
		}

		if _, err := os.Stat(existingRules); os.IsNotExist(err) {
			t.Log("Rules dir not found — skipping LLM judge test")
			return
		}

		scorerWithJudge := confidence.New("", 900, completer, judgeTmpl)
		reportWithJudge, err := scorerWithJudge.ScoreRules(context.Background(), existingTests, existingRules)
		if err != nil {
			t.Fatalf("scoring with judge failed: %v", err)
		}

		t.Logf("With judge: %d/%d passed", reportWithJudge.Summary.Passed, reportWithJudge.Summary.TotalRules)

		// Verify judge scores are populated
		hasJudge := false
		for _, s := range reportWithJudge.Scores {
			if s.JudgeVerdict != "" {
				hasJudge = true
				break
			}
		}
		if !hasJudge {
			t.Error("LLM judge ran but no scores have JudgeVerdict set")
		}
	}
}

// TestE2E_FullPipeline runs the complete pipeline: generate → test → score.
// This is the most comprehensive E2E test.
func TestE2E_FullPipeline(t *testing.T) {
	completer := skipIfNoLLM(t)
	skipIfNoKantra(t)

	if testing.Short() {
		t.Skip("skipping full pipeline in short mode")
	}

	outputDir := t.TempDir()
	ctx := context.Background()

	// Step 1: Generate rules
	t.Log("Step 1: Generating rules...")
	genResult, err := tools.RunGeneratePipeline(ctx, completer, tools.GenerateInput{
		Input:      "Replace golang.org/x/crypto/md4 with crypto/md5 for FIPS compliance. Replace golang.org/x/crypto/bcrypt with standard crypto.",
		Source:     "go-non-fips",
		Target:     "go-fips",
		Language:   "go",
		OutputPath: outputDir,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	t.Logf("  Generated %d rules", genResult.RuleCount)

	if genResult.RuleCount == 0 {
		t.Fatal("no rules generated")
	}

	// Step 2: Generate test data and run kantra
	t.Log("Step 2: Generating test data and running kantra...")
	rulesDir := filepath.Join(genResult.OutputPath, "rules")

	tmpl, err := templates.Load("testing/main.tmpl")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	fixTmpl, err := templates.Load("testing/fix_hint.tmpl")
	if err != nil {
		t.Fatalf("loading fix_hint template: %v", err)
	}

	gen := testgen.New(completer, tmpl)
	testResult, err := gen.RunWithTests(ctx, testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: genResult.OutputPath,
		Language:  "go",
		Source:    "go-non-fips",
		Target:    "go-fips",
	}, fixTmpl, 2)
	if err != nil {
		t.Fatalf("test failed: %v", err)
	}
	t.Logf("  Test result: %d/%d passed", testResult.TestResult.Passed, testResult.TestResult.Total)

	// Step 3: Score confidence
	t.Log("Step 3: Scoring confidence...")
	testsDir := filepath.Join(genResult.OutputPath, "tests")
	scorer := confidence.New("", 900, nil, nil)
	report, err := scorer.ScoreRules(ctx, testsDir, "")
	if err != nil {
		t.Fatalf("scoring failed: %v", err)
	}
	t.Logf("  Score: %d/%d passed (%.0f%%)", report.Summary.Passed, report.Summary.TotalRules, report.Summary.PassRate)

	// Final assertions
	if report.Summary.TotalRules == 0 {
		t.Error("no rules in score report")
	}

	// Log full report for debugging
	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	t.Logf("Full report:\n%s", string(reportJSON))
}
