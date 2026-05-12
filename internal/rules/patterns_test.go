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
			Source: "sb3", Target: "sb4", Language: "java",
			Patterns: []MigrationPattern{
				{SourcePattern: "A", SourceFQN: "com.example.A", Rationale: "r1", Complexity: "low", Category: "mandatory"},
				{SourcePattern: "B", DependencyName: "org.foo.bar", Rationale: "r2", Complexity: "low", Category: "mandatory"},
			},
		},
		{
			Source: "sb3", Target: "sb4", Language: "java",
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
		{Source: "sb3", Target: "sb4", Language: "java"},
		{Source: "sb3", Target: "sb4", Language: "java"},
	}
	result := MergePatterns(parts)
	if result.Output.Source != "sb3" || result.Output.Target != "sb4" || result.Output.Language != "java" {
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
	if absorbed != 2 {
		t.Fatalf("expected 2 absorbed, got %d", absorbed)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 remaining patterns, got %d", len(result))
	}
	// PACKAGE pattern should have enhanced message
	if !strings.Contains(result[0].Message, "SSLConnectionSocketFactory") {
		t.Error("PACKAGE message should contain absorbed class SSLConnectionSocketFactory")
	}
	if !strings.Contains(result[0].Message, "EntityTemplate") {
		t.Error("PACKAGE message should contain absorbed class EntityTemplate")
	}
	// METHOD_CALL should be kept
	if result[1].SourceFQN != "org.apache.http.HttpResponse.getStatusLine" {
		t.Errorf("METHOD_CALL pattern should be kept, got %q", result[1].SourceFQN)
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
		{SourceFQN: "com.example.HardClass", LocationType: "IMPORT", TargetPattern: "new class", Complexity: "high"},
		{SourceFQN: "com.example.EasyClass", LocationType: "IMPORT", TargetPattern: "new class", Complexity: "trivial"},
	}
	result, absorbed := consolidatePackages(patterns)
	if absorbed != 2 {
		t.Fatalf("expected 2 absorbed, got %d", absorbed)
	}
	if result[0].Complexity != "high" {
		t.Errorf("PACKAGE should inherit highest complexity 'high', got %q", result[0].Complexity)
	}
}

func TestWriteAndReadPatternsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patterns.json")

	original := &ExtractOutput{
		Source:   "sb3",
		Target:   "sb4",
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
		Source:   "httpcomponents-client-4",
		Target:   "httpcomponents-client-5",
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
		Source:   "sb3",
		Target:   "sb4",
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
