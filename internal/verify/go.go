package verify

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

const maxModuleZipSize = 100 * 1024 * 1024 // 100 MB

var errModuleNotFound = errors.New("module not found on Go module proxy")

type GoVerifier struct {
	cacheDir   string
	httpClient *http.Client
	moduleCache map[string]resolvedModule
}

type resolvedModule struct {
	modulePath string
	version    string
}

func NewGoVerifier(cacheDir string) *GoVerifier {
	return &GoVerifier{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		moduleCache: make(map[string]resolvedModule),
	}
}

func (v *GoVerifier) Language() string { return "go" }

func (v *GoVerifier) Verify(ctx context.Context, pattern rules.MigrationPattern) (Result, error) {
	if pattern.ProviderType == "builtin" {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusSkipped,
			Reason:    "builtin.filecontent patterns use regex, not Go packages",
		}, nil
	}

	if pattern.DependencyName != "" {
		return Result{
			SourceFQN: pattern.DependencyName,
			Status:    StatusVerified,
			Evidence:  "dependency patterns verified by go.mod pre-check",
		}, nil
	}

	if pattern.SourceFQN == "" {
		return Result{
			Status: StatusSkipped,
			Reason: "no source_fqn to verify",
		}, nil
	}

	if isStdlib(pattern.SourceFQN) {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusVerified,
			Evidence:  "Go standard library package",
		}, nil
	}

	modulePath, version, err := v.resolveModule(ctx, pattern.SourceFQN)
	if err != nil {
		if errors.Is(err, errModuleNotFound) {
			return Result{
				SourceFQN: pattern.SourceFQN,
				Status:    StatusNotFound,
				Reason:    fmt.Sprintf("module for %s not found on proxy.golang.org", pattern.SourceFQN),
			}, nil
		}
		if isNetworkError(err) {
			return Result{
				SourceFQN: pattern.SourceFQN,
				Status:    StatusOffline,
				Reason:    fmt.Sprintf("Go module proxy unreachable: %v", err),
			}, nil
		}
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusOffline,
			Reason:    fmt.Sprintf("resolver error: %v", err),
		}, nil
	}

	packages, err := v.getPackageList(ctx, modulePath, version)
	if err != nil {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusOffline,
			Reason:    fmt.Sprintf("package list error: %v", err),
		}, nil
	}

	pkgSubdir := strings.TrimPrefix(pattern.SourceFQN, modulePath)
	pkgSubdir = strings.TrimPrefix(pkgSubdir, "/")

	if pkgSubdir == "" {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusVerified,
			Evidence:  fmt.Sprintf("module root %s@%s exists on proxy.golang.org", modulePath, version),
		}, nil
	}

	if slices.Contains(packages, pkgSubdir) {
		return Result{
			SourceFQN: pattern.SourceFQN,
			Status:    StatusVerified,
			Evidence:  fmt.Sprintf("package %s found in %s@%s", pkgSubdir, modulePath, version),
		}, nil
	}

	var suggestions []string
	targetName := filepath.Base(pkgSubdir)
	for _, pkg := range packages {
		if filepath.Base(pkg) == targetName {
			suggestions = append(suggestions, modulePath+"/"+pkg)
		}
	}

	return Result{
		SourceFQN:   pattern.SourceFQN,
		Status:      StatusNotFound,
		Reason:      fmt.Sprintf("package %s not found in %s@%s", pkgSubdir, modulePath, version),
		Suggestions: suggestions,
	}, nil
}

func (v *GoVerifier) resolveModule(ctx context.Context, importPath string) (string, string, error) {
	if cached, ok := v.moduleCache[importPath]; ok {
		return cached.modulePath, cached.version, nil
	}

	parts := strings.Split(importPath, "/")

	// Try progressively shorter paths to find the module root.
	// Minimum is the first 2 components for domain-based paths (e.g., "golang.org/x").
	minParts := min(2, len(parts))

	for i := len(parts); i >= minParts; i-- {
		candidate := strings.Join(parts[:i], "/")

		version, err := v.getLatestVersion(ctx, candidate)
		if err != nil {
			if errors.Is(err, errModuleNotFound) {
				continue
			}
			return "", "", err
		}

		v.moduleCache[importPath] = resolvedModule{modulePath: candidate, version: version}
		return candidate, version, nil
	}

	return "", "", fmt.Errorf("resolving module for %s: %w", importPath, errModuleNotFound)
}

