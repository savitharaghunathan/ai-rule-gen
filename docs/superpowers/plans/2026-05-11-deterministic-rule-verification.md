# Deterministic Rule Verification — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a post-extraction verification step that checks whether extracted FQNs actually exist in published source library artifacts, stamping rules with `konveyor.io/source-verified=true|false`.

**Architecture:** A new `internal/verify` package with a `Verifier` interface per language. Java verifier downloads JARs from Maven Central, runs `jar tf` to list classes, and checks FQN existence. A new `cmd/verify` CLI command integrates into the pipeline between `merge-patterns` and `construct`. Results are stamped as labels on constructed rules and reported in the final report.

**Tech Stack:** Go 1.25.7, `net/http` for Maven Central downloads, `os/exec` for `jar tf`, existing `internal/rules` and `internal/workspace` packages.

**Note:** The spec lists `registry.go` in the architecture. For Phase 1 (Java only), registry logic lives directly in `java.go`. Extract a shared `registry.go` abstraction when adding the second language verifier.

---

### Task 1: Add `source_artifact` field to MigrationPattern

**Files:**
- Modify: `internal/rules/patterns.go`
- Modify: `internal/rules/patterns_test.go`

- [ ] **Step 1: Write failing test for source_artifact round-trip**

Add to `internal/rules/patterns_test.go`:

```go
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
					Version:   "4.5.14",
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/rules/ -run TestWriteAndReadPatternsFile_SourceArtifact -v`
Expected: FAIL — `ArtifactCoordinates` type and `SourceArtifact` field don't exist

- [ ] **Step 3: Add ArtifactCoordinates type and SourceArtifact field**

In `internal/rules/patterns.go`, add the type before `ExtractOutput`:

```go
// ArtifactCoordinates identifies a published library artifact in a package registry.
type ArtifactCoordinates struct {
	GroupID    string `json:"group_id"`
	ArtifactID string `json:"artifact_id"`
	Version   string `json:"version"`
}
```

Add the field to `MigrationPattern`, after the `Message` field:

```go
	// Source artifact coordinates for deterministic verification.
	// When set, the verifier downloads this artifact and checks that SourceFQN exists in it.
	SourceArtifact *ArtifactCoordinates `json:"source_artifact,omitempty"`
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/rules/ -v`
Expected: ALL PASS (both new tests and existing tests)

- [ ] **Step 5: Commit**

```bash
git add internal/rules/patterns.go internal/rules/patterns_test.go
git commit -m "feat: add source_artifact field to MigrationPattern for deterministic verification"
```

---

### Task 2: Create verify result types

**Files:**
- Create: `internal/verify/result.go`
- Create: `internal/verify/result_test.go`

- [ ] **Step 1: Write failing test for result types**

Create `internal/verify/result_test.go`:

```go
package verify

import (
	"testing"
)

func TestResultStatus_Constants(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusVerified, "verified"},
		{StatusNotFound, "not_found"},
		{StatusOffline, "registry_offline"},
		{StatusSkipped, "skipped"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("status %v = %q, want %q", tt.status, string(tt.status), tt.want)
		}
	}
}

func TestSummary(t *testing.T) {
	results := []Result{
		{PatternIndex: 0, SourceFQN: "com.example.A", Status: StatusVerified, Evidence: "found in foo-1.0.jar"},
		{PatternIndex: 1, SourceFQN: "com.example.B", Status: StatusVerified, Evidence: "found in foo-1.0.jar"},
		{PatternIndex: 2, SourceFQN: "com.example.C", Status: StatusNotFound, Reason: "not in foo-1.0.jar"},
		{PatternIndex: 3, Status: StatusSkipped, Reason: "no source_artifact"},
	}

	s := Summarize(results)
	if s.Verified != 2 {
		t.Errorf("verified = %d, want 2", s.Verified)
	}
	if s.NotFound != 1 {
		t.Errorf("not_found = %d, want 1", s.NotFound)
	}
	if s.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", s.Skipped)
	}
	if len(s.NotFoundDetails) != 1 {
		t.Fatalf("not_found_details length = %d, want 1", len(s.NotFoundDetails))
	}
	if s.NotFoundDetails[0].SourceFQN != "com.example.C" {
		t.Errorf("not_found fqn = %q, want com.example.C", s.NotFoundDetails[0].SourceFQN)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement result types**

Create `internal/verify/result.go`:

```go
package verify

// Status represents the verification outcome for a single pattern.
type Status string

const (
	StatusVerified Status = "verified"
	StatusNotFound Status = "not_found"
	StatusOffline  Status = "registry_offline"
	StatusSkipped  Status = "skipped"
)

