package testgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// LanguageConfig defines file structure for a given language.
type LanguageConfig struct {
	BuildFile     string
	BuildFileType string
	SourceDir     string
	MainFile      string
	MainFileType  string
}

var languageConfigs = map[string]LanguageConfig{
	"java": {
		BuildFile:     "pom.xml",
		BuildFileType: "xml",
		SourceDir:     "src/main/java/com/example",
		MainFile:      "Application.java",
		MainFileType:  "java",
	},
	"go": {
		BuildFile:     "go.mod",
		BuildFileType: "go",
		SourceDir:     ".",
		MainFile:      "main.go",
		MainFileType:  "go",
	},
	"nodejs": {
		BuildFile:     "package.json",
		BuildFileType: "json",
		SourceDir:     "src",
		MainFile:      "App.tsx",
		MainFileType:  "tsx",
	},
	"csharp": {
		BuildFile:     "Project.csproj",
		BuildFileType: "xml",
		SourceDir:     ".",
		MainFile:      "Program.cs",
		MainFileType:  "csharp",
	},
}

// TestFile represents a kantra .test.yaml file.
type TestFile struct {
	RulesPath string         `yaml:"rulesPath"`
	Providers []TestProvider `yaml:"providers"`
	Tests     []TestEntry    `yaml:"tests"`
}

// TestProvider is a provider entry in .test.yaml.
type TestProvider struct {
	Name     string `yaml:"name"`
	DataPath string `yaml:"dataPath"`
}

// TestEntry is a test entry for a single rule.
type TestEntry struct {
	RuleID    string     `yaml:"ruleID"`
	TestCases []TestCase `yaml:"testCases"`
}

// TestCase is a single test case within a test entry.
type TestCase struct {
	Name           string         `yaml:"name"`
	AnalysisParams AnalysisParams `yaml:"analysisParams"`
	HasIncidents   HasIncidents   `yaml:"hasIncidents"`
}

// AnalysisParams holds analysis parameters for a test case.
type AnalysisParams struct {
	Mode string `yaml:"mode"`
}

// HasIncidents specifies expected incident counts.
type HasIncidents struct {
	AtLeast int `yaml:"atLeast"`
}

// GenerateInput holds parameters for test data generation.
type GenerateInput struct {
	RulesDir   string
	OutputDir  string
	Language   string
	Source     string
	Target     string
	MaxRetries int
}

// GenerateOutput holds results from test data generation.
type GenerateOutput struct {
	TestFiles    []string `json:"test_files"`
	DataDirs     []string `json:"data_dirs"`
	RulesTested  int      `json:"rules_tested"`
	FilesWritten int      `json:"files_written"`
}

// Generator creates test data from generated rules.
type Generator struct {
	completer llm.Completer
	tmpl      *template.Template
}

// New creates a test data Generator.
func New(completer llm.Completer, tmpl *template.Template) *Generator {
	return &Generator{completer: completer, tmpl: tmpl}
}

