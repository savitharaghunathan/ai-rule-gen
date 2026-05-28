package testrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// Result is the JSON output of a test run.
type Result struct {
	Passed       int      `json:"passed"`
	Failed       int      `json:"failed"`
	Total        int      `json:"total"`
	PassRate     float64  `json:"pass_rate"`
	FailedRules  []string `json:"failed_rules,omitempty"`
	TimedOutFiles []string `json:"timed_out_files,omitempty"`
	Output       string   `json:"output,omitempty"`
}

// ProgressFunc is called before and after each test file.
// When starting: passed=-1, total=fileCount, elapsed=0.
// When complete: passed/total are rule counts from that file.
type ProgressFunc func(file string, index, fileCount, passed, total int, timedOut bool, elapsed time.Duration)

// Config holds the inputs for a test run.
type Config struct {
	RulesDir       string
	TestsDir       string
	Files          []string      // specific test files to run (if empty, scan TestsDir)
	TestTimeout    time.Duration // per-test-file timeout (0 = no timeout)
	RetryTimeouts  bool          // retry timed-out files once after the initial run
	OnProgress     ProgressFunc  // optional callback after each test file
}

// Run executes kantra test for each .test.yaml file and returns pass/fail results.
// It does NOT stamp rules or generate reports — the caller handles those.
func Run(cfg Config) (*Result, error) {
	if err := checkKantra(); err != nil {
		return nil, err
	}

	testFiles := cfg.Files
	if len(testFiles) == 0 {
		var err error
		testFiles, err = FindTestFiles(cfg.TestsDir)
		if err != nil {
			return nil, fmt.Errorf("finding test files: %w", err)
		}
	} else {
		// Resolve bare filenames relative to TestsDir.
		for i, f := range testFiles {
			if !filepath.IsAbs(f) && !strings.Contains(f, string(filepath.Separator)) {
				testFiles[i] = filepath.Join(cfg.TestsDir, f)
			}
		}
	}
	if len(testFiles) == 0 {
		return nil, fmt.Errorf("no .test.yaml files found in %s", cfg.TestsDir)
	}

	var allOutput strings.Builder
	erroredRuleIDs, timedOutFiles, err := runFiles(testFiles, cfg.TestTimeout, &allOutput, cfg.OnProgress)
	if err != nil {
		return nil, err
	}

	if cfg.RetryTimeouts && len(timedOutFiles) > 0 {
		var retryPaths []string
		for _, f := range timedOutFiles {
			retryPaths = append(retryPaths, filepath.Join(cfg.TestsDir, f))
		}
		retryErrored, retryTimedOut, retryErr := runFiles(retryPaths, cfg.TestTimeout, &allOutput, cfg.OnProgress)
		if retryErr != nil {
			return nil, retryErr
		}
		erroredRuleIDs = retryErrored
		timedOutFiles = retryTimedOut
	}

	combinedOutput := allOutput.String()

	// Write combined kantra output.
	outputPath := filepath.Join(cfg.TestsDir, "kantra-output.txt")
	if err := os.WriteFile(outputPath, []byte(combinedOutput), 0o644); err != nil {
		return nil, fmt.Errorf("writing kantra output: %w", err)
	}

	// Determine which rule IDs to check: when running a subset of test files,
	// only check rules referenced by those files (not all rules in the directory).
	var checkIDs []string
	if len(cfg.Files) > 0 {
		seen := make(map[string]bool)
		for _, tf := range testFiles {
			ids, err := kantraparser.TestFileRuleIDs(tf)
			if err != nil {
				return nil, fmt.Errorf("parsing test file %s: %w", filepath.Base(tf), err)
			}
			for _, id := range ids {
				if !seen[id] {
					seen[id] = true
					checkIDs = append(checkIDs, id)
				}
			}
		}
	} else {
		allRules, err := rules.ReadRulesDir(cfg.RulesDir)
		if err != nil {
			return nil, fmt.Errorf("reading rules: %w", err)
		}
		for _, r := range allRules {
			checkIDs = append(checkIDs, r.RuleID)
		}
	}

	passedIDs, failedIDs := kantraparser.PassedAndFailed(combinedOutput, checkIDs, erroredRuleIDs...)

	passed := len(passedIDs)
	failed := len(failedIDs)
	total := passed + failed
	var passRate float64
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	return &Result{
		Passed:        passed,
		Failed:        failed,
		Total:         total,
		PassRate:      passRate,
		FailedRules:   failedIDs,
		TimedOutFiles: timedOutFiles,
		Output:        combinedOutput,
	}, nil
}

