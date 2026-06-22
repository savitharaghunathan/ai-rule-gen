package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// setupTestLanguagesDir creates a temporary languages/ directory with config.json
// files for all supported languages, matching the real languages/ directory structure.
func setupTestLanguagesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "languages")

	configs := map[string]languageFile{
		"java": {Language: "java", Providers: []string{"java", "builtin"}, Scaffold: LanguageConfig{
			BuildFile: "pom.xml", BuildFileType: "xml",
			SourceDir: "src/main/java/com/example", MainFile: "Application.java", MainFileType: "java",
			TestSourceDir: "src/test/java/com/example", TestMainFile: "ApplicationTest.java",
		}},
		"go": {Language: "go", Providers: []string{"go", "builtin"}, Scaffold: LanguageConfig{
			BuildFile: "go.mod", BuildFileType: "go",
			SourceDir: ".", MainFile: "main.go", MainFileType: "go",
		}},
		"nodejs": {Language: "nodejs", Providers: []string{"nodejs", "builtin"}, Scaffold: LanguageConfig{
			BuildFile: "package.json", BuildFileType: "json",
			SourceDir: "src", MainFile: "App.tsx", MainFileType: "tsx",
		}},
		"csharp": {Language: "csharp", Providers: []string{"dotnet", "builtin"}, Scaffold: LanguageConfig{
			BuildFile: "Project.csproj", BuildFileType: "xml",
			SourceDir: ".", MainFile: "Program.cs", MainFileType: "csharp",
		}},
		"python": {Language: "python", Providers: []string{"python", "builtin"}, Scaffold: LanguageConfig{
			BuildFile: "requirements.txt", BuildFileType: "text",
			SourceDir: ".", MainFile: "main.py", MainFileType: "python",
		}},
	}

	for lang, cfg := range configs {
		langDir := filepath.Join(dir, lang)
		os.MkdirAll(langDir, 0o755)
		data, _ := json.Marshal(cfg)
		os.WriteFile(filepath.Join(langDir, "config.json"), data, 0o644)
	}
	return dir
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		rules    []rules.Rule
		expected string
	}{
		{
			name: "java referenced",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION"),
			}},
			expected: "java",
		},
		{
			name: "go referenced",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewGoReferenced("golang.org/x/crypto/md4"),
			}},
			expected: "go",
		},
		{
			name: "nodejs referenced",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewNodejsReferenced("express.Router"),
			}},
			expected: "nodejs",
		},
		{
			name: "builtin with java file pattern",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewBuiltinFilecontent("pattern", "*.java"),
			}},
			expected: "java",
		},
		{
			name: "python referenced",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewPythonReferenced("flask.Flask"),
			}},
			expected: "python",
		},
		{
			name: "csharp referenced",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewCSharpReferenced("System.Web.HttpContext", ""),
			}},
			expected: "csharp",
		},
		{
			name: "builtin with python file pattern",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewBuiltinFilecontent("pattern", "*.py"),
			}},
			expected: "python",
		},
		{
			name:     "no rules",
			rules:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLanguage(tt.rules)
			if got != tt.expected {
				t.Errorf("detectLanguage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectProviders(t *testing.T) {
	ruleList := []rules.Rule{
		{RuleID: "r1", When: rules.NewJavaReferenced("javax.ejb", "TYPE")},
		{RuleID: "r2", When: rules.NewBuiltinFilecontent("pattern", "*.xml")},
	}
	providers := detectProviders(ruleList)
	hasJava := false
	hasBuiltin := false
	for _, p := range providers {
		if p == "java" {
			hasJava = true
		}
		if p == "builtin" {
			hasBuiltin = true
		}
	}
	if !hasJava {
		t.Error("expected java provider")
	}
	if !hasBuiltin {
		t.Error("expected builtin provider")
	}
}

func TestSplitRules(t *testing.T) {
	t.Run("small group stays together", func(t *testing.T) {
		ruleList := make([]rules.Rule, 5)
		for i := range ruleList {
			ruleList[i] = rules.Rule{RuleID: "r" + string(rune('0'+i))}
		}
		groups := splitRules(ruleList, "web")
		if len(groups) != 1 {
			t.Fatalf("got %d groups, want 1", len(groups))
		}
		if groups[0].name != "web" {
			t.Errorf("name = %q, want %q", groups[0].name, "web")
		}
	})

	t.Run("large group splits", func(t *testing.T) {
		ruleList := make([]rules.Rule, 20)
		for i := range ruleList {
			ruleList[i] = rules.Rule{RuleID: "r" + string(rune('0'+i))}
		}
		groups := splitRules(ruleList, "web")
		if len(groups) != 3 {
			t.Fatalf("got %d groups, want 3", len(groups))
		}
		if groups[0].name != "web-1" {
			t.Errorf("group[0].name = %q, want %q", groups[0].name, "web-1")
		}
		if len(groups[0].rules) != 8 {
			t.Errorf("group[0] has %d rules, want 8", len(groups[0].rules))
		}
		if len(groups[2].rules) != 4 {
			t.Errorf("group[2] has %d rules, want 4", len(groups[2].rules))
		}
	})
}

func TestSplitRules_MixedProviders(t *testing.T) {
	builtinRule := rules.Rule{RuleID: "r1", When: rules.NewBuiltinFilecontent("pattern", "*.xml")}
	xmlRule := rules.Rule{RuleID: "r2", When: rules.Condition{BuiltinXML: &rules.BuiltinXML{XPath: "/project"}}}
	javaDepRule := rules.Rule{RuleID: "r3", When: rules.NewJavaDependency("org.example.foo", "0.0.0", "")}
	javaRefRule := rules.Rule{RuleID: "r4", When: rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION")}

	t.Run("builtin and java rules split into separate groups", func(t *testing.T) {
		ruleList := []rules.Rule{builtinRule, xmlRule, javaDepRule}
		groups := splitRules(ruleList, "build")
		if len(groups) != 2 {
			t.Fatalf("got %d groups, want 2", len(groups))
		}
		if len(groups[0].rules) != 2 {
			t.Errorf("builtin group has %d rules, want 2", len(groups[0].rules))
		}
		if len(groups[1].rules) != 1 {
			t.Errorf("java group has %d rules, want 1", len(groups[1].rules))
		}
	})

	t.Run("same-provider rules stay together", func(t *testing.T) {
		ruleList := []rules.Rule{javaDepRule, javaRefRule}
		groups := splitRules(ruleList, "core")
		if len(groups) != 1 {
			t.Fatalf("got %d groups, want 1", len(groups))
		}
		if groups[0].name != "core" {
			t.Errorf("name = %q, want %q", groups[0].name, "core")
		}
	})
}

func TestBuildTestFile(t *testing.T) {
	ruleList := []rules.Rule{
		{RuleID: "rule-00010", When: rules.NewJavaReferenced("javax.ejb", "TYPE")},
		{RuleID: "rule-00020", When: rules.NewJavaReferenced("javax.ws.rs", "IMPORT")},
	}
	tf := buildTestFile(ruleList, "/output/rules/web.yaml", "/output/tests/data/web", "/output/tests", []string{"java"})

	if tf.RulesPath != "../rules/web.yaml" {
		t.Errorf("RulesPath = %q, want %q", tf.RulesPath, "../rules/web.yaml")
	}
	if len(tf.Providers) != 1 || tf.Providers[0].Name != "java" {
		t.Errorf("Providers = %v, want [{java ./data/web}]", tf.Providers)
	}
	if len(tf.Tests) != 2 {
		t.Fatalf("got %d tests, want 2", len(tf.Tests))
	}
	if tf.Tests[0].RuleID != "rule-00010" {
		t.Errorf("Tests[0].RuleID = %q, want %q", tf.Tests[0].RuleID, "rule-00010")
	}
}

func TestRun(t *testing.T) {
	// Set up a rules directory with a rule file and ruleset.yaml
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	ruleList := []rules.Rule{
		{
			RuleID:  "java-ee-00010",
			Message: "test",
			When:    rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION"),
		},
		{
			RuleID:  "java-ee-00020",
			Message: "test2",
			When:    rules.NewJavaReferenced("javax.ws.rs.Path", "ANNOTATION"),
		},
	}
	ruleData, _ := yaml.Marshal(ruleList)
	os.WriteFile(filepath.Join(rulesDir, "ejb.yaml"), ruleData, 0o644)

	// Write a ruleset.yaml that should be skipped
	rsData, _ := yaml.Marshal(rules.Ruleset{Name: "test"})
	os.WriteFile(filepath.Join(rulesDir, "ruleset.yaml"), rsData, 0o644)

	langsDir := setupTestLanguagesDir(t)
	result, err := Run(rulesDir, dir, "", langsDir)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Language != "java" {
		t.Errorf("Language = %q, want %q", result.Language, "java")
	}
	if result.RuleCount != 2 {
		t.Errorf("RuleCount = %d, want 2", result.RuleCount)
	}
	if result.GroupCount != 1 {
		t.Errorf("GroupCount = %d, want 1", result.GroupCount)
	}

	// Verify manifest.json was written
	manifestData, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parsing manifest: %v", err)
	}
	if len(manifest.Groups) != 1 {
		t.Fatalf("manifest has %d groups, want 1", len(manifest.Groups))
	}
	if len(manifest.Groups[0].Files) != 2 {
		t.Errorf("group has %d files, want 2", len(manifest.Groups[0].Files))
	}
	if len(manifest.Groups[0].RuleIDs) != 2 {
		t.Errorf("group has %d ruleIDs, want 2", len(manifest.Groups[0].RuleIDs))
	}

	// Verify .test.yaml was written
	testFilePath := filepath.Join(dir, "ejb.test.yaml")
	testData, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}
	var tf TestFile
	if err := yaml.Unmarshal(testData, &tf); err != nil {
		t.Fatalf("parsing test file: %v", err)
	}
	if len(tf.Tests) != 2 {
		t.Errorf("test file has %d tests, want 2", len(tf.Tests))
	}

	// Verify data directory was created
	dataDir := filepath.Join(dir, "data", "ejb", "src", "main", "java", "com", "example")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Errorf("data directory %s was not created", dataDir)
	}
}

