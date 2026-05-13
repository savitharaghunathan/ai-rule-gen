package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergePatterns_Dedup(t *testing.T) {
	parts := []*ExtractOutput{
		{
			Sources: []string{"sb3"}, Targets: []string{"sb4"}, Language: "java",
			Patterns: []MigrationPattern{
				{SourcePattern: "A", SourceFQN: "com.example.A", Rationale: "r1", Complexity: "low", Category: "mandatory"},
				{SourcePattern: "B", DependencyName: "org.foo.bar", Rationale: "r2", Complexity: "low", Category: "mandatory"},
			},
		},
		{
			Sources: []string{"sb3"}, Targets: []string{"sb4"}, Language: "java",
			Patterns: []MigrationPattern{
				{SourcePattern: "A dup", SourceFQN: "com.example.A", Rationale: "r1-dup", Complexity: "low", Category: "mandatory"},
				{SourcePattern: "C", SourceFQN: "com.example.C", Rationale: "r3", Complexity: "low", Category: "mandatory"},
			},
		},
	}

	result := MergePatterns(parts)
	if len(result.Output.Patterns) != 3 {
		t.Fatalf("expected 3 patterns after dedup, got %d", len(result.Output.Patterns))
	}
	if result.Output.Patterns[0].SourcePattern != "A" {
		t.Errorf("first occurrence should win, got %q", result.Output.Patterns[0].SourcePattern)
	}
	if result.Duplicates != 1 {
		t.Errorf("expected 1 duplicate, got %d", result.Duplicates)
	}
}

func TestMergePatterns_Empty(t *testing.T) {
	result := MergePatterns(nil)
	if result == nil || result.Output == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Output.Patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(result.Output.Patterns))
	}
}

func TestMergePatterns_PreservesMetadata(t *testing.T) {
	parts := []*ExtractOutput{
		{Sources: []string{"sb3"}, Targets: []string{"sb4"}, Language: "java"},
		{Sources: []string{"sb3"}, Targets: []string{"sb4"}, Language: "java"},
	}
	result := MergePatterns(parts)
	if len(result.Output.Sources) != 1 || result.Output.Sources[0] != "sb3" || len(result.Output.Targets) != 1 || result.Output.Targets[0] != "sb4" || result.Output.Language != "java" {
		t.Errorf("metadata not preserved: %+v", result.Output)
	}
}

func TestConsolidatePackages_Basic(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "org.apache.http", LocationType: "PACKAGE", Message: "Migrate from org.apache.http to org.apache.hc", Complexity: "low"},
		{SourceFQN: "org.apache.http.conn.ssl.SSLConnectionSocketFactory", LocationType: "IMPORT", TargetPattern: "Use ClientTlsStrategyBuilder", Complexity: "medium"},
		{SourceFQN: "org.apache.http.entity.EntityTemplate", LocationType: "IMPORT", TargetPattern: "Use HttpEntities.create()", Complexity: "low"},
		{SourceFQN: "org.apache.http.HttpResponse.getStatusLine", LocationType: "METHOD_CALL", Rationale: "Use getCode()", Complexity: "low"},
	}

	result, absorbed := consolidatePackages(patterns)
	if absorbed != 0 {
		t.Fatalf("expected 0 absorbed (class replacements not absorbed), got %d", absorbed)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 remaining patterns, got %d", len(result))
	}
	// IMPORT patterns with class replacements should be kept
	if result[1].SourceFQN != "org.apache.http.conn.ssl.SSLConnectionSocketFactory" {
		t.Errorf("SSLConnectionSocketFactory should be kept, got %q", result[1].SourceFQN)
	}
	if result[2].SourceFQN != "org.apache.http.entity.EntityTemplate" {
		t.Errorf("EntityTemplate should be kept, got %q", result[2].SourceFQN)
	}
	// METHOD_CALL should be kept
	if result[3].SourceFQN != "org.apache.http.HttpResponse.getStatusLine" {
		t.Errorf("METHOD_CALL pattern should be kept, got %q", result[3].SourceFQN)
	}
}

func TestConsolidatePackages_NoPackagePattern(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "com.example.Foo", LocationType: "IMPORT", Complexity: "low"},
		{SourceFQN: "com.example.Bar", LocationType: "IMPORT", Complexity: "low"},
	}
	result, absorbed := consolidatePackages(patterns)
	if absorbed != 0 {
		t.Errorf("expected 0 absorbed without PACKAGE pattern, got %d", absorbed)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 patterns unchanged, got %d", len(result))
	}
}

