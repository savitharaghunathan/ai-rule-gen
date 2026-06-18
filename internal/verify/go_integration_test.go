//go:build integration

package verify

import (
	"context"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestGoVerifier_RealPackage(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewGoVerifier(cacheDir)

	tests := []struct {
		name    string
		pattern rules.MigrationPattern
		want    Status
	}{
		{
			name: "known package golang.org/x/crypto/md4",
			pattern: rules.MigrationPattern{
				SourceFQN:    "golang.org/x/crypto/md4",
				ProviderType: "go",
			},
			want: StatusVerified,
		},
		{
			name: "known package golang.org/x/crypto/blake2b",
			pattern: rules.MigrationPattern{
				SourceFQN:    "golang.org/x/crypto/blake2b",
				ProviderType: "go",
			},
			want: StatusVerified,
		},
		{
			name: "module root golang.org/x/crypto",
			pattern: rules.MigrationPattern{
				SourceFQN:    "golang.org/x/crypto",
				ProviderType: "go",
			},
			want: StatusVerified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := v.Verify(context.Background(), tt.pattern)
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

func TestGoVerifier_FakePackage(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewGoVerifier(cacheDir)

	result, err := v.Verify(context.Background(), rules.MigrationPattern{
		SourceFQN:    "golang.org/x/crypto/nonexistent",
		ProviderType: "go",
	})
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if result.Status != StatusNotFound {
		t.Errorf("status = %q, want not_found (evidence: %s, reason: %s)",
			result.Status, result.Evidence, result.Reason)
	}
}

func TestGoVerifier_FakeModule(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewGoVerifier(cacheDir)

	result, err := v.Verify(context.Background(), rules.MigrationPattern{
		SourceFQN:    "golang.org/x/nonexistentmodule/pkg",
		ProviderType: "go",
	})
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if result.Status != StatusNotFound {
		t.Errorf("status = %q, want not_found (evidence: %s, reason: %s)",
			result.Status, result.Evidence, result.Reason)
	}
}

func TestGoVerifier_CacheReuse(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewGoVerifier(cacheDir)

	pattern := rules.MigrationPattern{
		SourceFQN:    "golang.org/x/crypto/md4",
		ProviderType: "go",
	}

	r1, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if r1.Status != StatusVerified {
		t.Fatalf("first verify status = %q, want verified", r1.Status)
	}

	r2, err := v.Verify(pattern)
	if err != nil {
		t.Fatalf("second verify: %v", err)
	}
	if r2.Status != StatusVerified {
		t.Fatalf("second verify status = %q, want verified", r2.Status)
	}
}