// Result holds the verification outcome for a single pattern.
type Result struct {
	PatternIndex int      `json:"pattern_index"`
	SourceFQN    string   `json:"source_fqn,omitempty"`
	Status       Status   `json:"status"`
	Evidence     string   `json:"evidence,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
}

// Summary holds aggregate verification stats.
type Summary struct {
	Verified        int      `json:"verified"`
	NotFound        int      `json:"not_found"`
	Skipped         int      `json:"skipped"`
	Offline         int      `json:"offline"`
	NotFoundDetails []Result `json:"not_found_details,omitempty"`
}

// Summarize computes aggregate stats from a slice of results.
func Summarize(results []Result) *Summary {
	s := &Summary{}
	for _, r := range results {
		switch r.Status {
		case StatusVerified:
			s.Verified++
		case StatusNotFound:
			s.NotFound++
			s.NotFoundDetails = append(s.NotFoundDetails, r)
		case StatusSkipped:
			s.Skipped++
		case StatusOffline:
			s.Offline++
		}
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/verify/result.go internal/verify/result_test.go
git commit -m "feat: add verify result types and summary aggregation"
```

---

### Task 3: Create the Verifier interface and Java verifier

**Files:**
- Create: `internal/verify/verify.go`
- Create: `internal/verify/java.go`
- Create: `internal/verify/java_test.go`

- [ ] **Step 1: Write failing tests for FQN-to-path conversion and class listing lookup**

Create `internal/verify/java_test.go`:

```go
package verify

import (
	"testing"
)

func TestFQNToClassPath(t *testing.T) {
	tests := []struct {
		fqn  string
		want string
	}{
		{"org.apache.http.client.HttpClient", "org/apache/http/client/HttpClient.class"},
		{"com.example.MyClass", "com/example/MyClass.class"},
		{"Single", "Single.class"},
	}
	for _, tt := range tests {
		got := fqnToClassPath(tt.fqn)
		if got != tt.want {
			t.Errorf("fqnToClassPath(%q) = %q, want %q", tt.fqn, got, tt.want)
		}
	}
}

func TestFindInClassList(t *testing.T) {
	classLines := []string{
		"META-INF/MANIFEST.MF",
		"org/apache/http/client/HttpClient.class",
		"org/apache/http/client/methods/HttpGet.class",
		"org/apache/http/impl/client/CloseableHttpClient.class",
	}

	tests := []struct {
		fqn       string
		wantFound bool
	}{
		{"org.apache.http.client.HttpClient", true},
		{"org.apache.http.client.methods.HttpGet", true},
		{"org.apache.http.client.NonExistent", false},
	}
	for _, tt := range tests {
		found := findInClassList(classLines, fqnToClassPath(tt.fqn))
		if found != tt.wantFound {
			t.Errorf("findInClassList(%q) = %v, want %v", tt.fqn, found, tt.wantFound)
		}
	}
}

func TestFindSuggestions(t *testing.T) {
	classLines := []string{
		"org/apache/http/client/HttpClient.class",
		"org/apache/hc/client5/http/classic/HttpClient.class",
		"org/apache/http/impl/client/CloseableHttpClient.class",
	}

	suggestions := findSuggestions(classLines, "HttpClient")
	if len(suggestions) < 1 {
		t.Fatal("expected at least one suggestion")
	}

	found := false
	for _, s := range suggestions {
		if s == "org.apache.http.client.HttpClient" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected org.apache.http.client.HttpClient in suggestions, got %v", suggestions)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -run TestFQN -v`
Expected: FAIL — functions don't exist

- [ ] **Step 3: Create Verifier interface**

Create `internal/verify/verify.go`:

```go
package verify

import (
	"fmt"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// Verifier checks whether extracted patterns reference real artifacts.
type Verifier interface {
	Verify(pattern rules.MigrationPattern) (Result, error)
	Language() string
}

// NewVerifier returns the appropriate verifier for the given language.
// Returns nil if no verifier is available for the language.
func NewVerifier(language, cacheDir string) Verifier {
	switch language {
	case "java":
		return NewJavaVerifier(cacheDir)
	default:
		return nil
	}
}

// Run verifies all patterns in an ExtractOutput using the appropriate language verifier.
// If no verifier exists for the language, all patterns are skipped.
func Run(extract *rules.ExtractOutput, cacheDir string) ([]Result, error) {
	v := NewVerifier(extract.Language, cacheDir)

	results := make([]Result, 0, len(extract.Patterns))
	for i, p := range extract.Patterns {
		if v == nil {
			results = append(results, Result{
				PatternIndex: i,
				SourceFQN:    p.SourceFQN,
				Status:       StatusSkipped,
				Reason:       fmt.Sprintf("no verifier for language %q", extract.Language),
			})
			continue
		}
		r, err := v.Verify(p)
		if err != nil {
			return nil, fmt.Errorf("verifying pattern %d (%s): %w", i, p.SourceFQN, err)
		}
		r.PatternIndex = i
		results = append(results, r)
	}
	return results, nil
}
```

- [ ] **Step 4: Implement Java verifier helper functions**

Create `internal/verify/java.go`:

```go
package verify

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// JavaVerifier checks FQNs against published Maven Central JARs.
type JavaVerifier struct {
	cacheDir   string
	httpClient *http.Client
}

// NewJavaVerifier creates a verifier that downloads JARs from Maven Central.
func NewJavaVerifier(cacheDir string) *JavaVerifier {
	return &JavaVerifier{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (v *JavaVerifier) Language() string { return "java" }

func (v *JavaVerifier) Verify(pattern rules.MigrationPattern) (Result, error) {
	if pattern.DependencyName != "" {
		return Result{
			SourceFQN: pattern.DependencyName,
			Status:    StatusVerified,
			Evidence:  "dependency patterns verified by Maven pre-check",
		}, nil
	}

	if pattern.SourceFQN == "" {
		return Result{
			Status: StatusSkipped,
			Reason: "no source_fqn to verify",
		}, nil
	}

	if pattern.SourceArtifact == nil {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusSkipped,
			Reason:    "no source_artifact metadata",
		}, nil
	}

	sa := pattern.SourceArtifact
	classLines, err := v.getClassList(sa.GroupID, sa.ArtifactID, sa.Version)
	if err != nil {
		if isNetworkError(err) {
			return Result{
				SourceFQN: pattern.SourceFQN,
				Status:    StatusOffline,
				Reason:    fmt.Sprintf("Maven Central unreachable: %v", err),
			}, nil
		}
		return Result{}, err
	}

	target := fqnToClassPath(pattern.SourceFQN)
	if findInClassList(classLines, target) {
		jarName := fmt.Sprintf("%s-%s.jar", sa.ArtifactID, sa.Version)
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusVerified,
			Evidence:  fmt.Sprintf("found in %s", jarName),
		}, nil
	}

	className := classNameFromFQN(pattern.SourceFQN)
	suggestions := findSuggestions(classLines, className)
	jarName := fmt.Sprintf("%s-%s.jar", sa.ArtifactID, sa.Version)
	return Result{
		SourceFQN:   pattern.SourceFQN,
		Status:      StatusNotFound,
		Reason:      fmt.Sprintf("not found in %s", jarName),
		Suggestions: suggestions,
	}, nil
}

// getClassList returns the class listing for a JAR, using cache if available.
func (v *JavaVerifier) getClassList(groupID, artifactID, version string) ([]string, error) {
	cacheDir := filepath.Join(v.cacheDir, groupID, artifactID, version)
	classesFile := filepath.Join(cacheDir, "classes.txt")

	if data, err := os.ReadFile(classesFile); err == nil {
		return splitLines(string(data)), nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}

	jarPath := filepath.Join(cacheDir, fmt.Sprintf("%s-%s.jar", artifactID, version))
	if err := v.downloadJAR(groupID, artifactID, version, jarPath); err != nil {
		return nil, err
	}

	lines, err := listJARClasses(jarPath)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(classesFile, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return nil, fmt.Errorf("writing classes cache: %w", err)
	}

	return lines, nil
}

// downloadJAR fetches a JAR from Maven Central.
func (v *JavaVerifier) downloadJAR(groupID, artifactID, version, destPath string) error {
	groupPath := strings.ReplaceAll(groupID, ".", "/")
	url := fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.jar",
		groupPath, artifactID, version, artifactID, version)

	resp, err := v.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Maven Central returned %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing %s: %w", destPath, err)
	}
	return nil
}

// listJARClasses runs `jar tf` and returns all entries.
func listJARClasses(jarPath string) ([]string, error) {
	cmd := exec.Command("jar", "tf", jarPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("jar tf %s: %w", jarPath, err)
	}
	return splitLines(string(out)), nil
}

// fqnToClassPath converts a fully-qualified name to a .class file path.
// "org.apache.http.client.HttpClient" → "org/apache/http/client/HttpClient.class"
func fqnToClassPath(fqn string) string {
	return strings.ReplaceAll(fqn, ".", "/") + ".class"
}

// findInClassList checks if a class path exists in the listing.
func findInClassList(classLines []string, classPath string) bool {
	for _, line := range classLines {
		if line == classPath {
			return true
		}
	}
	return false
}

// classNameFromFQN extracts the simple class name from a FQN.
// "org.apache.http.client.HttpClient" → "HttpClient"
func classNameFromFQN(fqn string) string {
	parts := strings.Split(fqn, ".")
	return parts[len(parts)-1]
}

// findSuggestions returns FQNs from the class list that share the same simple class name.
func findSuggestions(classLines []string, className string) []string {
	suffix := "/" + className + ".class"
	var suggestions []string
	for _, line := range classLines {
		if strings.HasSuffix(line, suffix) || line == className+".class" {
			fqn := strings.TrimSuffix(line, ".class")
			fqn = strings.ReplaceAll(fqn, "/", ".")
			suggestions = append(suggestions, fqn)
		}
	}
	return suggestions
}

func isNetworkError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "i/o timeout")
}

func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
```

- [ ] **Step 5: Run all verify tests**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/verify/verify.go internal/verify/java.go internal/verify/java_test.go
git commit -m "feat: add Verifier interface and Java verifier with JAR class listing"
```

---

### Task 4: Write integration test for Java verifier against Maven Central

**Files:**
- Create: `internal/verify/java_integration_test.go`

- [ ] **Step 1: Write integration test with real Maven Central artifact**

Create `internal/verify/java_integration_test.go`:

```go
//go:build integration

package verify

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestJavaVerifier_RealArtifact(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewJavaVerifier(cacheDir)

	tests := []struct {
		name    string
		pattern rules.MigrationPattern
		want    Status
	}{
		{
			name: "known FQN in httpclient 4.5.14",
			pattern: rules.MigrationPattern{
				SourceFQN:    "org.apache.http.client.HttpClient",
				ProviderType: "java",
				SourceArtifact: &rules.ArtifactCoordinates{
					GroupID:    "org.apache.httpcomponents",
					ArtifactID: "httpclient",
					Version:   "4.5.14",
				},
			},
			want: StatusVerified,
		},
		{
			name: "hallucinated FQN",
			pattern: rules.MigrationPattern{
				SourceFQN:    "org.apache.http.client.FakeNonExistentClass",
				ProviderType: "java",
				SourceArtifact: &rules.ArtifactCoordinates{
					GroupID:    "org.apache.httpcomponents",
					ArtifactID: "httpclient",
					Version:   "4.5.14",
				},
			},
			want: StatusNotFound,
		},
		{
			name: "dependency pattern — auto verified",
			pattern: rules.MigrationPattern{
				DependencyName: "org.apache.httpcomponents.httpclient",
			},
			want: StatusVerified,
		},
		{
			name: "no source_artifact — skipped",
			pattern: rules.MigrationPattern{
				SourceFQN:    "org.apache.http.client.HttpClient",
				ProviderType: "java",
			},
			want: StatusSkipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.Verify(tt.pattern)
			if err != nil {
				t.Fatalf("Verify() error: %v", err)
			}
			if result.Status != tt.want {
				t.Errorf("status = %q, want %q (evidence: %s, reason: %s)",
					result.Status, tt.want, result.Evidence, result.Reason)
			}
		})
	}
}

