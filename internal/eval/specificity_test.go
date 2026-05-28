package eval

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/groundtruth"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestDetectSpecificityGaps(t *testing.T) {
	appDir := t.TempDir()

	javaFile := `package com.example;

import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.cookie.CookieSpecs;
import org.apache.http.impl.client.HttpClients;

public class MyClient {}
`
	srcDir := filepath.Join(appDir, "src", "main", "java", "com", "example")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, "MyClient.java"), []byte(javaFile), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
		{
			RuleID: "rule-00310",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.client.methods.HttpGet",
					Location: rules.LocationImport,
				},
			},
		},
		{
			RuleID: "rule-00150",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.impl.client.HttpClients",
					Location: rules.LocationImport,
				},
			},
		},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-00010": {Incidents: 4},
			"rule-00310": {Incidents: 1},
			"rule-00150": {Incidents: 1},
		},
	}

	gaps := DetectSpecificityGaps(ruleList, cov, appDir, nil)

	gapFQNs := make([]string, len(gaps))
	for i, g := range gaps {
		gapFQNs[i] = g.ImportFQN
	}
	sort.Strings(gapFQNs)

	// HttpGet and HttpClients have specific rules → no gaps
	// RequestConfig and CookieSpecs have no specific rules → gaps
	if len(gaps) != 2 {
		t.Fatalf("got %d gaps, want 2: %v", len(gaps), gapFQNs)
	}

	if gapFQNs[0] != "org.apache.http.client.config.RequestConfig" {
		t.Errorf("gap 0: got %q, want RequestConfig", gapFQNs[0])
	}
	if gapFQNs[1] != "org.apache.http.cookie.CookieSpecs" {
		t.Errorf("gap 1: got %q, want CookieSpecs", gapFQNs[1])
	}

	for _, g := range gaps {
		if g.BroadRuleID != "rule-00010" {
			t.Errorf("gap %s: broad rule should be rule-00010, got %q", g.ImportFQN, g.BroadRuleID)
		}
		if len(g.AppFiles) == 0 {
			t.Errorf("gap %s: should have app files", g.ImportFQN)
		}
	}
}

func TestDetectSpecificityGapsOrCombinator(t *testing.T) {
	appDir := t.TempDir()

	javaFile := `package com.example;

import org.apache.http.auth.AuthScope;
import org.apache.http.auth.UsernamePasswordCredentials;

public class AuthClient {}
`
	os.WriteFile(filepath.Join(appDir, "AuthClient.java"), []byte(javaFile), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
		{
			RuleID: "rule-00420",
			When: rules.Condition{
				Or: []rules.ConditionEntry{
					{Condition: rules.Condition{JavaReferenced: &rules.JavaReferenced{
						Pattern:  "org.apache.http.auth.AuthScope",
						Location: rules.LocationImport,
					}}},
					{Condition: rules.Condition{JavaReferenced: &rules.JavaReferenced{
						Pattern:  "org.apache.http.auth.UsernamePasswordCredentials",
						Location: rules.LocationImport,
					}}},
				},
			},
		},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-00010": {Incidents: 2},
			"rule-00420": {Incidents: 2},
		},
	}

	gaps := DetectSpecificityGaps(ruleList, cov, appDir, nil)
	if len(gaps) != 0 {
		fqns := make([]string, len(gaps))
		for i, g := range gaps {
			fqns[i] = g.ImportFQN
		}
		t.Errorf("expected no gaps when or: combinator covers both imports, got %d: %v", len(gaps), fqns)
	}
}

func TestDetectSpecificityGapsNoBroadRules(t *testing.T) {
	appDir := t.TempDir()
	javaFile := `import org.apache.http.HttpGet;`
	os.WriteFile(filepath.Join(appDir, "Test.java"), []byte(javaFile), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "rule-001",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.HttpGet",
					Location: rules.LocationImport,
				},
			},
		},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-001": {Incidents: 1},
		},
	}

	gaps := DetectSpecificityGaps(ruleList, cov, appDir, nil)
	if len(gaps) != 0 {
		t.Errorf("expected no gaps when no broad rules exist, got %d", len(gaps))
	}
}

func TestDetectSpecificityGapsNoAppDir(t *testing.T) {
	gaps := DetectSpecificityGaps(nil, nil, "", nil)
	if len(gaps) != 0 {
		t.Errorf("expected no gaps with empty appDir, got %d", len(gaps))
	}
}

func TestDetectSpecificityGapsMultipleFiles(t *testing.T) {
	appDir := t.TempDir()

	file1 := `import org.apache.http.client.config.RequestConfig;`
	file2 := `import org.apache.http.client.config.RequestConfig;`

	os.WriteFile(filepath.Join(appDir, "A.java"), []byte(file1), 0o644)
	os.WriteFile(filepath.Join(appDir, "B.java"), []byte(file2), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-00010": {Incidents: 2},
		},
	}

	gaps := DetectSpecificityGaps(ruleList, cov, appDir, nil)
	if len(gaps) != 1 {
		t.Fatalf("got %d gaps, want 1", len(gaps))
	}
	if len(gaps[0].AppFiles) != 2 {
		t.Errorf("got %d files, want 2", len(gaps[0].AppFiles))
	}
}