func TestConsolidatePackages_MethodCallNotAbsorbed(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "com.example", LocationType: "PACKAGE", Message: "pkg rename", Complexity: "low"},
		{SourceFQN: "com.example.Foo.doSomething", LocationType: "METHOD_CALL", Complexity: "low"},
		{SourceFQN: "com.example.Bar", LocationType: "TYPE", TargetPattern: "com.newpkg.Bar", Complexity: "low"},
	}
	result, absorbed := consolidatePackages(patterns)
	if absorbed != 1 {
		t.Fatalf("expected 1 absorbed (TYPE), got %d", absorbed)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 remaining (PACKAGE + METHOD_CALL), got %d", len(result))
	}
	if result[1].LocationType != "METHOD_CALL" {
		t.Errorf("METHOD_CALL should be kept, got %q", result[1].LocationType)
	}
}

func TestConsolidatePackages_MultiplePackages(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "com.alpha", LocationType: "PACKAGE", Message: "alpha pkg", Complexity: "low"},
		{SourceFQN: "com.beta", LocationType: "PACKAGE", Message: "beta pkg", Complexity: "low"},
		{SourceFQN: "com.alpha.Foo", LocationType: "IMPORT", TargetPattern: "new Foo", Complexity: "low"},
		{SourceFQN: "com.beta.Bar", LocationType: "IMPORT", TargetPattern: "new Bar", Complexity: "low"},
		{SourceFQN: "com.gamma.Baz", LocationType: "IMPORT", Complexity: "low"},
	}
	result, absorbed := consolidatePackages(patterns)
	if absorbed != 2 {
		t.Fatalf("expected 2 absorbed, got %d", absorbed)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 remaining (2 PACKAGE + 1 unmatched IMPORT), got %d", len(result))
	}
	if !strings.Contains(result[0].Message, "com.alpha.Foo") {
		t.Error("alpha package should absorb com.alpha.Foo")
	}
	if !strings.Contains(result[1].Message, "com.beta.Bar") {
		t.Error("beta package should absorb com.beta.Bar")
	}
}

func TestConsolidatePackages_ComplexityInheritance(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "com.example", LocationType: "PACKAGE", Message: "pkg", Complexity: "low"},
		{SourceFQN: "com.example.HardClass", LocationType: "IMPORT", TargetPattern: "new HardClass impl", Complexity: "high"},
		{SourceFQN: "com.example.EasyClass", LocationType: "IMPORT", TargetPattern: "new EasyClass impl", Complexity: "trivial"},
	}
	result, absorbed := consolidatePackages(patterns)
	if absorbed != 2 {
		t.Fatalf("expected 2 absorbed, got %d", absorbed)
	}
	if result[0].Complexity != "high" {
		t.Errorf("PACKAGE should inherit highest complexity 'high', got %q", result[0].Complexity)
	}
}

func TestConsolidatePackages_ClassReplacementNotAbsorbed(t *testing.T) {
	patterns := []MigrationPattern{
		{SourceFQN: "org.apache.http", LocationType: "PACKAGE", Message: "pkg migration", Complexity: "low"},
		// Class replacement: different name → not absorbed
		{SourceFQN: "org.apache.http.conn.ssl.SSLConnectionSocketFactory", LocationType: "IMPORT", TargetPattern: "ClientTlsStrategyBuilder", Complexity: "medium"},
		// Package move: same class name → absorbed
		{SourceFQN: "org.apache.http.client.HttpClient", LocationType: "IMPORT", TargetPattern: "org.apache.hc.client5.http.classic.HttpClient", Complexity: "low"},
		// Empty target: absorbed by default
		{SourceFQN: "org.apache.http.util.Args", LocationType: "IMPORT", Complexity: "low"},
		// TYPE with same class name in target: absorbed
		{SourceFQN: "org.apache.http.entity.ContentType", LocationType: "TYPE", TargetPattern: "org.apache.hc.core5.http.ContentType", Complexity: "low"},
	}

	result, absorbed := consolidatePackages(patterns)
	if absorbed != 3 {
		t.Fatalf("expected 3 absorbed (HttpClient, Args, ContentType), got %d", absorbed)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 remaining (PACKAGE + SSLConnectionSocketFactory), got %d", len(result))
	}
	if result[0].LocationType != "PACKAGE" {
		t.Errorf("first pattern should be PACKAGE, got %q", result[0].LocationType)
	}
	if result[1].SourceFQN != "org.apache.http.conn.ssl.SSLConnectionSocketFactory" {
		t.Errorf("SSLConnectionSocketFactory should be kept as standalone, got %q", result[1].SourceFQN)
	}
	if !strings.Contains(result[0].Message, "HttpClient") {
		t.Error("PACKAGE message should contain absorbed class HttpClient")
	}
	if !strings.Contains(result[0].Message, "Args") {
		t.Error("PACKAGE message should contain absorbed class Args")
	}
	if !strings.Contains(result[0].Message, "ContentType") {
		t.Error("PACKAGE message should contain absorbed class ContentType")
	}
}

