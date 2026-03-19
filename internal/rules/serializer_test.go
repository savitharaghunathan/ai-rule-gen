package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadRulesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test message",
		When:    NewJavaReferenced("javax.ejb.Stateless", LocationAnnotation),
	}}

	if err := WriteRulesFile(path, rules); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadRulesFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	if got[0].RuleID != "test-00010" {
		t.Errorf("ruleID: got %q", got[0].RuleID)
	}
}

func TestWriteAndReadRuleset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ruleset.yaml")

	rs := &Ruleset{
		Name:        "test/ruleset",
		Description: "Test ruleset",
	}

	if err := WriteRuleset(path, rs); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadRuleset(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if got.Name != "test/ruleset" {
		t.Errorf("name: got %q", got.Name)
	}
}

func TestReadRulesDir(t *testing.T) {
	dir := t.TempDir()

	// Write ruleset.yaml (should be skipped)
	if err := WriteRuleset(filepath.Join(dir, "ruleset.yaml"), &Ruleset{Name: "test"}); err != nil {
		t.Fatal(err)
	}

	// Write two rule files
	if err := WriteRulesFile(filepath.Join(dir, "security.yaml"), []Rule{
		{RuleID: "sec-00010", Message: "security rule", When: NewJavaReferenced("foo", "")},
	}); err != nil {
		t.Fatal(err)
	}
	if err := WriteRulesFile(filepath.Join(dir, "web.yaml"), []Rule{
		{RuleID: "web-00010", Message: "web rule", When: NewJavaReferenced("bar", "")},
		{RuleID: "web-00020", Message: "web rule 2", When: NewJavaReferenced("baz", "")},
	}); err != nil {
		t.Fatal(err)
	}

	// Write a non-yaml file (should be skipped)
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadRulesDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("expected 3 rules, got %d", len(got))
	}
}

func TestWriteRulesGrouped(t *testing.T) {
	dir := t.TempDir()
	grouped := map[string][]Rule{
		"security": {{RuleID: "sec-00010", Message: "sec", When: NewJavaReferenced("foo", "")}},
		"web":      {{RuleID: "web-00010", Message: "web", When: NewJavaReferenced("bar", "")}},
		"":         {{RuleID: "gen-00010", Message: "gen", When: NewJavaReferenced("baz", "")}},
	}

	if err := WriteRulesGrouped(dir, grouped); err != nil {
		t.Fatalf("write grouped: %v", err)
	}

	// Check files exist
	for _, name := range []string{"security.yaml", "web.yaml", "general.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestReadRulesFile_NotFound(t *testing.T) {
	_, err := ReadRulesFile("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
