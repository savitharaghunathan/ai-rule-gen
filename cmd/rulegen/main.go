package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/konveyor/ai-rule-gen/internal/confidence"
	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/server"
	"github.com/konveyor/ai-rule-gen/internal/testgen"
	"github.com/konveyor/ai-rule-gen/internal/tools"
	"github.com/konveyor/ai-rule-gen/templates"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	host string
	port int
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
	serveCmd.Flags().StringVar(&host, "host", "localhost", "Host to bind to")
	serveCmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")

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
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Generate test data for rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(testRulesDir, testOutputDir, testLanguage, testSource, testTarget, testProvider)
		},
	}
	testCmd.Flags().StringVar(&testRulesDir, "rules", "", "Path to rules directory")
	testCmd.Flags().StringVar(&testOutputDir, "output", "", "Output directory (parent of rules/)")
	testCmd.Flags().StringVar(&testLanguage, "language", "", "Programming language (auto-detected if omitted)")
	testCmd.Flags().StringVar(&testSource, "source", "", "Source technology")
	testCmd.Flags().StringVar(&testTarget, "target", "", "Target technology")
	testCmd.Flags().StringVar(&testProvider, "provider", "", "LLM provider: anthropic, openai, gemini, ollama")

	var scoreRulesDir, scoreOutputDir, scoreProvider string
	scoreCmd := &cobra.Command{
		Use:   "score",
		Short: "Score confidence on generated rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScore(scoreRulesDir, scoreOutputDir, scoreProvider)
		},
	}
	scoreCmd.Flags().StringVar(&scoreRulesDir, "rules", "", "Path to rules directory")
	scoreCmd.Flags().StringVar(&scoreOutputDir, "output", "", "Output directory for scores")
	scoreCmd.Flags().StringVar(&scoreProvider, "provider", "", "LLM provider: anthropic, openai, gemini, ollama")

	rootCmd.AddCommand(serveCmd, generateCmd, validateCmd, testCmd, scoreCmd)

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
	cfg := server.Config{Host: host, Port: port}
	return server.ListenAndServe(cfg, s)
}

func runGenerate(input, source, target, language, output, provider string) error {
	if input == "" {
		return fmt.Errorf("--input is required")
	}

	// --provider flag overrides env var
	if provider != "" {
		os.Setenv("RULEGEN_LLM_PROVIDER", provider)
	}

	completer, err := llm.NewCompleterFromEnv()
	if err != nil {
		return fmt.Errorf("LLM configuration error: %v", err)
	}
	if completer == nil {
		return fmt.Errorf("--provider is required (anthropic, openai, gemini, ollama)")
	}

	providerName := os.Getenv("RULEGEN_LLM_PROVIDER")
	slog.Info("starting rule generation",
		"input", input,
		"source", source,
		"target", target,
		"language", language,
		"provider", providerName,
	)

	result, err := tools.RunGeneratePipeline(context.Background(), completer, tools.GenerateInput{
		Input:      input,
		Source:     source,
		Target:     target,
		Language:   language,
		OutputPath: output,
	})
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(result, "", "  ")
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
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	if !result.Valid {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func runTest(rulesDir, outputDir, language, source, target, provider string) error {
	if rulesDir == "" {
		return fmt.Errorf("--rules is required")
	}
	if outputDir == "" {
		// Infer output dir: go up from rules/ to parent
		outputDir = fmt.Sprintf("%s/..", rulesDir)
	}
	if provider != "" {
		os.Setenv("RULEGEN_LLM_PROVIDER", provider)
	}

	completer, err := llm.NewCompleterFromEnv()
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

	gen := testgen.New(completer, tmpl)
	fmt.Println("Generating test data...")
	result, err := gen.Generate(context.Background(), testgen.GenerateInput{
		RulesDir:  rulesDir,
		OutputDir: outputDir,
		Language:  language,
		Source:    source,
		Target:    target,
	})
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return nil
}

func runScore(rulesDir, outputDir, provider string) error {
	if rulesDir == "" {
		return fmt.Errorf("--rules is required")
	}
	if provider != "" {
		os.Setenv("RULEGEN_LLM_PROVIDER", provider)
	}

	completer, err := llm.NewCompleterFromEnv()
	if err != nil {
		return fmt.Errorf("LLM configuration error: %v", err)
	}
	if completer == nil {
		return fmt.Errorf("--provider is required (anthropic, openai, gemini, ollama)")
	}

	ruleList, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		return err
	}
	if len(ruleList) == 0 {
		return fmt.Errorf("no rules found in %s", rulesDir)
	}

	tmpl, err := templates.Load("confidence/judge.tmpl")
	if err != nil {
		return fmt.Errorf("loading judge template: %v", err)
	}

	scorer := confidence.New(completer, tmpl)
	fmt.Printf("Scoring %d rules...\n", len(ruleList))
	report, err := scorer.ScoreRules(context.Background(), ruleList)
	if err != nil {
		return err
	}

	// Write scores to file if output dir specified
	if outputDir != "" {
		scoresPath := fmt.Sprintf("%s/confidence/scores.yaml", outputDir)
		if err := os.MkdirAll(fmt.Sprintf("%s/confidence", outputDir), 0o755); err != nil {
			return fmt.Errorf("creating confidence dir: %w", err)
		}
		scoresData, _ := yaml.Marshal(report)
		if err := os.WriteFile(scoresPath, scoresData, 0o644); err != nil {
			return fmt.Errorf("writing scores: %w", err)
		}
		fmt.Printf("Scores written to %s\n", scoresPath)
	}

	data, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(data))
	return nil
}