func TestDetectExtraFiles(t *testing.T) {
	t.Run("properties rule adds application.properties", func(t *testing.T) {
		ruleList := []rules.Rule{
			{RuleID: "r1", When: rules.NewBuiltinFilecontent("spring\\.jpa", `application.*\.properties`)},
		}
		extras := detectExtraFiles(ruleList, "tests/data/config", "java")
		found := false
		for _, f := range extras {
			if f.FileType == "properties" && f.Purpose == "config" {
				found = true
			}
		}
		if !found {
			t.Error("expected application.properties in extra files")
		}
	})

	t.Run("gradle rule adds build.gradle", func(t *testing.T) {
		ruleList := []rules.Rule{
			{RuleID: "r1", When: rules.NewBuiltinFilecontent("loaderImplementation", `.*\.gradle`)},
		}
		extras := detectExtraFiles(ruleList, "tests/data/build", "java")
		found := false
		for _, f := range extras {
			if f.FileType == "gradle" && f.Purpose == "build" {
				found = true
			}
		}
		if !found {
			t.Error("expected build.gradle in extra files")
		}
	})

	t.Run("csharp appsettings rule adds appsettings.json", func(t *testing.T) {
		ruleList := []rules.Rule{
			{RuleID: "r1", When: rules.NewBuiltinFilecontent("ConnectionString", `appsettings.*\.json`)},
		}
		extras := detectExtraFiles(ruleList, "tests/data/config", "csharp")
		found := false
		for _, f := range extras {
			if f.FileType == "json" && f.Purpose == "config" {
				found = true
			}
		}
		if !found {
			t.Error("expected appsettings.json in extra files for csharp")
		}
	})

	t.Run("nodejs env rule adds .env", func(t *testing.T) {
		ruleList := []rules.Rule{
			{RuleID: "r1", When: rules.NewBuiltinFilecontent("DB_HOST", `\.env`)},
		}
		extras := detectExtraFiles(ruleList, "tests/data/config", "nodejs")
		found := false
		for _, f := range extras {
			if f.FileType == "env" && f.Purpose == "config" {
				found = true
			}
		}
		if !found {
			t.Error("expected .env in extra files for nodejs")
		}
	})

	t.Run("java.referenced rule adds no extras", func(t *testing.T) {
		ruleList := []rules.Rule{
			{RuleID: "r1", When: rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION")},
		}
		extras := detectExtraFiles(ruleList, "tests/data/core", "java")
		if len(extras) != 0 {
			t.Errorf("expected 0 extras, got %d", len(extras))
		}
	})
}

func TestIsTestRelatedGroup(t *testing.T) {
	tests := []struct {
		name     string
		rules    []rules.Rule
		expected bool
	}{
		{
			name: "MockBean annotation is test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaReferenced("org.springframework.boot.test.mock.mockito.MockBean", "ANNOTATION"),
			}},
			expected: true,
		},
		{
			name: "JUnit import is test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaReferenced("org.junit.jupiter.api.Test", "ANNOTATION"),
			}},
			expected: true,
		},
		{
			name: "MockMvc import is test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaReferenced("org.springframework.test.web.servlet.MockMvc", "IMPORT"),
			}},
			expected: true,
		},
		{
			name: "spring-boot-starter-test dependency is test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaDependency("org.springframework.boot.spring-boot-starter-test", "0.0.0", ""),
			}},
			expected: true,
		},
		{
			name: "spock dependency is test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaDependency("org.spockframework.spock-spring", "0.0.0", ""),
			}},
			expected: true,
		},
		{
			name: "EJB annotation is not test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION"),
			}},
			expected: false,
		},
		{
			name: "spring-web dependency is not test-related",
			rules: []rules.Rule{{
				RuleID: "r1",
				When:   rules.NewJavaDependency("org.springframework.boot.spring-boot-starter-web", "0.0.0", ""),
			}},
			expected: false,
		},
		{
			name: "or combinator with test-related child",
			rules: []rules.Rule{{
				RuleID: "r1",
				When: rules.NewOr(
					rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION"),
					rules.NewJavaReferenced("org.mockito.Mock", "ANNOTATION"),
				),
			}},
			expected: true,
		},
		{
			name: "and combinator with test-related child",
			rules: []rules.Rule{{
				RuleID: "r1",
				When: rules.Condition{
					And: []rules.ConditionEntry{
						{Condition: rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION")},
						{Condition: rules.NewJavaReferenced("org.mockito.Mock", "ANNOTATION")},
					},
				},
			}},
			expected: true,
		},
		{
			name: "and combinator without test-related child",
			rules: []rules.Rule{{
				RuleID: "r1",
				When: rules.Condition{
					And: []rules.ConditionEntry{
						{Condition: rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION")},
						{Condition: rules.NewJavaReferenced("javax.inject.Inject", "ANNOTATION")},
					},
				},
			}},
			expected: false,
		},
		{
			name:     "empty rule list",
			rules:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTestRelatedGroup(tt.rules)
			if got != tt.expected {
				t.Errorf("isTestRelatedGroup() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRunWithTestRelatedRules(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	ruleList := []rules.Rule{
		{
			RuleID:  "test-00010",
			Message: "MockBean removed",
			When:    rules.NewJavaReferenced("org.springframework.boot.test.mock.mockito.MockBean", "ANNOTATION"),
		},
		{
			RuleID:  "test-00020",
			Message: "SpyBean removed",
			When:    rules.NewJavaReferenced("org.springframework.boot.test.mock.mockito.SpyBean", "ANNOTATION"),
		},
	}
	ruleData, _ := yaml.Marshal(ruleList)
	os.WriteFile(filepath.Join(rulesDir, "testing.yaml"), ruleData, 0o644)

	rsData, _ := yaml.Marshal(rules.Ruleset{Name: "test"})
	os.WriteFile(filepath.Join(rulesDir, "ruleset.yaml"), rsData, 0o644)

	langsDir := setupTestLanguagesDir(t)
	result, err := Run(rulesDir, dir, "", langsDir)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.Language != "java" {
		t.Errorf("Language = %q, want %q", result.Language, "java")
	}

	// Verify test source directory was created at src/test/java (not src/main/java)
	testSourceDir := filepath.Join(dir, "data", "testing", "src", "test", "java", "com", "example")
	if _, err := os.Stat(testSourceDir); os.IsNotExist(err) {
		t.Errorf("test source directory %s was not created", testSourceDir)
	}
	mainSourceDir := filepath.Join(dir, "data", "testing", "src", "main", "java", "com", "example")
	if _, err := os.Stat(mainSourceDir); !os.IsNotExist(err) {
		t.Errorf("main source directory %s should not exist for test-related rules", mainSourceDir)
	}

	// Verify manifest paths use test source dir
	manifestData, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parsing manifest: %v", err)
	}
	if len(manifest.Groups) != 1 {
		t.Fatalf("manifest has %d groups, want 1", len(manifest.Groups))
	}
	sourceFile := manifest.Groups[0].Files[1]
	expectedPath := filepath.Join("data", "testing", "src", "test", "java", "com", "example", "ApplicationTest.java")
	if sourceFile.Path != expectedPath {
		t.Errorf("source file path = %q, want %q", sourceFile.Path, expectedPath)
	}
}

func TestRunWithBuiltinRules(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	os.MkdirAll(rulesDir, 0o755)

	ruleList := []rules.Rule{
		{
			RuleID:  "config-00010",
			Message: "test",
			When:    rules.NewBuiltinFilecontent("spring\\.jpa\\.hibernate", `application.*\.properties`),
		},
	}
	ruleData, _ := yaml.Marshal(ruleList)
	os.WriteFile(filepath.Join(rulesDir, "config.yaml"), ruleData, 0o644)

	langsDir := setupTestLanguagesDir(t)
	result, err := Run(rulesDir, dir, "java", langsDir)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	manifestData, _ := os.ReadFile(result.ManifestPath)
	var manifest Manifest
	json.Unmarshal(manifestData, &manifest)

	if len(manifest.Groups) != 1 {
		t.Fatalf("manifest has %d groups, want 1", len(manifest.Groups))
	}
	// Expect 3 files: pom.xml + Application.java + application.properties
	if len(manifest.Groups[0].Files) != 3 {
		t.Errorf("group has %d files, want 3 (pom.xml + Application.java + application.properties)", len(manifest.Groups[0].Files))
	}
	hasConfig := false
	for _, f := range manifest.Groups[0].Files {
		if f.Purpose == "config" {
			hasConfig = true
		}
	}
	if !hasConfig {
		t.Error("expected a config file in manifest")
	}
}

