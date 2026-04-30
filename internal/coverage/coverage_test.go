package coverage

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestParseSections(t *testing.T) {
	guide := `# Title

Some intro text.

## Section One

Content of section one.

### Sub Section

Content of sub section.

## Section Two

Content of section two.
`
	sections := ParseSections(guide)
	if len(sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(sections))
	}

	tests := []struct {
		heading string
		level   int
	}{
		{"Title", 1},
		{"Section One", 2},
		{"Sub Section", 3},
		{"Section Two", 2},
	}
	for i, tt := range tests {
		if sections[i].Heading != tt.heading {
			t.Errorf("section %d: heading = %q, want %q", i, sections[i].Heading, tt.heading)
		}
		if sections[i].Level != tt.level {
			t.Errorf("section %d: level = %d, want %d", i, sections[i].Level, tt.level)
		}
	}
}

func TestParseSectionsContent(t *testing.T) {
	guide := `## First

Line one.
Line two.

## Second

Line three.
`
	sections := ParseSections(guide)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if !contains(sections[0].Content, "Line one") {
		t.Errorf("first section should contain 'Line one', got %q", sections[0].Content)
	}
	if !contains(sections[1].Content, "Line three") {
		t.Errorf("second section should contain 'Line three', got %q", sections[1].Content)
	}
}

func TestJavaScannerInlineCode(t *testing.T) {
	content := "Use `org.springframework.boot.autoconfigure.http.HttpMessageConverters` for converters. " +
		"Set `management.endpoint.health.probes.enabled` to `true`."

	scanner := NewScanner("java")
	artifacts := scanner.Scan(content)

	want := map[string]bool{
		"org.springframework.boot.autoconfigure.http.HttpMessageConverters": true,
		"management.endpoint.health.probes.enabled":                        true,
	}
	for _, a := range artifacts {
		delete(want, a)
	}
	for missing := range want {
		t.Errorf("missing artifact: %s", missing)
	}
}

func TestJavaScannerCodeBlocks(t *testing.T) {
	content := "Add this dependency:\n\n" +
		"```\n" +
		"<dependency>\n" +
		"    <groupId>org.springframework.boot</groupId>\n" +
		"    <artifactId>spring-boot-jackson2</artifactId>\n" +
		"</dependency>\n" +
		"```\n\n" +
		"Or in Gradle:\n\n" +
		"```\n" +
		"implementation(\"org.springframework.boot:spring-boot-jackson2\")\n" +
		"```\n"

	scanner := NewScanner("java")
	artifacts := scanner.Scan(content)

	found := false
	for _, a := range artifacts {
		if a == "spring-boot-jackson2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find spring-boot-jackson2, got: %v", artifacts)
	}
}

func TestJavaScannerXMLElements(t *testing.T) {
	content := "Remove the following:\n\n" +
		"```\n" +
		"<includeOptional>true</includeOptional>\n" +
		"```\n"

	scanner := NewScanner("java")
	artifacts := scanner.Scan(content)

	found := false
	for _, a := range artifacts {
		if a == "includeOptional" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find includeOptional, got: %v", artifacts)
	}
}

func TestGoScanner(t *testing.T) {
	content := "Replace `golang.org/x/crypto/md4` with a maintained alternative.\n\n" +
		"```go\n" +
		"import \"crypto/md5\"\n" +
		"```\n"

	scanner := NewScanner("go")
	artifacts := scanner.Scan(content)

	want := map[string]bool{
		"golang.org/x/crypto/md4": true,
		"crypto/md5":              true,
	}
	for _, a := range artifacts {
		delete(want, a)
	}
	for missing := range want {
		t.Errorf("missing artifact: %s", missing)
	}
}

func TestGenericScanner(t *testing.T) {
	content := "The `some.config.property` has changed. Use `new-package-name` instead."

	scanner := NewScanner("python")
	artifacts := scanner.Scan(content)

	if len(artifacts) != 2 {
		t.Errorf("expected 2 artifacts, got %d: %v", len(artifacts), artifacts)
	}
}