func TestJavaVerifier_CacheReuse(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewJavaVerifier(cacheDir)

	pattern := rules.MigrationPattern{
		SourceFQN:    "org.apache.http.client.HttpClient",
		ProviderType: "java",
		SourceArtifact: &rules.ArtifactCoordinates{
			GroupID:    "org.apache.httpcomponents",
			ArtifactID: "httpclient",
			Version:   "4.5.14",
		},
	}

	// First call downloads
	r1, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if r1.Status != StatusVerified {
		t.Fatalf("first verify status = %q, want verified", r1.Status)
	}

	// Second call should use cache (same result, faster)
	r2, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("second verify: %v", err)
	}
	if r2.Status != StatusVerified {
		t.Fatalf("second verify status = %q, want verified", r2.Status)
	}
}

func TestJavaVerifier_Suggestions(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewJavaVerifier(cacheDir)

	result, err := v.Verify(rules.MigrationPattern{
		SourceFQN:    "org.apache.http.client.FakeNonExistentClass",
		ProviderType: "java",
		SourceArtifact: &rules.ArtifactCoordinates{
			GroupID:    "org.apache.httpcomponents",
			ArtifactID: "httpclient",
			Version:   "4.5.14",
		},
	})
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}

	// "FakeNonExistentClass" shouldn't match anything, so suggestions should be empty
	if len(result.Suggestions) != 0 {
		t.Errorf("expected no suggestions for fake class, got %v", result.Suggestions)
	}

	// Try with a real class name in wrong package
	result2, err := v.Verify(rules.MigrationPattern{
		SourceFQN:    "com.wrong.package.HttpClient",
		ProviderType: "java",
		SourceArtifact: &rules.ArtifactCoordinates{
			GroupID:    "org.apache.httpcomponents",
			ArtifactID: "httpclient",
			Version:   "4.5.14",
		},
	})
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}

	if result2.Status != StatusNotFound {
		t.Fatalf("status = %q, want not_found", result2.Status)
	}
	if len(result2.Suggestions) == 0 {
		t.Error("expected suggestions for HttpClient in wrong package")
	}
}
```

- [ ] **Step 2: Run integration test**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -tags=integration -run TestJavaVerifier -v -timeout 60s`
Expected: ALL PASS (requires network access and `jar` command)

