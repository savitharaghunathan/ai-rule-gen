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
