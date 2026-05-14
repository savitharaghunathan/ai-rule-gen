package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupIntermediateArtifacts(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "verify-cache", "org.apache", "httpclient", "4.5.14"), 0o755)
	os.WriteFile(filepath.Join(dir, "verify-cache", "org.apache", "httpclient", "4.5.14", "classes.txt"), []byte("class list"), 0o644)
	os.MkdirAll(filepath.Join(dir, "contracts"), 0o755)
	os.WriteFile(filepath.Join(dir, "contracts", "rule-writer-input-1.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "patterns-1.json"), []byte("[]"), 0o644)
	os.WriteFile(filepath.Join(dir, "patterns-2.json"), []byte("[]"), 0o644)
	os.WriteFile(filepath.Join(dir, "patterns-gaps.json"), []byte("[]"), 0o644)
	os.WriteFile(filepath.Join(dir, "patterns.json"), []byte("[]"), 0o644)
	os.WriteFile(filepath.Join(dir, "report.yaml"), []byte("report"), 0o644)

	cleanupIntermediateArtifacts(dir)

	for _, gone := range []string{"verify-cache", "contracts", "patterns-1.json", "patterns-2.json", "patterns-gaps.json"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); err == nil {
			t.Errorf("expected %s to be removed", gone)
		}
	}
	for _, kept := range []string{"patterns.json", "report.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, kept)); err != nil {
			t.Errorf("expected %s to be kept, got error: %v", kept, err)
		}
	}
}