func TestIsClassReplacement(t *testing.T) {
	tests := []struct {
		name      string
		sourceFQN string
		target    string
		want      bool
	}{
		{"empty target", "com.example.Foo", "", false},
		{"same class in target FQN", "com.example.Bar", "com.newpkg.Bar", false},
		{"same class preceded by space", "com.example.Foo", "new Foo", false},
		{"different class", "org.apache.http.conn.ssl.SSLConnectionSocketFactory", "ClientTlsStrategyBuilder", true},
		{"class name is substring not word", "com.example.Client", "HttpClient5", true},
		{"method call target", "org.apache.http.entity.EntityTemplate", "HttpEntities.create()", true},
		{"class at start of target", "com.example.Foo", "Foo replacement", false},
		{"class at end of target", "com.example.Foo", "use Foo", false},
		{"class is entire target", "com.example.Foo", "Foo", false},
		{"partial match not word boundary", "com.example.Bar", "Foobar", true},
		{"no dot in sourceFQN", "Foo", "use Foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClassReplacement(tt.sourceFQN, tt.target)
			if got != tt.want {
				t.Errorf("isClassReplacement(%q, %q) = %v, want %v", tt.sourceFQN, tt.target, got, tt.want)
			}
		})
	}
}

func TestWriteAndReadPatternsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patterns.json")

	original := &ExtractOutput{
		Sources:  []string{"sb3"},
		Targets:  []string{"sb4"},
		Language: "java",
		Patterns: []MigrationPattern{
			{SourcePattern: "A", SourceFQN: "com.A", Rationale: "r", Complexity: "low", Category: "mandatory"},
		},
	}

	if err := WritePatternsFile(path, original); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty file")
	}

	read, err := ReadPatternsFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(read.Patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(read.Patterns))
	}
	if read.Patterns[0].SourceFQN != "com.A" {
		t.Errorf("source_fqn = %q, want com.A", read.Patterns[0].SourceFQN)
	}
}

func TestWriteAndReadPatternsFile_SourceArtifact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patterns.json")

	original := &ExtractOutput{
		Sources:  []string{"httpcomponents-client-4"},
		Targets:  []string{"httpcomponents-client-5"},
		Language: "java",
		Patterns: []MigrationPattern{
			{
				SourcePattern: "HttpClient",
				SourceFQN:     "org.apache.http.client.HttpClient",
				Rationale:     "Class moved in v5",
				Complexity:    "low",
				Category:      "mandatory",
				ProviderType:  "java",
				SourceArtifact: &ArtifactCoordinates{
					GroupID:    "org.apache.httpcomponents",
					ArtifactID: "httpclient",
					Version:    "4.5.14",
				},
			},
		},
	}

	if err := WritePatternsFile(path, original); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	read, err := ReadPatternsFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if read.Patterns[0].SourceArtifact == nil {
		t.Fatal("source_artifact is nil after round-trip")
	}
	sa := read.Patterns[0].SourceArtifact
	if sa.GroupID != "org.apache.httpcomponents" {
		t.Errorf("groupId = %q, want org.apache.httpcomponents", sa.GroupID)
	}
	if sa.ArtifactID != "httpclient" {
		t.Errorf("artifactId = %q, want httpclient", sa.ArtifactID)
	}
	if sa.Version != "4.5.14" {
		t.Errorf("version = %q, want 4.5.14", sa.Version)
	}
}

func TestWriteAndReadPatternsFile_NilSourceArtifact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patterns.json")

	original := &ExtractOutput{
		Sources:  []string{"sb3"},
		Targets:  []string{"sb4"},
		Language: "java",
		Patterns: []MigrationPattern{
			{SourcePattern: "A", SourceFQN: "com.A", Rationale: "r", Complexity: "low", Category: "mandatory"},
		},
	}

	if err := WritePatternsFile(path, original); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	read, err := ReadPatternsFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if read.Patterns[0].SourceArtifact != nil {
		t.Errorf("expected nil source_artifact, got %+v", read.Patterns[0].SourceArtifact)
	}
}

