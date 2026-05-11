package verify

import (
	"testing"
)

func TestFQNToClassPath(t *testing.T) {
	tests := []struct {
		fqn  string
		want string
	}{
		{"org.apache.http.client.HttpClient", "org/apache/http/client/HttpClient.class"},
		{"com.example.MyClass", "com/example/MyClass.class"},
		{"Single", "Single.class"},
	}
	for _, tt := range tests {
		got := fqnToClassPath(tt.fqn)
		if got != tt.want {
			t.Errorf("fqnToClassPath(%q) = %q, want %q", tt.fqn, got, tt.want)
		}
	}
}

func TestFindInClassList(t *testing.T) {
	classLines := []string{
		"META-INF/MANIFEST.MF",
		"org/apache/http/client/HttpClient.class",
		"org/apache/http/client/methods/HttpGet.class",
		"org/apache/http/impl/client/CloseableHttpClient.class",
	}

	tests := []struct {
		fqn       string
		wantFound bool
	}{
		{"org.apache.http.client.HttpClient", true},
		{"org.apache.http.client.methods.HttpGet", true},
		{"org.apache.http.client.NonExistent", false},
	}
	for _, tt := range tests {
		found := findInClassList(classLines, fqnToClassPath(tt.fqn))
		if found != tt.wantFound {
			t.Errorf("findInClassList(%q) = %v, want %v", tt.fqn, found, tt.wantFound)
		}
	}
}

func TestFindSuggestions(t *testing.T) {
	classLines := []string{
		"org/apache/http/client/HttpClient.class",
		"org/apache/hc/client5/http/classic/HttpClient.class",
		"org/apache/http/impl/client/CloseableHttpClient.class",
	}

	suggestions := findSuggestions(classLines, "HttpClient")
	if len(suggestions) < 1 {
		t.Fatal("expected at least one suggestion")
	}

	found := false
	for _, s := range suggestions {
		if s == "org.apache.http.client.HttpClient" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected org.apache.http.client.HttpClient in suggestions, got %v", suggestions)
	}
}
