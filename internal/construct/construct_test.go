package construct

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestBuildSingleCondition_JavaReferenced(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType: "java",
		SourceFQN:    "javax.ejb.Stateless",
		LocationType: "ANNOTATION",
	}
	c := buildSingleCondition(p)
	if c.JavaReferenced == nil {
		t.Fatal("expected java.referenced condition")
	}
	if c.JavaReferenced.Pattern != "javax.ejb.Stateless" {
		t.Errorf("pattern: got %q", c.JavaReferenced.Pattern)
	}
	if c.JavaReferenced.Location != "ANNOTATION" {
		t.Errorf("location: got %q", c.JavaReferenced.Location)
	}
}

func TestBuildSingleCondition_JavaDependency(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType:   "java",
		DependencyName: "org.springframework.boot.spring-boot-starter-undertow",
		UpperBound:     "4.0.0",
	}
	c := buildSingleCondition(p)
	if c.JavaDependency == nil {
		t.Fatal("expected java.dependency condition")
	}
	if c.JavaDependency.Name != "org.springframework.boot.spring-boot-starter-undertow" {
		t.Errorf("name: got %q", c.JavaDependency.Name)
	}
	if c.JavaDependency.Upperbound != "4.0.0" {
		t.Errorf("upperbound: got %q", c.JavaDependency.Upperbound)
	}
}

func TestBuildSingleCondition_JavaDependencyWithBounds(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType:   "java",
		DependencyName: "org.springframework.boot.spring-boot-properties-migrator",
		LowerBound:     "4.0.0-M1",
		UpperBound:     "4.0.0",
	}
	c := buildSingleCondition(p)
	if c.JavaDependency == nil {
		t.Fatal("expected java.dependency condition")
	}
	if c.JavaDependency.Lowerbound != "4.0.0-M1" {
		t.Errorf("lowerbound: got %q", c.JavaDependency.Lowerbound)
	}
	if c.JavaDependency.Upperbound != "4.0.0" {
		t.Errorf("upperbound: got %q", c.JavaDependency.Upperbound)
	}
}

func TestBuildSingleCondition_JavaDependencyDefaultProvider(t *testing.T) {
	p := rules.MigrationPattern{
		DependencyName: "org.flywaydb.flyway-core",
		UpperBound:     "4.0.0",
	}
	c := buildSingleCondition(p)
	if c.JavaDependency == nil {
		t.Fatal("expected java.dependency condition (default provider)")
	}
}

func TestBuildSingleCondition_GoDependency(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType:   "go",
		DependencyName: "golang.org/x/crypto",
		UpperBound:     "0.25.0",
	}
	c := buildSingleCondition(p)
	if c.GoDependency == nil {
		t.Fatal("expected go.dependency condition")
	}
	if c.GoDependency.Name != "golang.org/x/crypto" {
		t.Errorf("name: got %q", c.GoDependency.Name)
	}
	if c.GoDependency.Upperbound != "0.25.0" {
		t.Errorf("upperbound: got %q", c.GoDependency.Upperbound)
	}
}

func TestBuildSingleCondition_BuiltinXML(t *testing.T) {
	p := rules.MigrationPattern{
		XPath:      "/m:project/m:properties/m:spring-authorization-server.version",
		Namespaces: map[string]string{"m": "http://maven.apache.org/POM/4.0.0"},
		XPathFilepaths: []string{"pom.xml"},
	}
	c := buildSingleCondition(p)
	if c.BuiltinXML == nil {
		t.Fatal("expected builtin.xml condition")
	}
	if c.BuiltinXML.XPath != "/m:project/m:properties/m:spring-authorization-server.version" {
		t.Errorf("xpath: got %q", c.BuiltinXML.XPath)
	}
	if c.BuiltinXML.Namespaces["m"] != "http://maven.apache.org/POM/4.0.0" {
		t.Errorf("namespaces: got %v", c.BuiltinXML.Namespaces)
	}
	if len(c.BuiltinXML.Filepaths) != 1 || c.BuiltinXML.Filepaths[0] != "pom.xml" {
		t.Errorf("filepaths: got %v", c.BuiltinXML.Filepaths)
	}
}

func TestBuildSingleCondition_BuiltinXMLNoFilepaths(t *testing.T) {
	p := rules.MigrationPattern{
		XPath: "//bean[@class]",
	}
	c := buildSingleCondition(p)
	if c.BuiltinXML == nil {
		t.Fatal("expected builtin.xml condition")
	}
	if c.BuiltinXML.Filepaths != nil {
		t.Errorf("filepaths should be nil, got %v", c.BuiltinXML.Filepaths)
	}
}

