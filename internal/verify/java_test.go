package verify

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestFQNToClassPaths(t *testing.T) {
	tests := []struct {
		fqn       string
		wantFirst string
		wantLen   int
	}{
		{"org.apache.http.client.HttpClient", "org/apache/http/client/HttpClient.class", 1},
		{"com.example.MyClass", "com/example/MyClass.class", 1},
		{"Single", "Single.class", 1},
		{"com.example.Outer.Inner", "com/example/Outer/Inner.class", 2},
	}
	for _, tt := range tests {
		got := fqnToClassPaths(tt.fqn)
		if len(got) < 1 {
			t.Fatalf("fqnToClassPaths(%q) returned empty", tt.fqn)
		}
		if got[0] != tt.wantFirst {
			t.Errorf("fqnToClassPaths(%q)[0] = %q, want %q", tt.fqn, got[0], tt.wantFirst)
		}
		if len(got) != tt.wantLen {
			t.Errorf("fqnToClassPaths(%q) len = %d, want %d; paths: %v", tt.fqn, len(got), tt.wantLen, got)
		}
	}
}

func TestFQNToClassPaths_InnerClass(t *testing.T) {
	paths := fqnToClassPaths("com.example.Outer.Inner")
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[1] != "com/example/Outer$Inner.class" {
		t.Errorf("inner class path = %q, want %q", paths[1], "com/example/Outer$Inner.class")
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
		classPath string
		wantFound bool
	}{
		{"org/apache/http/client/HttpClient.class", true},
		{"org/apache/http/client/methods/HttpGet.class", true},
		{"org/apache/http/client/NonExistent.class", false},
	}
	for _, tt := range tests {
		found := findInClassList(classLines, tt.classPath)
		if found != tt.wantFound {
			t.Errorf("findInClassList(%q) = %v, want %v", tt.classPath, found, tt.wantFound)
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

func TestStripMethodName(t *testing.T) {
	tests := []struct {
		fqn  string
		want string
	}{
		{"org.apache.http.HttpResponse.getStatusLine", "org.apache.http.HttpResponse"},
		{"org.apache.http.impl.client.HttpClientBuilder.setRetryHandler", "org.apache.http.impl.client.HttpClientBuilder"},
		{"org.apache.http.HttpResponse", "org.apache.http.HttpResponse"},
		{"org.apache.http", "org.apache.http"},
		{"Single", "Single"},
		{"com.example.Outer.Inner", "com.example.Outer.Inner"}, // Inner starts uppercase — not a method
		{"javax.naming.Context.lookup", "javax.naming.Context"},
	}
	for _, tt := range tests {
		got := stripMethodName(tt.fqn)
		if got != tt.want {
			t.Errorf("stripMethodName(%q) = %q, want %q", tt.fqn, got, tt.want)
		}
	}
}

func TestVerifyMethodCall(t *testing.T) {
	classLines := []string{
		"org/apache/http/HttpResponse.class",
		"org/apache/http/client/HttpClient.class",
	}

	v := &JavaVerifier{}
	// METHOD_CALL with ClassName.methodName should verify against the class
	pattern := rules.MigrationPattern{
		SourceFQN:    "org.apache.http.HttpResponse.getStatusLine",
		LocationType: "METHOD_CALL",
	}
	result := v.verifyAgainstClassList(pattern, classLines, "test.jar")
	if result.Status != StatusVerified {
		t.Errorf("METHOD_CALL verify: got %q, want verified; reason: %s", result.Status, result.Reason)
	}
	if result.Evidence == "" {
		t.Error("expected evidence to be set")
	}
}

func TestVerifyMethodCallNoMethodComponent(t *testing.T) {
	classLines := []string{
		"org/apache/http/HttpResponse.class",
	}

	v := &JavaVerifier{}
	// METHOD_CALL where FQN has no lowercase trailing component
	pattern := rules.MigrationPattern{
		SourceFQN:    "org.apache.http.HttpResponse",
		LocationType: "METHOD_CALL",
	}
	result := v.verifyAgainstClassList(pattern, classLines, "test.jar")
	if result.Status != StatusVerified {
		t.Errorf("METHOD_CALL fallback: got %q, want verified", result.Status)
	}
}

func TestVerifyPackage(t *testing.T) {
	classLines := []string{
		"org/apache/http/HttpResponse.class",
		"org/apache/http/client/HttpClient.class",
		"org/apache/hc/core5/http/HttpHost.class",
	}

	v := &JavaVerifier{}
	pattern := rules.MigrationPattern{
		SourceFQN:    "org.apache.http",
		LocationType: "PACKAGE",
	}
	result := v.verifyAgainstClassList(pattern, classLines, "test.jar")
	if result.Status != StatusVerified {
		t.Errorf("PACKAGE verify: got %q, want verified", result.Status)
	}
}

func TestVerifyPackageWildcard(t *testing.T) {
	classLines := []string{
		"com/fasterxml/jackson/databind/ObjectMapper.class",
		"com/fasterxml/jackson/core/JsonParser.class",
	}

	v := &JavaVerifier{}
	pattern := rules.MigrationPattern{
		SourceFQN:    "com.fasterxml.jackson*",
		LocationType: "PACKAGE",
	}
	result := v.verifyAgainstClassList(pattern, classLines, "test.jar")
	if result.Status != StatusVerified {
		t.Errorf("PACKAGE wildcard verify: got %q, want verified", result.Status)
	}
}

func TestVerifyPackageWildcardNoMatch(t *testing.T) {
	classLines := []string{
		"org/apache/hc/core5/http/HttpHost.class",
	}

	v := &JavaVerifier{}
	pattern := rules.MigrationPattern{
		SourceFQN:    "com.fasterxml.jackson*",
		LocationType: "PACKAGE",
	}
	result := v.verifyAgainstClassList(pattern, classLines, "test.jar")
	if result.Status != StatusNotFound {
		t.Errorf("PACKAGE wildcard not found: got %q, want not_found", result.Status)
	}
}

func TestVerifyPackageNotFound(t *testing.T) {
	classLines := []string{
		"org/apache/hc/core5/http/HttpHost.class",
	}

	v := &JavaVerifier{}
	pattern := rules.MigrationPattern{
		SourceFQN:    "org.apache.http",
		LocationType: "PACKAGE",
	}
	result := v.verifyAgainstClassList(pattern, classLines, "test.jar")
	if result.Status != StatusNotFound {
		t.Errorf("PACKAGE not found: got %q, want not_found", result.Status)
	}
}

func TestValidateCoordinates(t *testing.T) {
	tests := []struct {
		name    string
		ac      *rules.ArtifactCoordinates
		wantErr bool
	}{
		{"valid", &rules.ArtifactCoordinates{GroupID: "org.apache", ArtifactID: "httpclient", Version: "4.5.14"}, false},
		{"empty groupId", &rules.ArtifactCoordinates{GroupID: "", ArtifactID: "httpclient", Version: "4.5.14"}, true},
		{"path traversal", &rules.ArtifactCoordinates{GroupID: "../../etc", ArtifactID: "passwd", Version: "1"}, true},
		{"slash in artifactId", &rules.ArtifactCoordinates{GroupID: "org.apache", ArtifactID: "http/client", Version: "1"}, true},
		{"backslash", &rules.ArtifactCoordinates{GroupID: "org.apache", ArtifactID: "client", Version: "1\\2"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCoordinates(tt.ac)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCoordinates() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