- [ ] **Step 3: Commit**

```bash
git add internal/verify/java_integration_test.go
git commit -m "test: add integration tests for Java verifier against Maven Central"
```

---

### Task 5: Add source-verified label stamping

**Files:**
- Modify: `internal/rules/labels.go`
- Modify: `internal/rules/labels_test.go`

- [ ] **Step 1: Write failing test for source-verified stamping**

Add to `internal/rules/labels_test.go`:

```go
func TestStampVerificationResults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	ruleList := []Rule{
		{RuleID: "rule-001", Message: "verified rule", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.A", LocationImport)},
		{RuleID: "rule-002", Message: "not found rule", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.B", LocationImport)},
		{RuleID: "rule-003", Message: "not stamped", Labels: []string{"konveyor.io/test-result=untested"}, When: NewJavaReferenced("com.example.C", LocationImport)},
	}

	if err := WriteRulesFile(path, ruleList); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := StampVerificationResults(dir, []string{"rule-001"}, []string{"rule-002"})
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}

	got, err := ReadRulesFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	tests := []struct {
		ruleID    string
		wantLabel string
		wantFound bool
	}{
		{"rule-001", "konveyor.io/source-verified=true", true},
		{"rule-002", "konveyor.io/source-verified=false", true},
		{"rule-003", "konveyor.io/source-verified=", false},
	}

	for _, tt := range tests {
		for _, r := range got {
			if r.RuleID != tt.ruleID {
				continue
			}
			found := false
			for _, l := range r.Labels {
				if strings.HasPrefix(l, "konveyor.io/source-verified=") {
					if !tt.wantFound {
						t.Errorf("%s: unexpected source-verified label %q", tt.ruleID, l)
					}
					if l != tt.wantLabel {
						t.Errorf("%s: label = %q, want %q", tt.ruleID, l, tt.wantLabel)
					}
					found = true
				}
			}
			if tt.wantFound && !found {
				t.Errorf("%s: expected label %q, got labels %v", tt.ruleID, tt.wantLabel, r.Labels)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/rules/ -run TestStampVerificationResults -v`