func TestBuildSingleCondition_PythonReferenced(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType: "python",
		SourceFQN:    "flask.Flask",
	}
	c := buildSingleCondition(p)
	if c.PythonReferenced == nil {
		t.Fatal("expected python.referenced condition")
	}
	if c.PythonReferenced.Pattern != "flask.Flask" {
		t.Errorf("pattern: got %q", c.PythonReferenced.Pattern)
	}
}

func TestBuildSingleCondition_BuiltinFilecontent(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType: "builtin",
		SourceFQN:    `spring\.jackson\.read\.`,
		FilePattern:  "application*",
	}
	c := buildSingleCondition(p)
	if c.BuiltinFilecontent == nil {
		t.Fatal("expected builtin.filecontent condition")
	}
	if c.BuiltinFilecontent.Pattern != `spring\.jackson\.read\.` {
		t.Errorf("pattern: got %q", c.BuiltinFilecontent.Pattern)
	}
}

func TestBuildSingleCondition_DependencyTakesPrecedence(t *testing.T) {
	// When both dependency_name and source_fqn are set, dependency wins
	p := rules.MigrationPattern{
		ProviderType:   "java",
		SourceFQN:      "org.springframework.boot.SomeClass",
		DependencyName: "org.springframework.boot.some-dep",
		UpperBound:     "4.0.0",
	}
	c := buildSingleCondition(p)
	if c.JavaDependency == nil {
		t.Fatal("expected java.dependency (dependency_name takes precedence)")
	}
	if c.JavaReferenced != nil {
		t.Error("java.referenced should be nil when dependency_name is set")
	}
}

func TestBuildCondition_WithAlternativeFQNs(t *testing.T) {
	p := rules.MigrationPattern{
		ProviderType:    "java",
		SourceFQN:       "org.springframework.boot.BootstrapRegistry",
		LocationType:    "IMPORT",
		AlternativeFQNs: []string{"org.springframework.boot.BootstrapRegistryInitializer"},
	}
	c := buildCondition(p)
	if len(c.Or) != 2 {
		t.Fatalf("expected 2 or entries, got %d", len(c.Or))
	}
	if c.Or[0].JavaReferenced == nil {
		t.Error("first or entry should be java.referenced")
	}
	if c.Or[1].JavaReferenced.Pattern != "org.springframework.boot.BootstrapRegistryInitializer" {
		t.Errorf("second or entry pattern: got %q", c.Or[1].JavaReferenced.Pattern)
	}
}

func TestRun_MixedConditionTypes(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"test-source"},
		Targets:  []string{"test-target"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "old annotation",
				SourceFQN:     "com.example.OldAnnotation",
				LocationType:  "ANNOTATION",
				ProviderType:  "java",
				Rationale:     "Annotation removed",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
			{
				SourcePattern:  "old dependency",
				DependencyName: "com.example.old-dep",
				UpperBound:     "2.0.0",
				ProviderType:   "java",
				Rationale:      "Dependency removed",
				Complexity:     "medium",
				Category:       "mandatory",
				Concern:        "core",
			},
			{
				SourcePattern:  "xml config",
				XPath:          "/project/properties/old-version",
				XPathFilepaths: []string{"pom.xml"},
				Rationale:      "Property renamed",
				Complexity:     "trivial",
				Category:       "mandatory",
				Concern:        "build",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 3 {
		t.Errorf("rules written: got %d, want 3", result.RulesWritten)
	}

	// Verify rule files exist
	for _, name := range []string{"core.yaml", "build.yaml", "ruleset.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", name, err)
		}
	}
}

func TestPatternToRule_FullDescription(t *testing.T) {
	long := "BootstrapRegistry moved from org.springframework.boot to org.springframework.boot.bootstrap in Spring Boot 4.0. Update all imports and spring.factories references accordingly."
	p := rules.MigrationPattern{
		SourcePattern: "BootstrapRegistry relocated",
		SourceFQN:     "org.springframework.boot.BootstrapRegistry",
		LocationType:  "IMPORT",
		ProviderType:  "java",
		Rationale:     long,
		Complexity:    "low",
		Category:      "mandatory",
	}
	idGen := rules.NewIDGenerator()
	rule := patternToRule(p, idGen, "core-import", []string{"sb3"}, []string{"sb4"})
	if rule.Description != long {
		t.Errorf("description was truncated: got %q", rule.Description)
	}
}

