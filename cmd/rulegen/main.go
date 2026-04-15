package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	stdtemplate "text/template"

	"github.com/konveyor/ai-rule-gen/internal/confidence"
	"github.com/konveyor/ai-rule-gen/internal/ingestion"
	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/server"
	"github.com/konveyor/ai-rule-gen/internal/testgen"
	"github.com/konveyor/ai-rule-gen/internal/tools"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
	"github.com/konveyor/ai-rule-gen/templates"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// signalContext returns a context that is cancelled on SIGINT or SIGTERM.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

var (
	host         string
	port         int
	transport    string
	experimental bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rulegen",
		Short: "AI-powered Konveyor analyzer rule generation",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		RunE:  runServe,
	}
	serveCmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio, http")
	serveCmd.Flags().StringVar(&host, "host", "localhost", "Host to bind to (http transport only)")
	serveCmd.Flags().IntVar(&port, "port", 8080, "Port to listen on (http transport only)")

	var genInput, genSource, genTarget, genLanguage, genOutput, genProvider string
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate rules from a migration guide",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(genInput, genSource, genTarget, genLanguage, genOutput, genProvider)
		},
	}
	generateCmd.Flags().StringVar(&genInput, "input", "", "URL, file path, or text content to generate rules from")
	generateCmd.Flags().StringVar(&genSource, "source", "", "Source technology (e.g., spring-boot-3)")
	generateCmd.Flags().StringVar(&genTarget, "target", "", "Target technology (e.g., spring-boot-4)")
	generateCmd.Flags().StringVar(&genLanguage, "language", "", "Programming language (java, go, nodejs, csharp)")
	generateCmd.Flags().StringVar(&genOutput, "output", "output", "Output directory path")
	generateCmd.Flags().StringVar(&genProvider, "provider", "", "LLM provider: anthropic, openai, gemini, ollama (overrides RULEGEN_LLM_PROVIDER)")

	var valRulesPath string
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate rule YAML files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(valRulesPath)
		},
	}
	validateCmd.Flags().StringVar(&valRulesPath, "rules", "", "Path to rules directory or file")

	var testRulesDir, testOutputDir, testLanguage, testSource, testTarget, testProvider string
	var testMaxIterations int
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Generate test data, run kantra tests, and fix failing test data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(testRulesDir, testOutputDir, testLanguage, testSource, testTarget, testProvider, testMaxIterations)
		},
	}
	testCmd.Flags().StringVar(&testRulesDir, "rules", "", "Path to rules directory")
	testCmd.Flags().StringVar(&testOutputDir, "output", "", "Output directory (parent of rules/)")
	testCmd.Flags().StringVar(&testLanguage, "language", "", "Programming language (auto-detected if omitted)")
	testCmd.Flags().StringVar(&testSource, "source", "", "Source technology")
	testCmd.Flags().StringVar(&testTarget, "target", "", "Target technology")
	testCmd.Flags().StringVar(&testProvider, "provider", "", "LLM provider: anthropic, openai, gemini, ollama")
	testCmd.Flags().IntVar(&testMaxIterations, "max-iterations", 3, "Max test-fix iterations")

	var scoreTestsDir, scoreRulesDir, scoreOutputDir, scoreKantraPath, scoreProvider string
	var scoreTimeout int
	scoreCmd := &cobra.Command{
		Use:   "score",
		Short: "Run kantra tests and score confidence on generated rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScore(scoreTestsDir, scoreRulesDir, scoreOutputDir, scoreKantraPath, scoreProvider, scoreTimeout)
		},
	}
	scoreCmd.Flags().StringVar(&scoreTestsDir, "tests", "", "Path to tests directory containing .test.yaml files")
	scoreCmd.Flags().StringVar(&scoreRulesDir, "rules", "", "Path to rules directory (required for LLM judge)")
	scoreCmd.Flags().StringVar(&scoreOutputDir, "output", "", "Output directory for scores")
	scoreCmd.Flags().StringVar(&scoreKantraPath, "kantra", "", "Path to kantra binary (default: kantra on PATH)")
	scoreCmd.Flags().StringVar(&scoreProvider, "provider", "", "LLM provider for judge (optional): anthropic, openai, gemini, ollama")
	scoreCmd.Flags().IntVar(&scoreTimeout, "timeout", 900, "Kantra timeout in seconds")

	var pipInput, pipSource, pipTarget, pipLanguage, pipOutput, pipProvider string
	var pipMaxIterations int
	pipelineCmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Run full pipeline: generate rules, test, stamp results, and write rules report",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(pipInput, pipSource, pipTarget, pipLanguage, pipOutput, pipProvider, pipMaxIterations)
		},
	}
	pipelineCmd.Flags().StringVar(&pipInput, "input", "", "URL, file path, or text content to generate rules from")
	pipelineCmd.Flags().StringVar(&pipSource, "source", "", "Source technology (e.g., spring-boot-3)")
	pipelineCmd.Flags().StringVar(&pipTarget, "target", "", "Target technology (e.g., spring-boot-4)")
	pipelineCmd.Flags().StringVar(&pipLanguage, "language", "", "Programming language (java, go, nodejs, csharp)")
	pipelineCmd.Flags().StringVar(&pipOutput, "output", "output", "Output directory path")
	pipelineCmd.Flags().StringVar(&pipProvider, "provider", "", "LLM provider: anthropic, openai, gemini, ollama")
	pipelineCmd.Flags().IntVar(&pipMaxIterations, "max-iterations", 3, "Max test-fix iterations")

	var constInput, constOutput string
	constructCmd := &cobra.Command{
		Use:   "construct",
		Short: "Construct validated rule YAML from JSON input (no LLM needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConstruct(constInput, constOutput)
		},
	}
	constructCmd.Flags().StringVar(&constInput, "input", "-", "Path to JSON input file, or \"-\" for stdin")
	constructCmd.Flags().StringVar(&constOutput, "output", "output", "Output directory path")

	var extInput, extSource, extTarget, extLanguage, extProvider string
	extractCmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract migration patterns from a document and output ConstructInput JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtract(extInput, extSource, extTarget, extLanguage, extProvider)
		},
	}
	extractCmd.Flags().StringVar(&extInput, "input", "", "URL, file path, or text content to extract patterns from")
	extractCmd.Flags().StringVar(&extSource, "source", "", "Source technology (e.g., spring-boot-3)")
	extractCmd.Flags().StringVar(&extTarget, "target", "", "Target technology (e.g., spring-boot-4)")
	extractCmd.Flags().StringVar(&extLanguage, "language", "", "Programming language (java, go, nodejs, csharp)")
	extractCmd.Flags().StringVar(&extProvider, "provider", "", "LLM provider: anthropic, openai, gemini, ollama")

	rootCmd.PersistentFlags().BoolVar(&experimental, "experimental", false, "Enable experimental commands (score)")

	rootCmd.AddCommand(serveCmd, generateCmd, validateCmd, testCmd, pipelineCmd, constructCmd, extractCmd)

	// Experimental commands — hidden unless --experimental is set
	scoreCmd.Hidden = true
	scoreCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if !experimental {
			return fmt.Errorf("'score' is experimental; use --experimental to enable it")
		}
		return nil
	}
	rootCmd.AddCommand(scoreCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	handlers := server.ToolHandlers{
		ConstructRule:    tools.ConstructRuleHandler(),
		ConstructRuleset: tools.ConstructRulesetHandler(),
		ValidateRules:    tools.ValidateRulesHandler(),
		GetHelp:          tools.GetHelpHandler(),
	}

	s := server.New(handlers)

	switch transport {
	case "stdio":
		return server.RunStdio(cmd.Context(), s)
	case "http":
		ctx, cancel := signalContext()
		defer cancel()
		cfg := server.Config{Host: host, Port: port}
		return server.ListenAndServe(ctx, cfg, s)
	default:
		return fmt.Errorf("unknown transport %q; valid: stdio, http", transport)
	}
}