func TestInitialLabels_SingleSourceTarget(t *testing.T) {
	labels := InitialLabels([]string{"sb3"}, []string{"sb4"})
	if len(labels) != 3 {
		t.Fatalf("expected 3 labels, got %d: %v", len(labels), labels)
	}
	if labels[0] != "konveyor.io/source=sb3" {
		t.Errorf("labels[0] = %q, want konveyor.io/source=sb3", labels[0])
	}
	if labels[1] != "konveyor.io/target=sb4" {
		t.Errorf("labels[1] = %q, want konveyor.io/target=sb4", labels[1])
	}
}

func TestInitialLabels_EmptyArrays(t *testing.T) {
	labels := InitialLabels(nil, nil)
	if len(labels) != 1 || labels[0] != "konveyor.io/generated-by=ai-rule-gen" {
		t.Errorf("InitialLabels(nil, nil) = %v, want only generated-by label", labels)
	}
}

func TestInitialLabels_MultipleSourcesTargets(t *testing.T) {
	labels := InitialLabels([]string{"oraclejdk7+", "oraclejdk"}, []string{"openjdk7+", "openjdk"})
	want := []string{
		"konveyor.io/source=oraclejdk7+",
		"konveyor.io/source=oraclejdk",
		"konveyor.io/target=openjdk7+",
		"konveyor.io/target=openjdk",
		"konveyor.io/generated-by=ai-rule-gen",
	}
	if len(labels) != len(want) {
		t.Fatalf("InitialLabels() length = %d, want %d", len(labels), len(want))
	}
	for i, l := range labels {
		if l != want[i] {
			t.Errorf("labels[%d] = %q, want %q", i, l, want[i])
		}
	}
}

func TestMergePatterns_MergesSourcesTargets(t *testing.T) {
	parts := []*ExtractOutput{
		{
			Sources: []string{"sb3", "spring-boot"}, Targets: []string{"sb4"}, Language: "java",
			Patterns: []MigrationPattern{
				{SourcePattern: "A", SourceFQN: "com.A", Rationale: "r", Complexity: "low", Category: "mandatory"},
			},
		},
		{
			Sources: []string{"spring-boot", "springboot"}, Targets: []string{"sb4", "spring-boot"}, Language: "java",
			Patterns: []MigrationPattern{
				{SourcePattern: "B", SourceFQN: "com.B", Rationale: "r", Complexity: "low", Category: "mandatory"},
			},
		},
	}
	result := MergePatterns(parts)
	wantSources := []string{"sb3", "spring-boot", "springboot"}
	wantTargets := []string{"sb4", "spring-boot"}

	if len(result.Output.Sources) != len(wantSources) {
		t.Fatalf("Sources = %v, want %v", result.Output.Sources, wantSources)
	}
	for i, s := range result.Output.Sources {
		if s != wantSources[i] {
			t.Errorf("Sources[%d] = %q, want %q", i, s, wantSources[i])
		}
	}
	if len(result.Output.Targets) != len(wantTargets) {
		t.Fatalf("Targets = %v, want %v", result.Output.Targets, wantTargets)
	}
	for i, s := range result.Output.Targets {
		if s != wantTargets[i] {
			t.Errorf("Targets[%d] = %q, want %q", i, s, wantTargets[i])
		}
	}
}

func TestUnionStrings(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{"both empty", nil, nil, []string{}},
		{"a only", []string{"x", "y"}, nil, []string{"x", "y"}},
		{"b only", nil, []string{"x"}, []string{"x"}},
		{"overlap", []string{"a", "b"}, []string{"b", "c"}, []string{"a", "b", "c"}},
		{"duplicates in a", []string{"a", "a"}, []string{"b"}, []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unionStrings(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Fatalf("unionStrings(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("unionStrings(%v, %v)[%d] = %q, want %q", tt.a, tt.b, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWriteAndReadPatternsFile_MultiSourceTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patterns.json")

	original := &ExtractOutput{
		Sources:  []string{"spring-boot3", "spring-boot"},
		Targets:  []string{"spring-boot4", "spring-boot"},
		Language: "java",
		Patterns: []MigrationPattern{
			{SourcePattern: "A", SourceFQN: "com.A", Rationale: "r", Complexity: "low", Category: "mandatory"},
		},
	}

	if err := WritePatternsFile(path, original); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	read, err := ReadPatternsFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(read.Sources) != 2 || read.Sources[0] != "spring-boot3" {
		t.Errorf("Sources = %v, want [spring-boot3 spring-boot]", read.Sources)
	}
	if len(read.Targets) != 2 || read.Targets[0] != "spring-boot4" {
		t.Errorf("Targets = %v, want [spring-boot4 spring-boot]", read.Targets)
	}
}