Expected: FAIL — `StampVerificationResults` doesn't exist

- [ ] **Step 3: Implement StampVerificationResults**

Add to `internal/rules/labels.go`:

```go
// StampVerificationResults updates rule files with source-verified labels.
func StampVerificationResults(rulesDir string, verified, notFound []string) error {
	verifiedSet := make(map[string]bool)
	for _, id := range verified {
		verifiedSet[id] = true
	}
	notFoundSet := make(map[string]bool)
	for _, id := range notFound {
		notFoundSet[id] = true
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return fmt.Errorf("reading rules dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if name == "ruleset.yaml" || name == "ruleset.yml" {
			continue
		}

		path := filepath.Join(rulesDir, name)
		ruleList, err := ReadRulesFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}

		modified := false
		for i := range ruleList {
			r := &ruleList[i]
			var newLabel string
			if verifiedSet[r.RuleID] {
				newLabel = "konveyor.io/source-verified=true"
			} else if notFoundSet[r.RuleID] {
				newLabel = "konveyor.io/source-verified=false"
			} else {
				continue
			}

			updated := false
			for j, l := range r.Labels {
				if strings.HasPrefix(l, "konveyor.io/source-verified=") {
					r.Labels[j] = newLabel
					updated = true
					break
				}
			}
			if !updated {
				r.Labels = append(r.Labels, newLabel)
			}
			modified = true
		}

		if modified {
			if err := WriteRulesFile(path, ruleList); err != nil {
				return fmt.Errorf("writing %s: %w", name, err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/rules/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/rules/labels.go internal/rules/labels_test.go
git commit -m "feat: add source-verified label stamping for verification results"
```

---

### Task 6: Add verification stats to Report

**Files:**
- Modify: `internal/workspace/report.go`
- Modify: `internal/workspace/report_test.go`

- [ ] **Step 1: Write failing test for report with verification stats**

Read the existing test file first, then add to `internal/workspace/report_test.go`:

