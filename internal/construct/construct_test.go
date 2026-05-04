package construct

import (
	"os"
	"path/filepath"
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
		Source:   "test-source",
		Target:   "test-target",
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
