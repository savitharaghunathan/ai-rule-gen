package groundtruth

import "testing"

func TestMergePreservesHumanReviewed(t *testing.T) {
	existing := &GroundTruth{
		SchemaVersion: 1,
		GuideURL:      "https://example.com/guide",
		GuideVersion:  "2026-05-27",
		Entries: []Entry{
			{OldAPI: "org.apache.http.HttpGet", NewAPI: "org.apache.hc.client5.http.classic.methods.HttpGet", ActionType: "package_change", ReviewedBy: "alice", ReviewedDate: "2026-05-20"},
			{OldAPI: "org.apache.http.HttpEntity", ActionType: "package_change", ReviewedBy: "japicmp", ReviewedDate: "2026-05-15"},
			{OldAPI: "org.apache.http.HttpPost", ActionType: "package_change", ReviewedBy: "", ReviewedDate: ""},
		},
	}

	newEntries := []Entry{
		{OldAPI: "org.apache.http.HttpGet", ActionType: "package_change", ReviewedBy: "japicmp", ReviewedDate: "2026-05-27"},
		{OldAPI: "org.apache.http.HttpEntity", ActionType: "package_change", Severity: "high", ReviewedBy: "japicmp", ReviewedDate: "2026-05-27"},
		{OldAPI: "org.apache.http.StatusLine", ActionType: "method_removal", ReviewedBy: "japicmp", ReviewedDate: "2026-05-27"},
	}

	result := Merge(existing, newEntries)

	if len(result.Entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(result.Entries))
	}

	// Human-reviewed entry preserved (alice reviewed HttpGet)
	httpGet := findEntry(result.Entries, "org.apache.http.HttpGet")
	if httpGet == nil {
		t.Fatal("missing HttpGet entry")
	}
	if httpGet.ReviewedBy != "alice" {
		t.Errorf("HttpGet should keep human review, got reviewed_by=%q", httpGet.ReviewedBy)
	}
	if httpGet.NewAPI != "org.apache.hc.client5.http.classic.methods.HttpGet" {
		t.Errorf("HttpGet should keep human new_api, got %q", httpGet.NewAPI)
	}

	// japicmp-only entry updated
	httpEntity := findEntry(result.Entries, "org.apache.http.HttpEntity")
	if httpEntity == nil {
		t.Fatal("missing HttpEntity entry")
	}
	if httpEntity.Severity != "high" {
		t.Errorf("HttpEntity should be updated, got severity=%q", httpEntity.Severity)
	}

	// New entry added
	statusLine := findEntry(result.Entries, "org.apache.http.StatusLine")
	if statusLine == nil {
		t.Fatal("missing StatusLine entry (should be added)")
	}

	// Unmatched existing entry preserved
	httpPost := findEntry(result.Entries, "org.apache.http.HttpPost")
	if httpPost == nil {
		t.Fatal("missing HttpPost entry (existing unmatched should be kept)")
	}
}

func TestMergePreservesMetadata(t *testing.T) {
	existing := &GroundTruth{
		SchemaVersion: 1,
		GuideURL:      "https://example.com",
		GuideVersion:  "2026-01-01",
	}

	result := Merge(existing, nil)
	if result.SchemaVersion != 1 {
		t.Errorf("schema_version: got %d, want 1", result.SchemaVersion)
	}
	if result.GuideURL != "https://example.com" {
		t.Errorf("guide_url: got %q", result.GuideURL)
	}
}

func findEntry(entries []Entry, oldAPI string) *Entry {
	for _, e := range entries {
		if e.OldAPI == oldAPI {
			return &e
		}
	}
	return nil
}