func TestDetectSpecificityGapsFromGuide(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
		{
			RuleID: "rule-00370",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.client.methods.HttpGet",
					Location: rules.LocationImport,
				},
			},
		},
	}

	gt := &groundtruth.GroundTruth{
		Entries: []groundtruth.Entry{
			{OldAPI: "org.apache.http.client.methods.HttpGet", ActionType: "package_change"},
			{OldAPI: "org.apache.http.client.config.RequestConfig", ActionType: "class_rename"},
			{OldAPI: "org.apache.http.auth.AuthScope", ActionType: "method_removal"},
			{OldAPI: "com.unrelated.SomeClass", ActionType: "class_rename"},
		},
	}

	gaps := DetectSpecificityGapsFromGuide(ruleList, gt)

	gapFQNs := make([]string, len(gaps))
	for i, g := range gaps {
		gapFQNs[i] = g.ImportFQN
	}
	sort.Strings(gapFQNs)

	if len(gaps) != 2 {
		t.Fatalf("got %d gaps, want 2: %v", len(gaps), gapFQNs)
	}
	if gapFQNs[0] != "org.apache.http.auth.AuthScope" {
		t.Errorf("gap 0: got %q, want AuthScope", gapFQNs[0])
	}
	if gapFQNs[1] != "org.apache.http.client.config.RequestConfig" {
		t.Errorf("gap 1: got %q, want RequestConfig", gapFQNs[1])
	}
	for _, g := range gaps {
		if g.Source != "ground_truth" {
			t.Errorf("gap %s: source should be ground_truth, got %q", g.ImportFQN, g.Source)
		}
		if len(g.AppFiles) != 0 {
			t.Errorf("gap %s: should have no app files", g.ImportFQN)
		}
	}
}

func TestDetectSpecificityGapsFromGuideSkipsPackageChange(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
	}

	gt := &groundtruth.GroundTruth{
		Entries: []groundtruth.Entry{
			{OldAPI: "org.apache.http.Header", ActionType: "package_change"},
			{OldAPI: "org.apache.http.HttpHost", ActionType: "package_change"},
			{OldAPI: "org.apache.http.entity.EntityTemplate", ActionType: "class_rename"},
		},
	}

	gaps := DetectSpecificityGapsFromGuide(ruleList, gt)
	if len(gaps) != 1 {
		t.Fatalf("got %d gaps, want 1 (only class_rename)", len(gaps))
	}
	if gaps[0].ImportFQN != "org.apache.http.entity.EntityTemplate" {
		t.Errorf("got %q, want EntityTemplate", gaps[0].ImportFQN)
	}
	if gaps[0].ActionType != "class_rename" {
		t.Errorf("action_type: got %q, want class_rename", gaps[0].ActionType)
	}
}

func TestDetectSpecificityGapsWithGTFiltering(t *testing.T) {
	appDir := t.TempDir()

	javaFile := `package com.example;

import org.apache.http.Header;
import org.apache.http.entity.EntityTemplate;
import org.apache.http.client.methods.HttpGet;

public class MyClient {}
`
	os.WriteFile(filepath.Join(appDir, "MyClient.java"), []byte(javaFile), 0o644)

	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
	}

	cov := &AppCoverage{
		Violations: map[string]Violation{
			"rule-00010": {Incidents: 3},
		},
	}

	gt := &groundtruth.GroundTruth{
		Entries: []groundtruth.Entry{
			{OldAPI: "org.apache.http.Header", ActionType: "package_change"},
			{OldAPI: "org.apache.http.entity.EntityTemplate", ActionType: "class_rename"},
			{OldAPI: "org.apache.http.client.methods.HttpGet", ActionType: "package_change"},
		},
	}

	gaps := DetectSpecificityGaps(ruleList, cov, appDir, gt)

	if len(gaps) != 1 {
		fqns := make([]string, len(gaps))
		for i, g := range gaps {
			fqns[i] = g.ImportFQN
		}
		t.Fatalf("got %d gaps, want 1 (only class_rename EntityTemplate): %v", len(gaps), fqns)
	}
	if gaps[0].ImportFQN != "org.apache.http.entity.EntityTemplate" {
		t.Errorf("got %q, want EntityTemplate", gaps[0].ImportFQN)
	}
	if gaps[0].ActionType != "class_rename" {
		t.Errorf("action_type: got %q, want class_rename", gaps[0].ActionType)
	}
}

func TestDetectSpecificityGapsFromGuideNoBroadRules(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-001",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.HttpGet",
					Location: rules.LocationImport,
				},
			},
		},
	}

	gt := &groundtruth.GroundTruth{
		Entries: []groundtruth.Entry{
			{OldAPI: "org.apache.http.client.config.RequestConfig"},
		},
	}

	gaps := DetectSpecificityGapsFromGuide(ruleList, gt)
	if len(gaps) != 0 {
		t.Errorf("expected no gaps when no broad rules exist, got %d", len(gaps))
	}
}

func TestDetectSpecificityGapsFromGuideAllCovered(t *testing.T) {
	ruleList := []rules.Rule{
		{
			RuleID: "rule-00010",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http",
					Location: rules.LocationPackage,
				},
			},
		},
		{
			RuleID: "rule-00370",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.client.methods.HttpGet",
					Location: rules.LocationImport,
				},
			},
		},
		{
			RuleID: "rule-00430",
			When: rules.Condition{
				JavaReferenced: &rules.JavaReferenced{
					Pattern:  "org.apache.http.auth.AuthScope",
					Location: rules.LocationImport,
				},
			},
		},
	}

	gt := &groundtruth.GroundTruth{
		Entries: []groundtruth.Entry{
			{OldAPI: "org.apache.http.client.methods.HttpGet", ActionType: "class_rename"},
			{OldAPI: "org.apache.http.auth.AuthScope", ActionType: "class_rename"},
		},
	}

	gaps := DetectSpecificityGapsFromGuide(ruleList, gt)
	if len(gaps) != 0 {
		t.Errorf("expected no gaps when all GT entries are covered, got %d", len(gaps))
	}
}
