package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

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

	result, err := Run(rulesDir, dir, "")
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
	testFilePath := filepath.Join(dir, "tests", "ejb.test.yaml")
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
	dataDir := filepath.Join(dir, "tests", "data", "ejb", "src", "main", "java", "com", "example")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Errorf("data directory %s was not created", dataDir)
	}
}

func TestGetLanguageConfig(t *testing.T) {
	cfg, ok := GetLanguageConfig("java")
	if !ok {
		t.Fatal("java config not found")
	}
	if cfg.BuildFile != "pom.xml" {
		t.Errorf("BuildFile = %q, want %q", cfg.BuildFile, "pom.xml")
	}

	_, ok = GetLanguageConfig("rust")
	if ok {
		t.Error("expected rust config to not exist")
	}
}
