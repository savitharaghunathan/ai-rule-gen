package verify

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

const maxJARSize = 200 * 1024 * 1024 // 200 MB

var errArtifactNotFound = errors.New("artifact not found on Maven Central")

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
	if err := validateCoordinates(sa); err != nil {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusSkipped,
			Reason:    fmt.Sprintf("invalid source_artifact: %v", err),
		}, nil
	}

	classLines, err := v.getClassList(sa.GroupID, sa.ArtifactID, sa.Version)
	if err != nil {
		if errors.Is(err, errArtifactNotFound) {
			return Result{
				SourceFQN: pattern.SourceFQN,
				Status:    StatusNotFound,
				Reason:    fmt.Sprintf("artifact %s:%s:%s not found on Maven Central", sa.GroupID, sa.ArtifactID, sa.Version),
			}, nil
		}
		if isNetworkError(err) {
			return Result{
				SourceFQN: pattern.SourceFQN,
				Status:    StatusOffline,
				Reason:    fmt.Sprintf("Maven Central unreachable: %v", err),
			}, nil
		}
		return Result{}, err
	}

	jarName := fmt.Sprintf("%s-%s.jar", sa.ArtifactID, sa.Version)
	return v.verifyAgainstClassList(pattern, classLines, jarName), nil
}

func (v *JavaVerifier) verifyAgainstClassList(pattern rules.MigrationPattern, classLines []string, jarName string) Result {
	switch strings.ToUpper(pattern.LocationType) {
	case "METHOD_CALL":
		classFQN := stripMethodName(pattern.SourceFQN)
		if classFQN != pattern.SourceFQN {
			for _, target := range fqnToClassPaths(classFQN) {
				if findInClassList(classLines, target) {
					return Result{
						SourceFQN: pattern.SourceFQN,
						Status:    StatusVerified,
						Evidence:  fmt.Sprintf("class %s found in %s", classFQN, jarName),
					}
				}
			}
		}
	case "PACKAGE":
		base := strings.TrimSuffix(pattern.SourceFQN, "*")
		base = strings.TrimSuffix(base, ".")
		prefix := strings.ReplaceAll(base, ".", "/") + "/"
		for _, line := range classLines {
			if strings.HasPrefix(line, prefix) {
				return Result{
					SourceFQN: pattern.SourceFQN,
					Status:    StatusVerified,
					Evidence:  fmt.Sprintf("package found in %s", jarName),
				}
			}
		}
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusNotFound,
			Reason:    fmt.Sprintf("no classes under package in %s", jarName),
		}
	}

	for _, target := range fqnToClassPaths(pattern.SourceFQN) {
		if findInClassList(classLines, target) {
			return Result{
				SourceFQN: pattern.SourceFQN,
				Status:    StatusVerified,
				Evidence:  fmt.Sprintf("found in %s", jarName),
			}
		}
	}

	className := classNameFromFQN(pattern.SourceFQN)
	suggestions := findSuggestions(classLines, className)
	return Result{
		SourceFQN:   pattern.SourceFQN,
		Status:      StatusNotFound,
		Reason:      fmt.Sprintf("not found in %s", jarName),
		Suggestions: suggestions,
	}
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
	jarURL := fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.jar",
		groupPath, artifactID, version, artifactID, version)

	resp, err := v.httpClient.Get(jarURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", jarURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("downloading %s: %w", jarURL, errArtifactNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Maven Central returned %d for %s", resp.StatusCode, jarURL)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", destPath, err)
	}

	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxJARSize)); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", destPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", destPath, err)
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

// stripMethodName removes a trailing method name from a FQN.
// "org.apache.http.HttpResponse.getStatusLine" → "org.apache.http.HttpResponse"
// Returns the original FQN if no method component is detected.
// stripMethodName removes a trailing method name from a FQN.
// A method component is a lowercase-starting part preceded by an
// uppercase-starting part (the class name).
// "org.apache.http.HttpResponse.getStatusLine" → "org.apache.http.HttpResponse"
// "org.apache.http" → "org.apache.http" (no class before "http")
func stripMethodName(fqn string) string {
	parts := strings.Split(fqn, ".")
	if len(parts) < 3 {
		return fqn
	}
	last := parts[len(parts)-1]
	prev := parts[len(parts)-2]
	if len(last) > 0 && last[0] >= 'a' && last[0] <= 'z' &&
		len(prev) > 0 && prev[0] >= 'A' && prev[0] <= 'Z' {
		return strings.Join(parts[:len(parts)-1], ".")
	}
	return fqn
}

func fqnToClassPaths(fqn string) []string {
	primary := strings.ReplaceAll(fqn, ".", "/") + ".class"
	paths := []string{primary}
	parts := strings.Split(fqn, ".")
	for i := len(parts) - 2; i >= 1; i-- {
		if len(parts[i]) > 0 && parts[i][0] >= 'A' && parts[i][0] <= 'Z' {
			pkg := strings.Join(parts[:i], "/")
			cls := strings.Join(parts[i:], "$")
			paths = append(paths, pkg+"/"+cls+".class")
		}
	}
	return paths
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
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		if errors.As(urlErr.Err, &netErr) {
			return true
		}
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return false
}

func validateCoordinates(ac *rules.ArtifactCoordinates) error {
	for _, pair := range []struct{ name, val string }{
		{"groupId", ac.GroupID},
		{"artifactId", ac.ArtifactID},
		{"version", ac.Version},
	} {
		if pair.val == "" {
			return fmt.Errorf("%s is empty", pair.name)
		}
		if strings.Contains(pair.val, "..") || strings.Contains(pair.val, "/") || strings.Contains(pair.val, "\\") {
			return fmt.Errorf("%s contains invalid characters: %q", pair.name, pair.val)
		}
	}
	return nil
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