func TestPatternToRule_LinksFromDocURL(t *testing.T) {
	p := rules.MigrationPattern{
		SourcePattern:    "test pattern",
		SourceFQN:        "com.example.Old",
		LocationType:     "IMPORT",
		ProviderType:     "java",
		Rationale:        "test",
		Complexity:       "low",
		Category:         "mandatory",
		DocumentationURL: "https://example.com/migration#section",
	}
	idGen := rules.NewIDGenerator()
	rule := patternToRule(p, idGen, "core-import", []string{"sb3"}, []string{"sb4"})
	if len(rule.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(rule.Links))
	}
	if rule.Links[0].URL != "https://example.com/migration#section" {
		t.Errorf("link URL: got %q", rule.Links[0].URL)
	}
	if rule.Links[0].Title != "Migration Documentation" {
		t.Errorf("link title: got %q", rule.Links[0].Title)
	}
}

func TestPatternToRule_NoLinksWithoutDocURL(t *testing.T) {
	p := rules.MigrationPattern{
		SourcePattern: "test",
		SourceFQN:     "com.example.Old",
		ProviderType:  "java",
		LocationType:  "IMPORT",
		Rationale:     "test",
		Complexity:    "low",
		Category:      "mandatory",
	}
	idGen := rules.NewIDGenerator()
	rule := patternToRule(p, idGen, "core-import", []string{"sb3"}, []string{"sb4"})
	if rule.Links != nil {
		t.Errorf("expected nil links when no documentation_url, got %v", rule.Links)
	}
}

func TestRun_PatternRuleMap(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"sb3"},
		Targets:  []string{"sb4"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{SourcePattern: "A", SourceFQN: "com.example.A", Rationale: "r1", Complexity: "low", Category: "mandatory", ProviderType: "java", LocationType: "IMPORT"},
			{SourcePattern: "B", SourceFQN: "com.example.B", Rationale: "r2", Complexity: "low", Category: "mandatory", ProviderType: "java", LocationType: "IMPORT"},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.PatternRuleMap) != 2 {
		t.Fatalf("PatternRuleMap length = %d, want 2", len(result.PatternRuleMap))
	}

	for idx, ruleID := range result.PatternRuleMap {
		if ruleID == "" {
			t.Errorf("pattern %d has empty rule ID", idx)
		}
	}
}

func TestRun_MultiSourceTargetLabels(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"oraclejdk7+", "oraclejdk"},
		Targets:  []string{"openjdk7+", "openjdk"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "JavaFX removed",
				SourceFQN:     "javafx.application.Application",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "JavaFX removed from Oracle JDK",
				Complexity:    "high",
				Category:      "mandatory",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify rule has all source/target labels
	for _, rr := range result.Grouped {
		for _, r := range rr {
			wantLabels := map[string]bool{
				"konveyor.io/source=oraclejdk7+":     true,
				"konveyor.io/source=oraclejdk":        true,
				"konveyor.io/target=openjdk7+":        true,
				"konveyor.io/target=openjdk":           true,
				"konveyor.io/generated-by=ai-rule-gen": true,
			}
			for _, l := range r.Labels {
				delete(wantLabels, l)
			}
			if len(wantLabels) > 0 {
				t.Errorf("rule %s missing labels: %v", r.RuleID, wantLabels)
			}
		}
	}

	// Verify ruleset has all labels
	rs := result.Ruleset
	if rs.Name != "openjdk7+/oraclejdk7+" {
		t.Errorf("ruleset name: got %q, want %q", rs.Name, "openjdk7+/oraclejdk7+")
	}
	wantRulesetLabels := []string{
		"konveyor.io/source=oraclejdk7+",
		"konveyor.io/source=oraclejdk",
		"konveyor.io/target=openjdk7+",
		"konveyor.io/target=openjdk",
	}
	if len(rs.Labels) != len(wantRulesetLabels) {
		t.Fatalf("ruleset labels = %v, want %v", rs.Labels, wantRulesetLabels)
	}
	for i, l := range rs.Labels {
		if l != wantRulesetLabels[i] {
			t.Errorf("ruleset label[%d] = %q, want %q", i, l, wantRulesetLabels[i])
		}
	}

	// Verify rule ID prefix uses {concern}-{change-type} format
	ruleID := result.PatternRuleMap[0]
	if !strings.HasPrefix(ruleID, "general-import-") {
		t.Errorf("rule ID %q should start with general-import-", ruleID)
	}
}