func TestCheckCoverageNoGaps(t *testing.T) {
	sections := []Section{
		{Heading: "API Change", StartLine: 1, Content: "Use `org.springframework.boot.Something` instead."},
	}
	patterns := &rules.ExtractOutput{
		Patterns: []rules.MigrationPattern{
			{SourceFQN: "org.springframework.boot.Something"},
		},
	}

	result := CheckCoverage(sections, NewScanner("java"), patterns)
	if result.GapCount != 0 {
		t.Errorf("expected 0 gaps, got %d", result.GapCount)
	}
	if result.CoveredSections != 1 {
		t.Errorf("expected 1 covered section, got %d", result.CoveredSections)
	}
}

func TestCheckCoverageWithGap(t *testing.T) {
	sections := []Section{
		{Heading: "Covered Section", StartLine: 1, Content: "Use `org.foo.Bar` instead."},
		{Heading: "Skipped Section", StartLine: 10, Content: "Set `management.health.probes.enabled` to disable."},
		{Heading: "Header Only", StartLine: 20, Content: "\n\n"},
	}
	patterns := &rules.ExtractOutput{
		Patterns: []rules.MigrationPattern{
			{SourceFQN: "org.foo.Bar"},
		},
	}

	result := CheckCoverage(sections, NewScanner("java"), patterns)
	if result.GapCount != 1 {
		t.Fatalf("expected 1 gap, got %d", result.GapCount)
	}
	if result.Gaps[0].Heading != "Skipped Section" {
		t.Errorf("gap heading = %q, want %q", result.Gaps[0].Heading, "Skipped Section")
	}
	if result.CoveredSections != 1 {
		t.Errorf("expected 1 covered section, got %d", result.CoveredSections)
	}
}

func TestCheckCoverageDependencyShortForm(t *testing.T) {
	sections := []Section{
		{Heading: "Dep Section", StartLine: 1, Content: "Remove `spring-boot-starter-undertow`."},
	}
	patterns := &rules.ExtractOutput{
		Patterns: []rules.MigrationPattern{
			{DependencyName: "org.springframework.boot.spring-boot-starter-undertow"},
		},
	}

	result := CheckCoverage(sections, NewScanner("java"), patterns)
	if result.GapCount != 0 {
		t.Errorf("expected 0 gaps (dependency short form match), got %d", result.GapCount)
	}
}

func TestCheckCoverageRegexNormalization(t *testing.T) {
	sections := []Section{
		{Heading: "Property Section", StartLine: 1, Content: "Rename `spring.session.redis.flush-mode` property."},
	}
	patterns := &rules.ExtractOutput{
		Patterns: []rules.MigrationPattern{
			{SourceFQN: "spring\\.session\\.redis"},
		},
	}

	result := CheckCoverage(sections, NewScanner("java"), patterns)
	if result.GapCount != 0 {
		t.Errorf("expected 0 gaps (regex normalization match), got %d", result.GapCount)
	}
}

func TestNewScannerSwitch(t *testing.T) {
	tests := []struct {
		lang string
		typ  string
	}{
		{"java", "*coverage.javaScanner"},
		{"go", "*coverage.goScanner"},
		{"golang", "*coverage.goScanner"},
		{"python", "*coverage.genericScanner"},
		{"", "*coverage.genericScanner"},
	}
	for _, tt := range tests {
		scanner := NewScanner(tt.lang)
		if scanner == nil {
			t.Errorf("NewScanner(%q) returned nil", tt.lang)
		}
	}
}

func TestEmptySectionNotAGap(t *testing.T) {
	sections := []Section{
		{Heading: "Empty Header", StartLine: 1, Content: ""},
		{Heading: "Whitespace Only", StartLine: 5, Content: "   \n\n   "},
	}
	patterns := &rules.ExtractOutput{}

	result := CheckCoverage(sections, NewScanner("java"), patterns)
	if result.GapCount != 0 {
		t.Errorf("expected 0 gaps for empty sections, got %d", result.GapCount)
	}
	if result.SectionsWithContent != 0 {
		t.Errorf("expected 0 sections with content, got %d", result.SectionsWithContent)
	}
}

func TestNoiseFiltering(t *testing.T) {
	content := "Set `true` or `false`. Use `null` value."
	scanner := NewScanner("java")
	artifacts := scanner.Scan(content)
	if len(artifacts) != 0 {
		t.Errorf("expected 0 artifacts (all noise), got %d: %v", len(artifacts), artifacts)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