```go
func TestBuildReport_WithVerification(t *testing.T) {
	r := BuildReport("sb3", "sb4", 20, 15, 3, 2, []string{"r1", "r2", "r3"}, []string{"k1", "k2"})
	r.Verification = &VerificationStats{
		Verified: 14,
		NotFound: 3,
		Skipped:  3,
		NotFoundRules: []NotFoundRule{
			{RuleID: "r1", SourceFQN: "com.example.Fake", Reason: "not found in foo-1.0.jar"},
		},
	}

	if r.Verification.Verified != 14 {
		t.Errorf("verified = %d, want 14", r.Verification.Verified)
	}
	if len(r.Verification.NotFoundRules) != 1 {
		t.Fatalf("not_found_rules length = %d, want 1", len(r.Verification.NotFoundRules))
	}
	if r.Verification.NotFoundRules[0].RuleID != "r1" {
		t.Errorf("not_found rule_id = %q, want r1", r.Verification.NotFoundRules[0].RuleID)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "report.yaml")
	if err := WriteReport(path, r); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "verification:") {
		t.Error("report YAML missing verification section")
	}
	if !strings.Contains(content, "verified: 14") {
		t.Error("report YAML missing verified count")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/workspace/ -run TestBuildReport_WithVerification -v`
Expected: FAIL — `VerificationStats`, `NotFoundRule` types don't exist

- [ ] **Step 3: Add verification types and field to Report**

In `internal/workspace/report.go`, add the types and field:

```go
// VerificationStats holds aggregate results from deterministic FQN verification.
type VerificationStats struct {
	Verified      int            `yaml:"verified" json:"verified"`
	NotFound      int            `yaml:"not_found" json:"not_found"`
	Skipped       int            `yaml:"skipped" json:"skipped"`
	NotFoundRules []NotFoundRule `yaml:"not_found_rules,omitempty" json:"not_found_rules,omitempty"`
}

// NotFoundRule records a rule whose source FQN was not found in the published artifact.
type NotFoundRule struct {
	RuleID    string `yaml:"rule_id" json:"rule_id"`
	SourceFQN string `yaml:"source_fqn" json:"source_fqn"`
	Reason    string `yaml:"reason" json:"reason"`
}
```

Add to the `Report` struct, after `KantraLimitationRule`:

```go
	Verification     *VerificationStats `yaml:"verification,omitempty" json:"verification,omitempty"`
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/workspace/ -v`
Expected: ALL PASS (both new and existing tests)

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/report.go internal/workspace/report_test.go
git commit -m "feat: add verification stats to pipeline report"
```

---

### Task 7: Create `cmd/verify` CLI command

**Files:**
- Create: `cmd/verify/main.go`

- [ ] **Step 1: Implement the CLI command**

Create `cmd/verify/main.go`:

```go
package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/verify"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	patternsPath := flag.String("patterns", "", "Path to patterns.json (required)")
	language := flag.String("language", "", "Override language (default: read from patterns.json)")
	outputPath := flag.String("output", "", "Write verification results to this JSON file")
	cacheDir := flag.String("cache", "", "Directory for cached JARs/artifacts (default: <patterns-dir>/verify-cache)")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *patternsPath == "" {
		cli.Fail("invalid_arguments", "--patterns is required", "verify", "set --patterns to a patterns.json file", nil)
	}

	extract, err := rules.ReadPatternsFile(*patternsPath)
	if err != nil {
		cli.Fail("read_patterns_failed", err.Error(), "verify", "check patterns.json path and format", map[string]string{"patterns": *patternsPath})
	}

	lang := *language
	if lang == "" {
		lang = extract.Language
	}
	if lang != "" {
		extract.Language = lang
	}

	cache := *cacheDir
	if cache == "" {
		dir := filepath.Dir(*patternsPath)
		cache = filepath.Join(dir, "verify-cache")
	}

	cli.Log("verifying %d patterns for language %q", len(extract.Patterns), extract.Language)

	results, err := verify.Run(extract, cache)
	if err != nil {
		cli.Fail("verify_failed", err.Error(), "verify", "check network connectivity and jar command availability", nil)
	}

	summary := verify.Summarize(results)

	cli.Log("verification complete: %d verified, %d not_found, %d skipped, %d offline",
		summary.Verified, summary.NotFound, summary.Skipped, summary.Offline)

	if *outputPath != "" {
		output := struct {
			Results []verify.Result  `json:"results"`
			Summary *verify.Summary `json:"summary"`
		}{
			Results: results,
			Summary: summary,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			cli.Fail("marshal_failed", err.Error(), "verify", "unexpected marshal error", nil)
		}
		if err := os.WriteFile(*outputPath, data, 0o644); err != nil {
			cli.Fail("write_failed", err.Error(), "verify", "check output path permissions", map[string]string{"output": *outputPath})
		}
	}

	cli.WriteJSON(map[string]interface{}{
		"status":    "ok",
		"verified":  summary.Verified,
		"not_found": summary.NotFound,
		"skipped":   summary.Skipped,
		"offline":   summary.Offline,
	})
}
```

Note: you will need to add `"path/filepath"` to the imports.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go build ./cmd/verify/`
Expected: Builds successfully

