//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/scaffold"
	"gopkg.in/yaml.v3"
)

type languageConfigFile struct {
	Scaffold scaffold.LanguageConfig `json:"scaffold"`
}

func TestScaffoldRun_UsesRepositoryLanguageConfig(t *testing.T) {
	tmpDir := t.TempDir()
	rulesDir := filepath.Join(tmpDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("create rules dir: %v", err)
	}

	ruleList := []rules.Rule{
		{
			RuleID:  "java-rule-0001",
			Message: "detect javax usage",
			When:    rules.NewJavaReferenced("javax.ejb.Stateless", "ANNOTATION"),
		},
	}
	ruleData, err := yaml.Marshal(ruleList)
	if err != nil {
		t.Fatalf("marshal rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "web.yaml"), ruleData, 0o644); err != nil {
		t.Fatalf("write rules file: %v", err)
	}

	rulesetData, err := yaml.Marshal(rules.Ruleset{Name: "integration"})
	if err != nil {
		t.Fatalf("marshal ruleset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "ruleset.yaml"), rulesetData, 0o644); err != nil {
		t.Fatalf("write ruleset file: %v", err)
	}

	languagesDir := filepath.Join(repoRoot(t), "languages")
	outDir := filepath.Join(tmpDir, "out")
	result, err := scaffold.Run(rulesDir, outDir, "", languagesDir)
	if err != nil {
		t.Fatalf("scaffold run failed: %v", err)
	}

	if result.Language != "java" {
		t.Fatalf("detected language = %q, want %q", result.Language, "java")
	}

	manifestBytes, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest scaffold.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(manifest.Groups) != 1 {
		t.Fatalf("manifest groups = %d, want 1", len(manifest.Groups))
	}

	expectedCfg := loadLanguageConfigFromRepoFile(t, languagesDir, "java")
	group := manifest.Groups[0]
	expectedBuildPath := filepath.Join(group.DataDir, expectedCfg.BuildFile)
	expectedMainPath := filepath.Join(group.DataDir, expectedCfg.SourceDir, expectedCfg.MainFile)

	if !hasManifestPath(group.Files, expectedBuildPath) {
		t.Fatalf("manifest missing build file path %q", expectedBuildPath)
	}
	if !hasManifestPath(group.Files, expectedMainPath) {
		t.Fatalf("manifest missing source file path %q", expectedMainPath)
	}
}

func loadLanguageConfigFromRepoFile(t *testing.T, languagesDir, language string) scaffold.LanguageConfig {
	t.Helper()

	configPath := filepath.Join(languagesDir, language, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read language config %s: %v", configPath, err)
	}

	var cfg languageConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse language config %s: %v", configPath, err)
	}
	return cfg.Scaffold
}

func hasManifestPath(files []scaffold.ManifestFile, wantPath string) bool {
	for _, f := range files {
		if f.Path == wantPath {
			return true
		}
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}
