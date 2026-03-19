package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	base := t.TempDir()
	w, err := New(base, "springboot", "quarkus")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	expected := filepath.Join(base, "springboot-to-quarkus")
	if w.Root != expected {
		t.Errorf("root: got %q, want %q", w.Root, expected)
	}

	// Check all subdirectories exist
	for _, dir := range []string{w.RulesDir(), w.TestsDir(), w.TestDataDir(), w.ConfidenceDir()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s does not exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestNewFromPath(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "custom-output")

	w, err := NewFromPath(root)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}

	if w.Root != root {
		t.Errorf("root: got %q, want %q", w.Root, root)
	}

	if _, err := os.Stat(w.RulesDir()); err != nil {
		t.Errorf("rules dir missing: %v", err)
	}
}

func TestWorkspace_Paths(t *testing.T) {
	w := &Workspace{Root: "/output/java-ee-to-quarkus"}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"RulesDir", w.RulesDir(), "/output/java-ee-to-quarkus/rules"},
		{"TestsDir", w.TestsDir(), "/output/java-ee-to-quarkus/tests"},
		{"TestDataDir", w.TestDataDir(), "/output/java-ee-to-quarkus/tests/data"},
		{"ConfidenceDir", w.ConfidenceDir(), "/output/java-ee-to-quarkus/confidence"},
		{"RulesetPath", w.RulesetPath(), "/output/java-ee-to-quarkus/rules/ruleset.yaml"},
		{"ScoresPath", w.ScoresPath(), "/output/java-ee-to-quarkus/confidence/scores.yaml"},
		{"RulesFilePath(security)", w.RulesFilePath("security"), "/output/java-ee-to-quarkus/rules/security.yaml"},
		{"RulesFilePath(empty)", w.RulesFilePath(""), "/output/java-ee-to-quarkus/rules/general.yaml"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}