func (v *GoVerifier) getLatestVersion(ctx context.Context, modulePath string) (string, error) {
	escaped, err := escapeModulePath(modulePath)
	if err != nil {
		return "", err
	}

	latestURL := fmt.Sprintf("https://proxy.golang.org/%s/@latest", escaped)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %s: %w", latestURL, err)
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", latestURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return "", errModuleNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy.golang.org returned %d for %s", resp.StatusCode, latestURL)
	}

	var info struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("parsing version info: %w", err)
	}
	return info.Version, nil
}

func (v *GoVerifier) getPackageList(ctx context.Context, modulePath, version string) ([]string, error) {
	escapedModule := escapeFSPath(modulePath)
	cacheDir := filepath.Join(v.cacheDir, escapedModule, version)
	packagesFile := filepath.Join(cacheDir, "packages.txt")

	if data, err := os.ReadFile(packagesFile); err == nil {
		return splitLines(string(data)), nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}

	zipPath := filepath.Join(cacheDir, "module.zip")
	if err := v.downloadModuleZip(ctx, modulePath, version, zipPath); err != nil {
		return nil, err
	}

	packages, err := listPackagesInZip(zipPath, modulePath, version)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(packagesFile, []byte(strings.Join(packages, "\n")), 0o644); err != nil {
		return nil, fmt.Errorf("writing packages cache: %w", err)
	}

	return packages, nil
}

func (v *GoVerifier) downloadModuleZip(ctx context.Context, modulePath, version, destPath string) error {
	escaped, err := escapeModulePath(modulePath)
	if err != nil {
		return err
	}

	zipURL := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.zip", escaped, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", zipURL, err)
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", zipURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return fmt.Errorf("downloading %s: %w", zipURL, errModuleNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy.golang.org returned %d for %s", resp.StatusCode, zipURL)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", destPath, err)
	}

	n, err := io.Copy(f, io.LimitReader(resp.Body, maxModuleZipSize+1))
	if err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", destPath, err)
	}
	if n > maxModuleZipSize {
		f.Close()
		os.Remove(destPath)
		return fmt.Errorf("module zip %s exceeds %d bytes", zipURL, maxModuleZipSize)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", destPath, err)
	}
	return nil
}

func listPackagesInZip(zipPath, modulePath, version string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("opening zip %s: %w", zipPath, err)
	}
	defer r.Close()

	// Module zip entries are prefixed with "module@version/"
	prefix := modulePath + "@" + version + "/"

	pkgSet := make(map[string]bool)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(name, prefix)
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		// Skip vendored code
		if strings.Contains(rel, "/vendor/") || strings.HasPrefix(rel, "vendor/") {
			continue
		}
		dir := filepath.Dir(rel)
		if dir == "." {
			dir = ""
		}
		pkgSet[dir] = true
	}

	packages := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		packages = append(packages, pkg)
	}
	return packages, nil
}

// isStdlib returns true if the import path is a Go standard library package.
// Standard library paths don't contain a dot in the first path component.
func isStdlib(importPath string) bool {
	if importPath == "" {
		return false
	}
	firstComponent, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(firstComponent, ".")
}

// escapeModulePath applies Go module case-encoding for proxy URLs.
// Uppercase letters are replaced with "!" followed by the lowercase letter.
func escapeModulePath(path string) (string, error) {
	var b strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}

// escapeFSPath converts a module path into a filesystem-safe directory name.
// Slashes become "!" and uppercase letters become "!!" followed by lowercase.
func escapeFSPath(path string) string {
	var b strings.Builder
	for _, r := range path {
		switch {
		case r == '/':
			b.WriteByte('!')
		case r >= 'A' && r <= 'Z':
			b.WriteString("!!")
			b.WriteRune(r + ('a' - 'A'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

