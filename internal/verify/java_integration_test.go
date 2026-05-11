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
					Version:    "4.5.14",
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
					Version:    "4.5.14",
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
			Version:    "4.5.14",
		},
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

func TestJavaVerifier_Suggestions(t *testing.T) {
	cacheDir := t.TempDir()
	v := NewJavaVerifier(cacheDir)

	result, err := v.Verify(rules.MigrationPattern{
		SourceFQN:    "org.apache.http.client.FakeNonExistentClass",
		ProviderType: "java",
		SourceArtifact: &rules.ArtifactCoordinates{
			GroupID:    "org.apache.httpcomponents",
			ArtifactID: "httpclient",
			Version:    "4.5.14",
		},
	})
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected no suggestions for fake class, got %v", result.Suggestions)
	}

	result2, err := v.Verify(rules.MigrationPattern{
		SourceFQN:    "com.wrong.package.HttpClient",
		ProviderType: "java",
		SourceArtifact: &rules.ArtifactCoordinates{
			GroupID:    "org.apache.httpcomponents",
			ArtifactID: "httpclient",
			Version:    "4.5.14",
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