- [ ] **Step 3: Run full test suite to check nothing is broken**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./... 2>&1 | tail -20`
Expected: All existing tests pass

- [ ] **Step 4: Commit**

```bash
git add cmd/verify/main.go
git commit -m "feat: add cmd/verify CLI for deterministic pattern verification"
```

---

### Task 8: Add VerifyCacheDir to Workspace

**Files:**
- Modify: `internal/workspace/workspace.go`
- Modify: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/workspace/workspace_test.go`:

```go
func TestWorkspace_VerifyCacheDir(t *testing.T) {
	dir := t.TempDir()
	w, err := NewFromPath(dir)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}

	want := filepath.Join(dir, "verify-cache")
	got := w.VerifyCacheDir()
	if got != want {
		t.Errorf("VerifyCacheDir() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/workspace/ -run TestWorkspace_VerifyCacheDir -v`
Expected: FAIL — `VerifyCacheDir` doesn't exist

- [ ] **Step 3: Add VerifyCacheDir method**

In `internal/workspace/workspace.go`, add after the `ScoresPath` method:

```go
// VerifyCacheDir returns the path for cached verification artifacts (JARs, class listings).
func (w *Workspace) VerifyCacheDir() string {
	return filepath.Join(w.Root, "verify-cache")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/workspace/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "feat: add VerifyCacheDir to Workspace for artifact cache path"
```

---

### Task 9: Update construct to propagate pattern index → rule ID mapping

**Files:**
- Modify: `internal/construct/construct.go`
- Modify: `internal/construct/construct_test.go`

- [ ] **Step 1: Write failing test for PatternRuleMap**

Add to `internal/construct/construct_test.go`:

```go
func TestRun_PatternRuleMap(t *testing.T) {
	extract := &rules.ExtractOutput{
		Source:   "sb3",
		Target:   "sb4",
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/construct/ -run TestRun_PatternRuleMap -v`
Expected: FAIL — `PatternRuleMap` field doesn't exist on `Result`

- [ ] **Step 3: Add PatternRuleMap to Result and populate it**

In `internal/construct/construct.go`, add to the `Result` struct:

```go
	PatternRuleMap map[int]string `json:"pattern_rule_map,omitempty"`
```

In the `Run` function, create and populate the map. After the `for _, p := range extract.Patterns` loop, add a second loop that tracks the index-to-ruleID mapping. Replace the existing loop:

```go
	patternRuleMap := make(map[int]string)
	prefix := rulePrefix(extract.Source, extract.Target)
	idGen := rules.NewIDGenerator(prefix)

	grouped := make(map[string][]rules.Rule)

	for i, p := range extract.Patterns {
		rule := patternToRule(p, idGen, extract.Source, extract.Target)
		patternRuleMap[i] = rule.RuleID
		concern := p.Concern
		if concern == "" {
			concern = "general"
		}
		grouped[concern] = append(grouped[concern], rule)
	}
```

And add `PatternRuleMap: patternRuleMap,` to the return `Result` struct.

- [ ] **Step 4: Run all construct tests**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/construct/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/construct/construct.go internal/construct/construct_test.go
git commit -m "feat: add PatternRuleMap to construct Result for verification label mapping"
```

---

### Task 10: End-to-end wiring test

**Files:**
- Create: `internal/verify/e2e_test.go`

- [ ] **Step 1: Write end-to-end test that verifies patterns, constructs rules, and stamps labels**

Create `internal/verify/e2e_test.go`:

```go
package verify_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/construct"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/verify"
)

