package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// LanguageConfig defines file structure for a given language.
type LanguageConfig struct {
	BuildFile     string `json:"build_file"`
	BuildFileType string `json:"build_file_type"`
	SourceDir     string `json:"source_dir"`
	MainFile      string `json:"main_file"`
	MainFileType  string `json:"main_file_type"`
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
	AnalysisParams AnalysisParams `yaml:"analysisParams,omitempty"`
	HasIncidents   HasIncidents   `yaml:"hasIncidents"`
}

// AnalysisParams holds analysis parameters for a test case.
type AnalysisParams struct {
	Mode string `yaml:"mode,omitempty"`
}

// HasIncidents specifies expected incident counts.
type HasIncidents struct {
	AtLeast int `yaml:"atLeast"`
}

// Manifest describes the test scaffold output for the agent.
// The agent reads this to know what source files to generate.
type Manifest struct {
	Language string          `json:"language"`
	Groups   []ManifestGroup `json:"groups"`
}

// ManifestGroup is a test group (one per concern/condition-type grouping).
type ManifestGroup struct {
	Name      string         `json:"name"`
	DataDir   string         `json:"data_dir"`
	TestFile  string         `json:"test_file"`
	RuleCount int            `json:"rule_count"`
	Providers []string       `json:"providers"`
	Files     []ManifestFile `json:"files"`
	RuleIDs   []string       `json:"rule_ids"`
}

// ManifestFile is a file the agent needs to generate.
type ManifestFile struct {
	Path     string `json:"path"`
	FileType string `json:"file_type"`
	Purpose  string `json:"purpose"`
}

// Result holds the output of the scaffold operation.
type Result struct {
	Language     string `json:"language"`
	GroupCount   int    `json:"group_count"`
	RuleCount    int    `json:"rule_count"`
	ManifestPath string `json:"manifest_path"`
}

const maxRulesPerGroup = 8

// Run reads rules from rulesDir, creates the test scaffold structure
// (directories, .test.yaml files), and writes a manifest.json for the agent.
func Run(rulesDir, outputDir, language string) (*Result, error) {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("reading rules dir: %w", err)
	}

	testsDir := filepath.Join(outputDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating tests dir: %w", err)
	}

	var manifest Manifest
	totalRules := 0

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "ruleset.yaml" || entry.Name() == "ruleset.yml" {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		concern := strings.TrimSuffix(entry.Name(), ext)
		ruleFilePath := filepath.Join(rulesDir, entry.Name())

		ruleList, err := rules.ReadRulesFile(ruleFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		if len(ruleList) == 0 {
			continue
		}

		// Auto-detect language from rules if not specified
		if language == "" {
			if detected := detectLanguage(ruleList); detected != "" {
				language = detected
			}
		}
		if language == "" {
			language = "java"
		}
		manifest.Language = language

		langConfig, ok := languageConfigs[language]
		if !ok {
			return nil, fmt.Errorf("unsupported language %q", language)
		}

		// Split large rule files into groups of maxRulesPerGroup
		groups := splitRules(ruleList, concern)

		for _, group := range groups {
			dataDir := filepath.Join(testsDir, "data", group.name)
			sourceDir := filepath.Join(dataDir, langConfig.SourceDir)
			if err := os.MkdirAll(sourceDir, 0o755); err != nil {
				return nil, fmt.Errorf("creating data dir: %w", err)
			}

			// Generate .test.yaml
			providers := detectProviders(group.rules)
			testFile := buildTestFile(group.rules, ruleFilePath, dataDir, testsDir, providers)
			testFilePath := filepath.Join(testsDir, group.name+".test.yaml")

			testData, err := yaml.Marshal(testFile)
			if err != nil {
				return nil, fmt.Errorf("marshaling test file: %w", err)
			}
			if err := os.WriteFile(testFilePath, testData, 0o644); err != nil {
				return nil, fmt.Errorf("writing test file: %w", err)
			}

			// Build manifest group
			var ruleIDs []string
			for _, r := range group.rules {
				ruleIDs = append(ruleIDs, r.RuleID)
			}

			relDataDir, _ := filepath.Rel(outputDir, dataDir)
			relTestFile, _ := filepath.Rel(outputDir, testFilePath)

			files := []ManifestFile{
				{
					Path:     filepath.Join(relDataDir, langConfig.BuildFile),
					FileType: langConfig.BuildFileType,
					Purpose:  "build",
				},
				{
					Path:     filepath.Join(relDataDir, langConfig.SourceDir, langConfig.MainFile),
					FileType: langConfig.MainFileType,
					Purpose:  "source",
				},
			}

			// Add extra files for builtin rules that target non-source files
			extraFiles := detectExtraFiles(group.rules, relDataDir, language)
			files = append(files, extraFiles...)

			mg := ManifestGroup{
				Name:      group.name,
				DataDir:   relDataDir,
				TestFile:  relTestFile,
				RuleCount: len(group.rules),
				Providers: providers,
				RuleIDs:   ruleIDs,
				Files:     files,
			}
			manifest.Groups = append(manifest.Groups, mg)
			totalRules += len(group.rules)
		}
	}

	if len(manifest.Groups) == 0 {
		return nil, fmt.Errorf("no rules found in %s", rulesDir)
	}

	// Write manifest.json
	manifestPath := filepath.Join(outputDir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	return &Result{
		Language:     manifest.Language,
		GroupCount:   len(manifest.Groups),
		RuleCount:    totalRules,
		ManifestPath: manifestPath,
	}, nil
}

type ruleGroup struct {
	name  string
	rules []rules.Rule
}

func splitRules(ruleList []rules.Rule, concern string) []ruleGroup {
	if len(ruleList) <= maxRulesPerGroup {
		return []ruleGroup{{name: concern, rules: ruleList}}
	}
	var groups []ruleGroup
	for i := 0; i < len(ruleList); i += maxRulesPerGroup {
		end := i + maxRulesPerGroup
		if end > len(ruleList) {
			end = len(ruleList)
		}
		name := fmt.Sprintf("%s-%d", concern, i/maxRulesPerGroup+1)
		groups = append(groups, ruleGroup{name: name, rules: ruleList[i:end]})
	}
	return groups
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
		tc := TestCase{
			Name: "tc-1",
			HasIncidents: HasIncidents{
				AtLeast: 1,
			},
		}
		if needsSourceOnly(r.When) {
			tc.AnalysisParams = AnalysisParams{Mode: "source-only"}
		}
		tests = append(tests, TestEntry{
			RuleID:    r.RuleID,
			TestCases: []TestCase{tc},
		})
	}

	return TestFile{
		RulesPath: relRulesPath,
		Providers: testProviders,
		Tests:     tests,
	}
}

