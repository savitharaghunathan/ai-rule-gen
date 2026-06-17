package verify

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestIsStdlib(t *testing.T) {
	tests := []struct {
		importPath string
		want       bool
	}{
		{"crypto/sha256", true},
		{"crypto/aes", true},
		{"fmt", true},
		{"net/http", true},
		{"golang.org/x/crypto/md4", false},
		{"github.com/user/repo", false},
		{"example.com/pkg", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isStdlib(tt.importPath)
		if got != tt.want {
			t.Errorf("isStdlib(%q) = %v, want %v", tt.importPath, got, tt.want)
		}
	}
}

func TestEscapeModulePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"golang.org/x/crypto", "golang.org/x/crypto"},
		{"github.com/Azure/go-sdk", "github.com/!azure/go-sdk"},
		{"github.com/BurntSushi/toml", "github.com/!burnt!sushi/toml"},
	}
	for _, tt := range tests {
		got, err := escapeModulePath(tt.input)
		if err != nil {
			t.Fatalf("escapeModulePath(%q) error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("escapeModulePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeFSPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"golang.org/x/crypto", "golang.org!x!crypto"},
		{"github.com/user/repo", "github.com!user!repo"},
	}
	for _, tt := range tests {
		got := escapeFSPath(tt.input)
		if got != tt.want {
			t.Errorf("escapeFSPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGoVerifier_StdlibAutoVerified(t *testing.T) {
	v := NewGoVerifier(t.TempDir())
	pattern := rules.MigrationPattern{
		SourceFQN:    "crypto/sha256",
		ProviderType: "go",
	}
	result, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if result.Status != StatusVerified {
		t.Errorf("status = %q, want verified; reason: %s", result.Status, result.Reason)
	}
	if result.Evidence != "Go standard library package" {
		t.Errorf("evidence = %q, want 'Go standard library package'", result.Evidence)
	}
}

func TestGoVerifier_DependencyAutoVerified(t *testing.T) {
	v := NewGoVerifier(t.TempDir())
	pattern := rules.MigrationPattern{
		DependencyName: "golang.org/x/crypto",
	}
	result, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if result.Status != StatusVerified {
		t.Errorf("status = %q, want verified", result.Status)
	}
}

func TestGoVerifier_BuiltinSkipped(t *testing.T) {
	v := NewGoVerifier(t.TempDir())
	tests := []struct {
		name      string
		pattern   rules.MigrationPattern
	}{
		{
			name: "regex pattern",
			pattern: rules.MigrationPattern{
				SourceFQN:    `GOEXPERIMENT\s*=\s*boringcrypto`,
				ProviderType: "builtin",
				FilePattern:  `.*\.Dockerfile`,
			},
		},
		{
			name: "InsecureSkipVerify regex",
			pattern: rules.MigrationPattern{
				SourceFQN:    `InsecureSkipVerify\s*:\s*true`,
				ProviderType: "builtin",
				FilePattern:  `.*\.go`,
			},
		},
		{
			name: "go.mod version regex",
			pattern: rules.MigrationPattern{
				SourceFQN:    `^go 1\\.([0-9]|1[0-9]|2[0-3])(\.[0-9]+)?$`,
				ProviderType: "builtin",
				FilePattern:  `go\.mod`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.Verify(tt.pattern)
			if err != nil {
				t.Fatalf("Verify() error: %v", err)
			}
			if result.Status != StatusSkipped {
				t.Errorf("status = %q, want skipped", result.Status)
			}
			if result.Reason != "builtin.filecontent patterns use regex, not Go packages" {
				t.Errorf("reason = %q, want builtin skip reason", result.Reason)
			}
		})
	}
}

func TestGoVerifier_NoSourceFQN(t *testing.T) {
	v := NewGoVerifier(t.TempDir())
	pattern := rules.MigrationPattern{
		ProviderType: "go",
	}
	result, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if result.Status != StatusSkipped {
		t.Errorf("status = %q, want skipped", result.Status)
	}
}