// Generate creates test data for all rule files in a rules directory.
func (g *Generator) Generate(ctx context.Context, input GenerateInput) (*GenerateOutput, error) {
	// Read all rule files
	entries, err := os.ReadDir(input.RulesDir)
	if err != nil {
		return nil, fmt.Errorf("reading rules dir: %w", err)
	}

	language := input.Language
	if language == "" {
		language = "java" // default
	}
	langConfig, ok := languageConfigs[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language %q", language)
	}

	testsDir := filepath.Join(input.OutputDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating tests dir: %w", err)
	}

	var output GenerateOutput

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "ruleset.yaml" || entry.Name() == "ruleset.yml" {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		concern := strings.TrimSuffix(entry.Name(), ext)
		ruleFilePath := filepath.Join(input.RulesDir, entry.Name())

		ruleList, err := rules.ReadRulesFile(ruleFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		if len(ruleList) == 0 {
			continue
		}

		// Detect language from rules if not provided
		if input.Language == "" {
			detected := detectLanguage(ruleList)
			if detected != "" {
				language = detected
				if cfg, ok := languageConfigs[language]; ok {
					langConfig = cfg
				}
			}
		}

		// Generate test data for this concern
		dataDir := filepath.Join(testsDir, "data", concern)
		if err := os.MkdirAll(filepath.Join(dataDir, langConfig.SourceDir), 0o755); err != nil {
			return nil, fmt.Errorf("creating data dir: %w", err)
		}

		fmt.Printf("  Generating test data for %s (%d rules)...\n", concern, len(ruleList))

		buildContent, sourceContent, err := g.generateCode(ctx, ruleList, language, langConfig, input.Source, input.Target)
		if err != nil {
			return nil, fmt.Errorf("generating test code for %s: %w", concern, err)
		}

		// Write build file
		buildPath := filepath.Join(dataDir, langConfig.BuildFile)
		if err := os.WriteFile(buildPath, []byte(buildContent), 0o644); err != nil {
			return nil, fmt.Errorf("writing build file: %w", err)
		}
		output.FilesWritten++

		// Write source file
		sourcePath := filepath.Join(dataDir, langConfig.SourceDir, langConfig.MainFile)
		if err := os.WriteFile(sourcePath, []byte(sourceContent), 0o644); err != nil {
			return nil, fmt.Errorf("writing source file: %w", err)
		}
		output.FilesWritten++

		// Resolve dependencies after writing files
		fmt.Printf("    Resolving dependencies in %s...\n", dataDir)
		runDepResolve(language, dataDir)

		// Generate .test.yaml
		providers := detectProviders(ruleList)
		testFile := buildTestFile(ruleList, ruleFilePath, dataDir, testsDir, providers)
		testFilePath := filepath.Join(testsDir, concern+".test.yaml")

		testData, err := yaml.Marshal(testFile)
		if err != nil {
			return nil, fmt.Errorf("marshaling test file: %w", err)
		}
		if err := os.WriteFile(testFilePath, testData, 0o644); err != nil {
			return nil, fmt.Errorf("writing test file: %w", err)
		}
		output.FilesWritten++

		output.TestFiles = append(output.TestFiles, concern+".test.yaml")
		output.DataDirs = append(output.DataDirs, filepath.Join("data", concern))
		output.RulesTested += len(ruleList)
	}

	return &output, nil
}

// generateCode calls the LLM to produce build file and source file content.
func (g *Generator) generateCode(ctx context.Context, ruleList []rules.Rule, language string, langConfig LanguageConfig, source, target string) (buildContent, sourceContent string, err error) {
	// Build the prompt
	rulesJSON, _ := json.MarshalIndent(ruleList, "", "  ")

	var buf bytes.Buffer
	data := map[string]string{
		"Language":      language,
		"Source":        source,
		"Target":        target,
		"BuildFile":     langConfig.BuildFile,
		"BuildFileType": langConfig.BuildFileType,
		"MainFile":      langConfig.MainFile,
		"MainFileType":  langConfig.MainFileType,
		"SourceDir":     langConfig.SourceDir,
		"Rules":         string(rulesJSON),
	}
	if err := g.tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("rendering template: %w", err)
	}

	response, err := g.completer.Complete(ctx, buf.String())
	if err != nil {
		return "", "", fmt.Errorf("LLM generation: %w", err)
	}

	// Extract code blocks from response
	blocks := extractCodeBlocks(response)
	if len(blocks) < 2 {
		return "", "", fmt.Errorf("expected at least 2 code blocks, got %d", len(blocks))
	}

	return blocks[0].Content, blocks[1].Content, nil
}

// CodeBlock represents a fenced code block extracted from LLM response.
type CodeBlock struct {
	Language string
	Content  string
}

// extractCodeBlocks parses fenced code blocks from markdown response.
func extractCodeBlocks(response string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(response, "\n")
	var current *CodeBlock

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && current == nil {
			lang := strings.TrimPrefix(trimmed, "```")
			lang = strings.TrimSpace(lang)
			current = &CodeBlock{Language: lang}
		} else if strings.HasPrefix(trimmed, "```") && current != nil {
			blocks = append(blocks, *current)
			current = nil
		} else if current != nil {
			if current.Content != "" {
				current.Content += "\n"
			}
			current.Content += line
		}
	}

	return blocks
}

// detectLanguage infers the programming language from rule conditions.
func detectLanguage(ruleList []rules.Rule) string {
	for _, r := range ruleList {
		if lang := detectConditionLanguage(r.When); lang != "" {
			return lang
		}
	}
	return ""
}

