package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanCache(t *testing.T) {
	cacheDir := t.TempDir()

	artifactDir := filepath.Join(cacheDir, "org.apache.httpcomponents", "httpclient", "4.5.14")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	jarPath := filepath.Join(artifactDir, "httpclient-4.5.14.jar")
	if err := os.WriteFile(jarPath, []byte("fake jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	classesPath := filepath.Join(artifactDir, "classes.txt")
	if err := os.WriteFile(classesPath, []byte("com/example/A.class\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	count, bytes, err := CleanCache(cacheDir)
	if err != nil {
		t.Fatalf("CleanCache: %v", err)
	}
	if count != 2 {
		t.Errorf("files removed = %d, want 2", count)
	}
	if bytes <= 0 {
		t.Error("expected positive bytes freed")
	}

	if _, err := os.Stat(artifactDir); !os.IsNotExist(err) {
		t.Error("expected artifact directory to be removed")
	}
}

func TestCleanCache_EmptyDir(t *testing.T) {
	cacheDir := t.TempDir()

	count, bytes, err := CleanCache(cacheDir)
	if err != nil {
		t.Fatalf("CleanCache: %v", err)
	}
	if count != 0 || bytes != 0 {
		t.Errorf("expected 0 files/0 bytes, got %d/%d", count, bytes)
	}
}

func TestCleanCache_NonExistentDir(t *testing.T) {
	count, bytes, err := CleanCache("/tmp/nonexistent-verify-cache-test")
	if err != nil {
		t.Fatalf("CleanCache: %v", err)
	}
	if count != 0 || bytes != 0 {
		t.Errorf("expected 0 files/0 bytes for nonexistent dir, got %d/%d", count, bytes)
	}
}