// FindTestFiles recursively finds .test.yaml/.test.yml files under dir.
func FindTestFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".test.yaml") || strings.HasSuffix(d.Name(), ".test.yml") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// RunKantraTest executes kantra test on a single test file and returns the combined output.
func RunKantraTest(testFile string) (string, error) {
	return runKantraTestWithTimeout(testFile, 0)
}

// runFiles runs kantra test on each file sequentially, appending output to w.
// Returns rule IDs from errored/timed-out files and the list of timed-out filenames.
func runFiles(testFiles []string, timeout time.Duration, w *strings.Builder, onProgress ProgressFunc) (erroredRuleIDs, timedOutFiles []string, err error) {
	n := len(testFiles)
	for i, tf := range testFiles {
		name := filepath.Base(tf)
		if onProgress != nil {
			onProgress(name, i+1, n, -1, 0, false, 0)
		}
		start := time.Now()
		out, runErr := runKantraTestWithTimeout(tf, timeout)
		elapsed := time.Since(start)
		w.WriteString(out)
		w.WriteString("\n")
		if runErr != nil {
			if isTimeout(runErr) {
				timedOutFiles = append(timedOutFiles, name)
				if ids, parseErr := kantraparser.TestFileRuleIDs(tf); parseErr == nil {
					erroredRuleIDs = append(erroredRuleIDs, ids...)
				}
				if onProgress != nil {
					onProgress(name, i+1, n, 0, 0, true, elapsed)
				}
				continue
			}
			if out == "" {
				return nil, nil, fmt.Errorf("kantra test %s: %w", name, runErr)
			}
			if _, total := kantraparser.ParseSummary(out); total == 0 {
				if ids, parseErr := kantraparser.TestFileRuleIDs(tf); parseErr == nil {
					erroredRuleIDs = append(erroredRuleIDs, ids...)
				}
			}
		}
		if onProgress != nil {
			passed, total := kantraparser.ParseSummary(out)
			onProgress(name, i+1, n, passed, total, false, elapsed)
		}
	}
	return erroredRuleIDs, timedOutFiles, nil
}

func runKantraTestWithTimeout(testFile string, timeout time.Duration) (string, error) {
	var cmd *exec.Cmd
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, "kantra", "test", "--run-local=true", testFile)
	} else {
		cmd = exec.Command("kantra", "test", "--run-local=true", testFile)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	return output, err
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if err == context.DeadlineExceeded {
		return true
	}
	var exitErr *exec.ExitError
	if ok := errors.As(err, &exitErr); ok && exitErr.ProcessState != nil {
		return exitErr.String() == "signal: killed"
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// ReadSourceTarget extracts source and target from ruleset.yaml labels.
func ReadSourceTarget(rulesDir string) (source, target string) {
	rsPath := filepath.Join(rulesDir, "ruleset.yaml")
	rs, err := rules.ReadRuleset(rsPath)
	if err != nil {
		return "", ""
	}
	for _, l := range rs.Labels {
		if v, ok := strings.CutPrefix(l, "konveyor.io/source="); ok {
			source = v
		}
		if v, ok := strings.CutPrefix(l, "konveyor.io/target="); ok {
			target = v
		}
	}
	return source, target
}

// checkKantra verifies that kantra is installed and in PATH.
func checkKantra() error {
	_, err := exec.LookPath("kantra")
	if err != nil {
		return fmt.Errorf("kantra not found in PATH: install kantra before running tests")
	}
	return nil
}