func detectConditionLanguage(c rules.Condition) string {
	if c.JavaReferenced != nil || c.JavaDependency != nil {
		return "java"
	}
	if c.GoReferenced != nil || c.GoDependency != nil {
		return "go"
	}
	if c.NodejsReferenced != nil {
		return "nodejs"
	}
	if c.CSharpReferenced != nil {
		return "csharp"
	}
	if c.BuiltinFilecontent != nil {
		fp := c.BuiltinFilecontent.FilePattern
		switch {
		case strings.Contains(fp, ".java"):
			return "java"
		case strings.Contains(fp, ".go"):
			return "go"
		case strings.Contains(fp, ".ts") || strings.Contains(fp, ".tsx") || strings.Contains(fp, ".js"):
			return "nodejs"
		case strings.Contains(fp, ".cs"):
			return "csharp"
		}
	}
	// Check or/and combinators
	for _, entry := range c.Or {
		if lang := detectConditionLanguage(entry.Condition); lang != "" {
			return lang
		}
	}
	for _, entry := range c.And {
		if lang := detectConditionLanguage(entry.Condition); lang != "" {
			return lang
		}
	}
	return ""
}

// detectProviders returns the unique provider names used in a rule list.
func detectProviders(ruleList []rules.Rule) []string {
	seen := make(map[string]bool)
	for _, r := range ruleList {
		for _, p := range conditionProviders(r.When) {
			seen[p] = true
		}
	}
	var providers []string
	for p := range seen {
		providers = append(providers, p)
	}
	if len(providers) == 0 {
		providers = []string{"builtin"}
	}
	return providers
}

func conditionProviders(c rules.Condition) []string {
	var providers []string
	if c.JavaReferenced != nil || c.JavaDependency != nil {
		providers = append(providers, "java")
	}
	if c.GoReferenced != nil || c.GoDependency != nil {
		providers = append(providers, "go")
	}
	if c.NodejsReferenced != nil {
		providers = append(providers, "nodejs")
	}
	if c.CSharpReferenced != nil {
		providers = append(providers, "dotnet")
	}
	if c.BuiltinFilecontent != nil || c.BuiltinFile != nil || c.BuiltinXML != nil ||
		c.BuiltinJSON != nil || c.BuiltinXMLPublicID != nil || len(c.BuiltinHasTags) > 0 {
		providers = append(providers, "builtin")
	}
	for _, entry := range c.Or {
		providers = append(providers, conditionProviders(entry.Condition)...)
	}
	for _, entry := range c.And {
		providers = append(providers, conditionProviders(entry.Condition)...)
	}
	return providers
}

// buildTestFile creates a .test.yaml structure for kantra.
func buildTestFile(ruleList []rules.Rule, ruleFilePath, dataDir, testsDir string, providers []string) TestFile {
	// Compute relative paths from test file location
	relRulesPath, _ := filepath.Rel(testsDir, ruleFilePath)
	relDataPath, _ := filepath.Rel(testsDir, dataDir)

	var testProviders []TestProvider
	for _, p := range providers {
		testProviders = append(testProviders, TestProvider{
			Name:     p,
			DataPath: "./" + relDataPath,
		})
	}

	var tests []TestEntry
	for _, r := range ruleList {
		tests = append(tests, TestEntry{
			RuleID: r.RuleID,
			TestCases: []TestCase{
				{
					Name: "tc-1",
					AnalysisParams: AnalysisParams{
						Mode: "source-only",
					},
					HasIncidents: HasIncidents{
						AtLeast: 1,
					},
				},
			},
		})
	}

	return TestFile{
		RulesPath: relRulesPath,
		Providers: testProviders,
		Tests:     tests,
	}
}

// runDepResolve runs the language-appropriate dependency resolution command.
// For Go, it also vendors dependencies so gopls inside kantra's container can resolve them.
func runDepResolve(language, dir string) {
	switch language {
	case "go":
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = dir
		if out, err := tidyCmd.CombinedOutput(); err != nil {
			fmt.Printf("    Warning: go mod tidy failed: %v\n%s\n", err, string(out))
			return
		}
		vendorCmd := exec.Command("go", "mod", "vendor")
		vendorCmd.Dir = dir
		if out, err := vendorCmd.CombinedOutput(); err != nil {
			fmt.Printf("    Warning: go mod vendor failed: %v\n%s\n", err, string(out))
		}
	case "java":
		cmd := exec.Command("mvn", "dependency:resolve", "-q", "-B")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("    Warning: dependency resolution failed: %v\n%s\n", err, string(out))
		}
	case "nodejs":
		cmd := exec.Command("npm", "install")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("    Warning: dependency resolution failed: %v\n%s\n", err, string(out))
		}
	case "csharp":
		cmd := exec.Command("dotnet", "restore")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("    Warning: dependency resolution failed: %v\n%s\n", err, string(out))
		}
	}
}
