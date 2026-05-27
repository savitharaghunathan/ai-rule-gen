package groundtruth

import "testing"

func TestMapActionType(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"class_removed", "package_change"},
		{"method_removed", "method_removal"},
		{"method_changed", "signature_change"},
		{"field_removed", "config_change"},
		{"unknown", "behavioral_change"},
	}
	for _, tt := range tests {
		if got := mapActionType(tt.kind); got != tt.want {
			t.Errorf("mapActionType(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"class_removed", "high"},
		{"method_removed", "high"},
		{"method_changed", "high"},
		{"field_removed", "medium"},
		{"unknown", "low"},
	}
	for _, tt := range tests {
		if got := mapSeverity(tt.kind); got != tt.want {
			t.Errorf("mapSeverity(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestConvertChanges(t *testing.T) {
	changes := []APIChange{
		{OldAPI: "org.apache.http.HttpEntity", ChangeKind: "class_removed"},
		{OldAPI: "org.apache.http.HttpClient.execute", ChangeKind: "method_removed"},
	}

	entries := ConvertChanges(changes)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].ActionType != "package_change" {
		t.Errorf("entry 0 action: got %q, want package_change", entries[0].ActionType)
	}
	if entries[0].Severity != "high" {
		t.Errorf("entry 0 severity: got %q, want high", entries[0].Severity)
	}
	if entries[0].ReviewedBy != "japicmp" {
		t.Errorf("entry 0 reviewed_by: got %q, want japicmp", entries[0].ReviewedBy)
	}
	if entries[0].NewAPI != "" {
		t.Errorf("entry 0 new_api should be empty, got %q", entries[0].NewAPI)
	}

	if entries[1].ActionType != "method_removal" {
		t.Errorf("entry 1 action: got %q, want method_removal", entries[1].ActionType)
	}
}