func TestEndToEnd_VerifyAndStamp(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	rulesDir := filepath.Join(dir, "rules")

	extract := &rules.ExtractOutput{
		Source:   "lib-v1",
		Target:   "lib-v2",
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "A",
				SourceFQN:     "com.example.RealClass",
				Rationale:     "class moved",
				Complexity:    "low",
				Category:      "mandatory",
				ProviderType:  "java",
				LocationType:  "IMPORT",
				// No SourceArtifact — should be skipped
			},
			{
				SourcePattern:  "B",
				DependencyName: "org.example.dep",
				Rationale:      "dep removed",
				Complexity:     "low",
				Category:       "mandatory",
				ProviderType:   "java",
				// Dependency — auto-verified
			},
		},
	}

	// Step 1: Verify patterns
	results, err := verify.Run(extract, cacheDir)
	if err != nil {
		t.Fatalf("verify.Run: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}
	if results[0].Status != verify.StatusSkipped {
		t.Errorf("pattern 0 status = %q, want skipped", results[0].Status)
	}
	if results[1].Status != verify.StatusVerified {
		t.Errorf("pattern 1 status = %q, want verified", results[1].Status)
	}

	// Step 2: Construct rules
	constructResult, err := construct.Run(extract, rulesDir)
	if err != nil {
		t.Fatalf("construct.Run: %v", err)
	}

	// Step 3: Map verification results to rule IDs
	var verifiedIDs, notFoundIDs []string
	for _, r := range results {
		ruleID, ok := constructResult.PatternRuleMap[r.PatternIndex]
		if !ok {
			continue
		}
		switch r.Status {
		case verify.StatusVerified:
			verifiedIDs = append(verifiedIDs, ruleID)
		case verify.StatusNotFound:
			notFoundIDs = append(notFoundIDs, ruleID)
		}
	}

	// Step 4: Stamp verification labels
	if err := rules.StampVerificationResults(rulesDir, verifiedIDs, notFoundIDs); err != nil {
		t.Fatalf("StampVerificationResults: %v", err)
	}

	// Step 5: Read back and check labels
	allRules, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("ReadRulesDir: %v", err)
	}

	for _, r := range allRules {
		ruleID := r.RuleID
		for _, l := range r.Labels {
			if strings.HasPrefix(l, "konveyor.io/source-verified=") {
				// The dependency rule should be verified
				if ruleID == constructResult.PatternRuleMap[1] {
					if l != "konveyor.io/source-verified=true" {
						t.Errorf("dependency rule %s: label = %q, want source-verified=true", ruleID, l)
					}
				}
			}
		}
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -run TestEndToEnd -v`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./...`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/verify/e2e_test.go
git commit -m "test: add end-to-end test for verify → construct → stamp pipeline"
```

---

### Task 11: Add cache cleanup command

**Files:**
- Create: `internal/verify/cache.go`
- Create: `internal/verify/cache_test.go`

- [ ] **Step 1: Write failing test for cache cleanup**

Create `internal/verify/cache_test.go`:

```go
package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanCache(t *testing.T) {
	cacheDir := t.TempDir()

	// Create a fake cached artifact
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

	// Directory should be gone
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -run TestCleanCache -v`
Expected: FAIL — `CleanCache` doesn't exist

- [ ] **Step 3: Implement CleanCache**

Create `internal/verify/cache.go`:

```go
package verify

import (
	"fmt"
	"os"
	"path/filepath"
)

// CleanCache removes all cached artifacts (JARs, class listings) from the cache directory.
// Returns the number of files removed and total bytes freed.
func CleanCache(cacheDir string) (int, int64, error) {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var count int
	var totalBytes int64

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
			totalBytes += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("walking cache dir: %w", err)
	}

	if err := os.RemoveAll(cacheDir); err != nil {
		return 0, 0, fmt.Errorf("removing cache dir: %w", err)
	}

	return count, totalBytes, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./internal/verify/ -run TestCleanCache -v`
Expected: ALL PASS

- [ ] **Step 5: Add --clean flag to cmd/verify**

In `cmd/verify/main.go`, add a `--clean` flag:

```go
cleanFlag := flag.Bool("clean", false, "Remove cached artifacts and exit")
```

After flag parsing, before the main verify logic:

```go
if *cleanFlag {
	cache := *cacheDir
	if cache == "" {
		if *patternsPath != "" {
			cache = filepath.Join(filepath.Dir(*patternsPath), "verify-cache")
		} else {
			cli.Fail("invalid_arguments", "--cache or --patterns required with --clean", "verify", "set --cache to the cache directory to clean", nil)
		}
	}
	count, bytes, err := verify.CleanCache(cache)
	if err != nil {
		cli.Fail("clean_failed", err.Error(), "verify", "check cache directory permissions", map[string]string{"cache": cache})
	}
	cli.WriteJSON(map[string]interface{}{
		"status":        "ok",
		"files_removed": count,
		"bytes_freed":   bytes,
	})
	return
}
```

- [ ] **Step 6: Run full test suite**

Run: `cd /Users/sraghuna/local_dev/konveyor/ai-rule-gen-skill-first && go test ./... && go build ./cmd/verify/`
Expected: ALL PASS, builds successfully

- [ ] **Step 7: Commit**

```bash
git add internal/verify/cache.go internal/verify/cache_test.go cmd/verify/main.go
git commit -m "feat: add cache cleanup for verification artifacts"
```

---

### Task 12: Update development guidelines

**Files:**
- Modify: `docs/development-guidelines.md`

- [ ] **Step 1: Read the current guidelines**

Read `docs/development-guidelines.md` to understand existing structure.

- [ ] **Step 2: Add verification section**

Add a section documenting the `cmd/verify` command, the `internal/verify` package, the `source_artifact` field on `MigrationPattern`, and the `konveyor.io/source-verified` label. Follow the existing documentation style.

- [ ] **Step 3: Commit**

```bash
git add docs/development-guidelines.md
git commit -m "docs: add deterministic verification to development guidelines"
```
