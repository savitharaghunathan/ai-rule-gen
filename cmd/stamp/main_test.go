package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestStampCLI_VerifiedNotFound(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ruleList := []rules.Rule{
		{RuleID: "rule-001", Message: "verified", When: rules.NewJavaReferenced("com.example.A", rules.LocationImport)},
		{RuleID: "rule-002", Message: "not found", When: rules.NewJavaReferenced("com.example.B", rules.LocationImport)},
		{RuleID: "rule-003", Message: "untouched", When: rules.NewJavaReferenced("com.example.C", rules.LocationImport)},
	}
	if err := rules.WriteRulesFile(filepath.Join(rulesDir, "test.yaml"), ruleList); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./", "--rules", rulesDir, "--verified", "rule-001", "--not-found", "rule-002")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stamp failed: %v\n%s", err, out)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("parse output: %v\n%s", err, out)
	}
	if result["verified"].(float64) != 1 {
		t.Errorf("verified = %v, want 1", result["verified"])
	}
	if result["not_found"].(float64) != 1 {
		t.Errorf("not_found = %v, want 1", result["not_found"])
	}

	got, err := rules.ReadRulesFile(filepath.Join(rulesDir, "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range got {
		switch r.RuleID {
		case "rule-001":
			assertLabel(t, r, "konveyor.io/source-verified=true")
		case "rule-002":
			assertLabel(t, r, "konveyor.io/source-verified=false")
		case "rule-003":
			for _, l := range r.Labels {
				if strings.HasPrefix(l, "konveyor.io/source-verified=") {
					t.Errorf("rule-003 should not have source-verified label, got %q", l)
				}
			}
		}
	}
}

func assertLabel(t *testing.T, r rules.Rule, want string) {
	t.Helper()
	for _, l := range r.Labels {
		if l == want {
			return
		}
	}
	t.Errorf("%s: expected label %q, got %v", r.RuleID, want, r.Labels)
}