func TestRun_EmptySources(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  nil,
		Targets:  []string{"openjdk17"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "SecurityManager removed",
				SourceFQN:     "java.lang.SecurityManager",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "SecurityManager removed in JDK 17",
				Complexity:    "high",
				Category:      "mandatory",
				Concern:       "security",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1", result.RulesWritten)
	}

	// No source labels on rules
	for _, rr := range result.Grouped {
		for _, r := range rr {
			for _, l := range r.Labels {
				if strings.HasPrefix(l, "konveyor.io/source=") {
					t.Errorf("rule %s should have no source label, got %q", r.RuleID, l)
				}
			}
		}
	}

	// Ruleset name uses target only
	if result.Ruleset.Name != "openjdk17" {
		t.Errorf("ruleset name: got %q, want %q", result.Ruleset.Name, "openjdk17")
	}

	// Rule ID uses concern-summary format
	ruleID := result.PatternRuleMap[0]
	if !strings.HasPrefix(ruleID, "security-import-") {
		t.Errorf("rule ID %q should start with security-import-", ruleID)
	}
}

func TestRun_RejectsUnqualifiedMethodCall(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"hc4"},
		Targets:  []string{"hc5"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "bare method name",
				SourceFQN:     "setRetryHandler",
				LocationType:  "METHOD_CALL",
				ProviderType:  "java",
				Rationale:     "Retry handler changed",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
			{
				SourcePattern: "qualified import",
				SourceFQN:     "org.apache.http.HttpEntity",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "Package relocated",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1 (bare METHOD_CALL should be skipped)", result.RulesWritten)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings: got %d, want 1", len(result.Warnings))
	}
	if !strings.Contains(result.Warnings[0], "setRetryHandler") {
		t.Errorf("warning should mention the bare method name, got: %s", result.Warnings[0])
	}
}

func TestRun_AcceptsQualifiedMethodCall(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"hc4"},
		Targets:  []string{"hc5"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "qualified method call",
				SourceFQN:     "org.apache.http.impl.client.HttpClientBuilder.setRetryHandler",
				LocationType:  "METHOD_CALL",
				ProviderType:  "java",
				Rationale:     "Retry handler changed",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1", result.RulesWritten)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestRun_IgnoresNonMethodCallBareNames(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"sb3"},
		Targets:  []string{"sb4"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "bare import name",
				SourceFQN:     "SecurityManager",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "Removed in target",
				Complexity:    "high",
				Category:      "mandatory",
				Concern:       "security",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1 (IMPORT with bare name should not be rejected)", result.RulesWritten)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for IMPORT bare name, got %v", result.Warnings)
	}
}

func TestRun_SkipsEmptySourceFQNMethodCall(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"hc4"},
		Targets:  []string{"hc5"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern:  "dependency pattern",
				DependencyName: "org.apache.httpcomponents.httpclient",
				UpperBound:     "5.0.0",
				LocationType:   "METHOD_CALL",
				ProviderType:   "java",
				Rationale:      "Dependency changed",
				Complexity:     "low",
				Category:       "mandatory",
				Concern:        "core",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1 (empty SourceFQN should not trigger rejection)", result.RulesWritten)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for empty SourceFQN, got %v", result.Warnings)
	}
}

func TestRun_RejectsSourceEqualsTarget(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"sb3"},
		Targets:  []string{"sb4"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "wrong direction",
				SourceFQN:     "new.package.HttpMessageConverters",
				TargetPattern: "new.package.HttpMessageConverters",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "Class relocated",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
			{
				SourcePattern: "valid pattern",
				SourceFQN:     "old.package.SomeClass",
				TargetPattern: "new.package.SomeClass",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "Package relocated",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1 (source==target should be skipped)", result.RulesWritten)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings: got %d, want 1", len(result.Warnings))
	}
	if !strings.Contains(result.Warnings[0], "equals target_pattern") {
		t.Errorf("warning should mention equals target_pattern, got: %s", result.Warnings[0])
	}
}

func TestRun_AcceptsDifferentSourceAndTarget(t *testing.T) {
	extract := &rules.ExtractOutput{
		Sources:  []string{"sb3"},
		Targets:  []string{"sb4"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "class relocated",
				SourceFQN:     "old.package.HttpMessageConverters",
				TargetPattern: "new.package.HttpMessageConverters",
				LocationType:  "IMPORT",
				ProviderType:  "java",
				Rationale:     "Class relocated",
				Complexity:    "low",
				Category:      "mandatory",
				Concern:       "core",
			},
		},
	}

	dir := t.TempDir()
	result, err := Run(extract, dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.RulesWritten != 1 {
		t.Errorf("rules written: got %d, want 1", result.RulesWritten)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}
