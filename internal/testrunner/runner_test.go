package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindTestFiles(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure matching real layout: tests/*.test.yaml
	subdir := filepath.Join(dir, "tests")
	os.MkdirAll(subdir, 0o755)

	for _, name := range []string{"core.test.yaml", "web.test.yaml"} {
		os.WriteFile(filepath.Join(subdir, name), []byte(""), 0o644)
	}
	// Non-test files should be ignored.
	os.WriteFile(filepath.Join(subdir, "data.yaml"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o644)

	files, err := FindTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 test files, got %d: %v", len(files), files)
	}
}

func TestFindTestFiles_flat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.test.yaml"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "b.test.yml"), []byte(""), 0o644)

	files, err := FindTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 test files, got %d", len(files))
	}
}

func TestFindTestFiles_empty(t *testing.T) {
	dir := t.TempDir()
	files, err := FindTestFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 test files, got %d", len(files))
	}
}

func TestResolveFilesRelativeToTestsDir(t *testing.T) {
	// Simulate what Run() does: bare filenames should be joined with TestsDir.
	testsDir := "/some/path/tests"
	files := []string{"core.test.yaml", "web.test.yaml"}

	cfg := Config{
		RulesDir: "/some/path/rules",
		TestsDir: testsDir,
		Files:    files,
	}

	// Replicate the resolution logic from Run().
	resolved := make([]string, len(cfg.Files))
	copy(resolved, cfg.Files)
	for i, f := range resolved {
		if !filepath.IsAbs(f) && !strings.Contains(f, string(filepath.Separator)) {
			resolved[i] = filepath.Join(cfg.TestsDir, f)
		}
	}

	for i, r := range resolved {
		expected := filepath.Join(testsDir, files[i])
		if r != expected {
			t.Errorf("resolved[%d] = %q, want %q", i, r, expected)
		}
	}
}

func TestResolveFilesAbsolutePathUntouched(t *testing.T) {
	// Absolute paths should not be modified.
	absPath := "/absolute/path/to/core.test.yaml"
	cfg := Config{
		TestsDir: "/some/path/tests",
		Files:    []string{absPath},
	}

	resolved := make([]string, len(cfg.Files))
	copy(resolved, cfg.Files)
	for i, f := range resolved {
		if !filepath.IsAbs(f) && !strings.Contains(f, string(filepath.Separator)) {
			resolved[i] = filepath.Join(cfg.TestsDir, f)
		}
	}

	if resolved[0] != absPath {
		t.Errorf("absolute path was modified: got %q, want %q", resolved[0], absPath)
	}
}

func TestResolveFilesRelativePathWithSeparator(t *testing.T) {
	// Relative paths with separators (e.g., "subdir/core.test.yaml") should not be modified.
	relPath := filepath.Join("subdir", "core.test.yaml")
	cfg := Config{
		TestsDir: "/some/path/tests",
		Files:    []string{relPath},
	}

	resolved := make([]string, len(cfg.Files))
	copy(resolved, cfg.Files)
	for i, f := range resolved {
		if !filepath.IsAbs(f) && !strings.Contains(f, string(filepath.Separator)) {
			resolved[i] = filepath.Join(cfg.TestsDir, f)
		}
	}

	if resolved[0] != relPath {
		t.Errorf("relative path with separator was modified: got %q, want %q", resolved[0], relPath)
	}
}
