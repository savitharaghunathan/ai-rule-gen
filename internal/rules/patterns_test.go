package rules

import (
	"os"
	"path/filepath"
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

	merged := MergePatterns(parts)
	if len(merged.Patterns) != 3 {
		t.Fatalf("expected 3 patterns after dedup, got %d", len(merged.Patterns))
	}
	if merged.Patterns[0].SourcePattern != "A" {
		t.Errorf("first occurrence should win, got %q", merged.Patterns[0].SourcePattern)
	}
}

func TestMergePatterns_Empty(t *testing.T) {
	merged := MergePatterns(nil)
	if merged == nil {
		t.Fatal("expected non-nil result")
	}
	if len(merged.Patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(merged.Patterns))
	}
}

func TestMergePatterns_PreservesMetadata(t *testing.T) {
	parts := []*ExtractOutput{
		{Source: "sb3", Target: "sb4", Language: "java"},
		{Source: "sb3", Target: "sb4", Language: "java"},
	}
	merged := MergePatterns(parts)
	if merged.Source != "sb3" || merged.Target != "sb4" || merged.Language != "java" {
		t.Errorf("metadata not preserved: %+v", merged)
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