func resolveProvider(flag string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv("RULEGEN_LLM_PROVIDER")
}

// describeInput returns a safe log-friendly description of the input.
// Never logs raw text content — only type and sanitized identifier.
func describeInput(input string) string {
	switch ingestion.DetectType(input) {
	case ingestion.InputURL:
		if u, err := url.Parse(input); err == nil {
			return fmt.Sprintf("url(%s)", u.Host)
		}
		return "url"
	case ingestion.InputFile:
		return fmt.Sprintf("file(%s)", filepath.Base(input))
	default:
		return fmt.Sprintf("text(%d chars)", len(input))
	}
}

func runGenerate(input, source, target, language, output, provider string) error {
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	providerName := resolveProvider(provider)
	completer, err := llm.NewCompleter(providerName)
	if err != nil {
		return fmt.Errorf("LLM configuration error: %v", err)
	}
	if completer == nil {
		return fmt.Errorf("--provider is required (anthropic, openai, gemini, ollama)")
	}

	slog.Info("starting rule generation",
		"input_type", describeInput(input),
		"source", source,
		"target", target,
		"language", language,
		"provider", providerName,
	)

	ctx, cancel := signalContext()
	defer cancel()

	result, err := tools.RunGeneratePipeline(ctx, completer, tools.GenerateInput{
		Input:      input,
		Source:     source,
		Target:     target,
		Language:   language,
		OutputPath: output,
	})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runValidate(rulesPath string) error {
	if rulesPath == "" {
		return fmt.Errorf("--rules is required")
	}

	var ruleList []rules.Rule
	var err error

	info, statErr := os.Stat(rulesPath)
	if statErr != nil {
		return fmt.Errorf("cannot access %s: %w", rulesPath, statErr)
	}
	if info.IsDir() {
		ruleList, err = rules.ReadRulesDir(rulesPath)
	} else {
		ruleList, err = rules.ReadRulesFile(rulesPath)
	}
	if err != nil {
		return err
	}

	result := rules.Validate(ruleList)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling validation result: %w", err)
	}
	fmt.Println(string(data))
	if !result.Valid {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func runConstruct(input, output string) error {
	var data []byte
	var err error

	if input == "-" || input == "" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("reading input file %s: %w", input, err)
		}
	}

	if len(data) == 0 {
		return fmt.Errorf("empty input")
	}

	result, err := tools.ConstructRules(data, output)
	if err != nil {
		if result != nil {
			printResult(result)
		}
		return err
	}

	printResult(result)
	return nil
}

