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

type JavaVerifier struct {
	cacheDir   string
	httpClient *http.Client
}

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

func listJARClasses(jarPath string) ([]string, error) {
	cmd := exec.Command("jar", "tf", jarPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("jar tf %s: %w", jarPath, err)
	}
	return splitLines(string(out)), nil
}

func fqnToClassPath(fqn string) string {
	return strings.ReplaceAll(fqn, ".", "/") + ".class"
}

func findInClassList(classLines []string, classPath string) bool {
	for _, line := range classLines {
		if line == classPath {
			return true
		}
	}
	return false
}

func classNameFromFQN(fqn string) string {
	parts := strings.Split(fqn, ".")
	return parts[len(parts)-1]
}

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