// needsSourceOnly returns true if the rule uses a *.referenced condition
// (JDTLS/gopls source analysis). Dependency, XML, filecontent, and other
// builtin conditions do NOT use source-only mode.
func needsSourceOnly(c rules.Condition) bool {
	if c.JavaReferenced != nil || c.GoReferenced != nil ||
		c.NodejsReferenced != nil || c.CSharpReferenced != nil {
		return true
	}
	// For or/and combinators, check if ANY child needs source-only
	for _, entry := range c.Or {
		if needsSourceOnly(entry.Condition) {
			return true
		}
	}
	for _, entry := range c.And {
		if needsSourceOnly(entry.Condition) {
			return true
		}
	}
	return false
}

// extraFileMapping maps a filePattern hint (substring in the regex) to
// the extra file that needs to be generated, keyed by language.
// Each entry: hint substring → {language → ManifestFile template}.
// Path is relative to the data dir and joined at runtime.
type extraFileRule struct {
	hint    string                  // substring to look for in filePattern
	perLang map[string]ManifestFile // language → file template (Path is relative)
}

var extraFileRules = []extraFileRule{
	{
		hint: "properties",
		perLang: map[string]ManifestFile{
			"java": {Path: "src/main/resources/application.properties", FileType: "properties", Purpose: "config"},
		},
	},
	{
		hint: "yml",
		perLang: map[string]ManifestFile{
			"java":   {Path: "src/main/resources/application.yml", FileType: "yaml", Purpose: "config"},
			"nodejs": {Path: "config.yml", FileType: "yaml", Purpose: "config"},
			"go":     {Path: "config.yml", FileType: "yaml", Purpose: "config"},
		},
	},
	{
		hint: "gradle",
		perLang: map[string]ManifestFile{
			"java": {Path: "build.gradle", FileType: "gradle", Purpose: "build"},
		},
	},
	{
		hint: ".env",
		perLang: map[string]ManifestFile{
			"nodejs": {Path: ".env", FileType: "env", Purpose: "config"},
		},
	},
	{
		hint: "appsettings",
		perLang: map[string]ManifestFile{
			"csharp": {Path: "appsettings.json", FileType: "json", Purpose: "config"},
		},
	},
}

// detectExtraFiles inspects rules for builtin.filecontent conditions
// and adds extra files to the manifest when the filePattern targets
// files beyond the standard build + source pair.
func detectExtraFiles(ruleList []rules.Rule, relDataDir, language string) []ManifestFile {
	needed := make(map[string]bool) // keyed by relative path to dedup

	for _, r := range ruleList {
		collectExtraHints(r.When, &needed, language)
	}

	var extras []ManifestFile
	for path := range needed {
		// Find the matching rule to get FileType and Purpose
		for _, efr := range extraFileRules {
			if mf, ok := efr.perLang[language]; ok && mf.Path == path {
				extras = append(extras, ManifestFile{
					Path:     filepath.Join(relDataDir, mf.Path),
					FileType: mf.FileType,
					Purpose:  mf.Purpose,
				})
				break
			}
		}
	}
	return extras
}

func collectExtraHints(c rules.Condition, needed *map[string]bool, language string) {
	if c.BuiltinFilecontent != nil {
		fp := c.BuiltinFilecontent.FilePattern
		for _, efr := range extraFileRules {
			if strings.Contains(fp, efr.hint) {
				if mf, ok := efr.perLang[language]; ok {
					(*needed)[mf.Path] = true
				}
			}
		}
	}
	for _, entry := range c.Or {
		collectExtraHints(entry.Condition, needed, language)
	}
	for _, entry := range c.And {
		collectExtraHints(entry.Condition, needed, language)
	}
}

// GetLanguageConfig returns the LanguageConfig for a given language.
func GetLanguageConfig(language string) (LanguageConfig, bool) {
	cfg, ok := languageConfigs[language]
	return cfg, ok
}
