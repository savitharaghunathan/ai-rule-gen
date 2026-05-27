package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelativeFromURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		appDir  string
		want    string
	}{
		{
			name:   "file URI with appDir",
			uri:    "file:///home/user/app/src/main/java/com/example/Foo.java",
			appDir: "/home/user/app",
			want:   "src/main/java/com/example/Foo.java",
		},
		{
			name:   "file URI without appDir",
			uri:    "file:///home/user/app/src/main/java/com/example/Foo.java",
			appDir: "",
			want:   "/home/user/app/src/main/java/com/example/Foo.java",
		},
		{
			name:   "plain path with appDir",
			uri:    "/home/user/app/src/com/example/Bar.java",
			appDir: "/home/user/app",
			want:   "src/com/example/Bar.java",
		},
		{
			name:   "path outside appDir falls back to full path",
			uri:    "file:///other/project/Foo.java",
			appDir: "/home/user/app",
			want:   "/other/project/Foo.java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeFromURI(tt.uri, tt.appDir)
			if got != tt.want {
				t.Errorf("relativeFromURI(%q, %q) = %q, want %q", tt.uri, tt.appDir, got, tt.want)
			}
		})
	}
}

func TestParseAnalyzeOutputPreservesPath(t *testing.T) {
	dir := t.TempDir()
	appDir := "/home/user/app"

	outputYAML := `- name: test-ruleset
  violations:
    rule-001:
      incidents:
        - uri: file:///home/user/app/src/main/java/com/example/Foo.java
        - uri: file:///home/user/app/src/main/java/com/other/Foo.java
`
	path := filepath.Join(dir, "output.yaml")
	os.WriteFile(path, []byte(outputYAML), 0o644)

	cov, err := parseAnalyzeOutput(path, appDir)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := cov.Violations["rule-001"]
	if !ok {
		t.Fatal("expected rule-001 violation")
	}
	if v.Incidents != 2 {
		t.Errorf("incidents: got %d, want 2", v.Incidents)
	}
	if len(v.Files) != 2 {
		t.Errorf("files: got %d, want 2 (same filename in different dirs should be distinct)", len(v.Files))
	}

	fileSet := make(map[string]bool, len(v.Files))
	for _, f := range v.Files {
		fileSet[f] = true
	}
	if !fileSet["src/main/java/com/example/Foo.java"] {
		t.Error("missing src/main/java/com/example/Foo.java")
	}
	if !fileSet["src/main/java/com/other/Foo.java"] {
		t.Error("missing src/main/java/com/other/Foo.java")
	}
}
