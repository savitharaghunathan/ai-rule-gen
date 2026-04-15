package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Workspace manages the output directory for a migration path.
type Workspace struct {
	Root string
}

// New creates a workspace for the given source→target migration.
// The root directory is output/<source>-to-<target>/.
func New(outputBase, source, target string) (*Workspace, error) {
	dirName := sanitizePath(source) + "-to-" + sanitizePath(target)
	root := filepath.Join(outputBase, dirName)
	// Ensure the resolved path stays within outputBase to prevent traversal.
	absBase, err := filepath.Abs(outputBase)
	if err != nil {
		return nil, fmt.Errorf("resolving output base: %w", err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace root: %w", err)
	}
	if !strings.HasPrefix(absRoot, absBase+string(filepath.Separator)) && absRoot != absBase {
		return nil, fmt.Errorf("workspace path %q escapes output base %q", absRoot, absBase)
	}
	w := &Workspace{Root: root}
	if err := w.init(); err != nil {
		return nil, err
	}
	return w, nil
}

// NewFromPath creates a workspace from an existing directory path.
func NewFromPath(root string) (*Workspace, error) {
	w := &Workspace{Root: root}
	if err := w.init(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Workspace) init() error {
	dirs := []string{
		w.RulesDir(),
		w.TestsDir(),
		w.TestDataDir(),
		w.ConfidenceDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating workspace directory %s: %w", dir, err)
		}
	}
	return nil
}

// sanitizePath strips directory separators and traversal sequences from a path component.
func sanitizePath(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "..", "")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

// RulesDir returns the path to the rules directory.
func (w *Workspace) RulesDir() string {
	return filepath.Join(w.Root, "rules")
}

// TestsDir returns the path to the tests directory.
func (w *Workspace) TestsDir() string {
	return filepath.Join(w.Root, "tests")
}

// TestDataDir returns the path to the test data directory.
func (w *Workspace) TestDataDir() string {
	return filepath.Join(w.Root, "tests", "data")
}

// ConfidenceDir returns the path to the confidence scores directory.
func (w *Workspace) ConfidenceDir() string {
	return filepath.Join(w.Root, "confidence")
}

// RulesetPath returns the path to the ruleset.yaml file.
func (w *Workspace) RulesetPath() string {
	return filepath.Join(w.RulesDir(), "ruleset.yaml")
}

// RulesFilePath returns the path for a rules file by concern name.
func (w *Workspace) RulesFilePath(concern string) string {
	if concern == "" {
		concern = "general"
	}
	return filepath.Join(w.RulesDir(), concern+".yaml")
}

// ScoresPath returns the path to the confidence scores file.
func (w *Workspace) ScoresPath() string {
	return filepath.Join(w.ConfidenceDir(), "scores.yaml")
}

// RulesReportPath returns the path to the rules report file.
func (w *Workspace) RulesReportPath() string {
	return filepath.Join(w.Root, "rules-report.yaml")
}