func runExtract(input, source, target, language, provider string) error {
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	providerName := resolveProvider(provider)
	completer, err := llm.NewCompleter(providerName)
	if err != nil {
		return fmt.Errorf("LLM configuration error: %v", err)
	}
	if completer == nil {
		return fmt.Errorf("--provider is required (anthropic, openai, gemini, ollama)")
	}

	slog.Info("starting extraction",
		"input_type", describeInput(input),
		"source", source,
		"target", target,
		"language", language,
		"provider", providerName,
	)

	ctx, cancel := signalContext()
	defer cancel()

	ci, err := tools.RunExtractPipeline(ctx, completer, tools.GenerateInput{
		Input:    input,
		Source:   source,
		Target:   target,
		Language: language,
	})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(ci, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printResult(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func runTest(rulesDir, outputDir, language, source, target, provider string, maxIterations int) error {
	if rulesDir == "" {
		return fmt.Errorf("--rules is required")
	}
	if outputDir == "" {
		// Infer output dir: go up from rules/ to parent
		outputDir = filepath.Dir(filepath.Clean(rulesDir))
	}

	providerName := resolveProvider(provider)
	completer, err := llm.NewCompleter(providerName)
	if err != nil {
		return fmt.Errorf("LLM configuration error: %v", err)
	}
	if completer == nil {
		return fmt.Errorf("--provider is required (anthropic, openai, gemini, ollama)")
	}

	tmpl, err := templates.Load("testing/main.tmpl")
	if err != nil {
		return fmt.Errorf("loading test template: %v", err)
	}

	fixTmpl, err := templates.Load("testing/fix_hint.tmpl")
	if err != nil {
		return fmt.Errorf("loading fix hint template: %v", err)
	}

	ctx, cancel := signalContext()
	defer cancel()

	gen := testgen.New(completer, tmpl)
	fmt.Println("Generating test data and running kantra tests...")
	result, err := gen.RunWithTests(ctx, testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: outputDir,
		Language:  language,
		Source:    source,
		Target:    target,
	}, fixTmpl, maxIterations)
	if err != nil {
		return err
	}

	// Stamp test results back onto rule labels and write rules report
	if result.TestResult != nil {
		if err := stampTestResultsOnFiles(rulesDir, result.TestResult); err != nil {
			slog.Warn("failed to stamp test results on rules", "error", err)
		}

		// Write rules report
		allRules, readErr := rules.ReadRulesDir(rulesDir)
		if readErr != nil {
			slog.Warn("failed to read rules for rules report", "error", readErr)
		} else {
			reportPath := filepath.Join(outputDir, "rules-report.yaml")
			if writeErr := workspace.WriteRulesReport(reportPath, allRules); writeErr != nil {
				slog.Warn("failed to write rules report", "error", writeErr)
			} else {
				slog.Info("rules report written", "path", reportPath)
			}
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling test result: %w", err)
	}
	fmt.Println(string(data))

	// Run bidirectional consistency check: rules ↔ tests
	testsDir := filepath.Join(outputDir, "tests")
	fmt.Println("\nChecking rule ↔ test consistency...")
	consistency, err := rules.ValidateConsistency(rulesDir, testsDir)
	if err != nil {
		fmt.Printf("Warning: consistency check failed: %v\n", err)
	} else {
		cdata, err := json.MarshalIndent(consistency, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling consistency result: %w", err)
		}
		fmt.Println(string(cdata))
	}

	return nil
}

// stampTestResultsOnFiles reads all rule files in a directory, stamps test-result
// labels based on kantra results, and writes the updated rules back.
func stampTestResultsOnFiles(rulesDir string, tr *testgen.TestResult) error {
	failedIDs := make(map[string]bool, len(tr.Failures))
	for _, f := range tr.Failures {
		failedIDs[f.RuleID] = true
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return fmt.Errorf("reading rules dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "ruleset.yaml" || name == "ruleset.yml" {
			continue
		}
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(rulesDir, name)
		ruleList, err := rules.ReadRulesFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}

		// Build passedIDs: any tested rule that isn't in failedIDs
		passedIDs := make(map[string]bool)
		for _, r := range ruleList {
			if r.RuleID != "" && !failedIDs[r.RuleID] {
				passedIDs[r.RuleID] = true
			}
		}

		ruleList = rules.StampTestResults(ruleList, passedIDs, failedIDs)

		if err := rules.WriteRulesFile(path, ruleList); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}

	slog.Info("stamped test results on rules", "passed", tr.Passed, "failed", len(tr.Failures))
	return nil
}

func runScore(testsDir, rulesDir, outputDir, kantraPath, provider string, timeout int) error {
	if testsDir == "" {
		return fmt.Errorf("--tests is required")
	}

	// Optional LLM judge
	var completer llm.Completer
	var judgeTmpl *stdtemplate.Template
	providerName := resolveProvider(provider)
	if providerName != "" {
		var err error
		completer, err = llm.NewCompleter(providerName)
		if err != nil {
			return fmt.Errorf("LLM configuration error: %v", err)
		}
		judgeTmpl, err = templates.Load("confidence/judge.tmpl")
		if err != nil {
			return fmt.Errorf("loading judge template: %v", err)
		}
		if rulesDir == "" {
			return fmt.Errorf("--rules is required when using --provider for LLM judge")
		}
	}

	ctx, cancel := signalContext()
	defer cancel()

	scorer := confidence.New(kantraPath, timeout, completer, judgeTmpl)
	fmt.Println("Running kantra tests and scoring rules...")
	report, err := scorer.ScoreRules(ctx, testsDir, rulesDir)
	if err != nil {
		return err
	}

	// Write scores to file if output dir specified
	if outputDir != "" {
		scoresPath := fmt.Sprintf("%s/confidence/scores.yaml", outputDir)
		if err := os.MkdirAll(fmt.Sprintf("%s/confidence", outputDir), 0o755); err != nil {
			return fmt.Errorf("creating confidence dir: %w", err)
		}
		scoresData, err := yaml.Marshal(report)
		if err != nil {
			return fmt.Errorf("marshaling scores: %w", err)
		}
		if err := os.WriteFile(scoresPath, scoresData, 0o644); err != nil {
			return fmt.Errorf("writing scores: %w", err)
		}
		fmt.Printf("Scores written to %s\n", scoresPath)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling score report: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runPipeline(input, source, target, language, output, provider string, maxIterations int) error {
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	providerName := resolveProvider(provider)
	completer, err := llm.NewCompleter(providerName)
	if err != nil {
		return fmt.Errorf("LLM configuration error: %v", err)
	}
	if completer == nil {
		return fmt.Errorf("--provider is required (anthropic, openai, gemini, ollama)")
	}

	ctx, cancel := signalContext()
	defer cancel()

	// Step 1: Generate rules
	slog.Info("pipeline: generating rules")
	genResult, err := tools.RunGeneratePipeline(ctx, completer, tools.GenerateInput{
		Input:      input,
		Source:     source,
		Target:     target,
		Language:   language,
		OutputPath: output,
	})
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	slog.Info("pipeline: rules generated", "count", genResult.RuleCount, "output", genResult.OutputPath)

	rulesDir := filepath.Join(genResult.OutputPath, "rules")

	// Step 2: Generate test data + run kantra tests
	slog.Info("pipeline: testing rules")
	tmpl, err := templates.Load("testing/main.tmpl")
	if err != nil {
		return fmt.Errorf("loading test template: %v", err)
	}
	fixTmpl, err := templates.Load("testing/fix_hint.tmpl")
	if err != nil {
		return fmt.Errorf("loading fix hint template: %v", err)
	}

	gen := testgen.New(completer, tmpl)
	testResult, err := gen.RunWithTests(ctx, testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: genResult.OutputPath,
		Language:  language,
		Source:    source,
		Target:    target,
	}, fixTmpl, maxIterations)
	if err != nil {
		return fmt.Errorf("test: %w", err)
	}

	// Step 3: Stamp test results on rules
	if testResult.TestResult != nil {
		if err := stampTestResultsOnFiles(rulesDir, testResult.TestResult); err != nil {
			slog.Warn("failed to stamp test results", "error", err)
		}
	}

	// Step 4: Write rules report
	allRules, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		slog.Warn("failed to read rules for rules report", "error", err)
	} else {
		reportPath := filepath.Join(genResult.OutputPath, "rules-report.yaml")
		if err := workspace.WriteRulesReport(reportPath, allRules); err != nil {
			slog.Warn("failed to write rules report", "error", err)
		} else {
			slog.Info("rules report written", "path", reportPath)
		}
	}

	// Print summary
	var testPassed, testTotal int
	if testResult.TestResult != nil {
		testPassed = testResult.TestResult.Passed
		testTotal = testResult.TestResult.Total
	}
	summary := map[string]any{
		"output":      genResult.OutputPath,
		"rules":       genResult.RuleCount,
		"patterns":    genResult.PatternsExtracted,
		"test_passed": testPassed,
		"test_total":  testTotal,
		"iterations":  testResult.Iterations,
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling summary: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
