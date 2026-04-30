package coverage

import (
	"regexp"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

type Section struct {
	Heading   string `json:"heading"`
	Level     int    `json:"level"`
	StartLine int    `json:"start_line"`
	Content   string `json:"-"`
}

type Gap struct {
	Heading   string   `json:"heading"`
	Line      int      `json:"line"`
	Artifacts []string `json:"artifacts"`
}

type Result struct {
	TotalSections       int   `json:"total_sections"`
	SectionsWithContent int   `json:"sections_with_content"`
	CoveredSections     int   `json:"covered_sections"`
	GapCount            int   `json:"gap_count"`
	Gaps                []Gap `json:"gaps"`
}

// ParseSections splits a markdown guide into sections by heading.
func ParseSections(guide string) []Section {
	lines := strings.Split(guide, "\n")
	var sections []Section
	var current *Section

	for i, line := range lines {
		level, title := parseHeading(line)
		if level > 0 {
			if current != nil {
				current.Content = strings.Join(lines[current.StartLine:i], "\n")
				sections = append(sections, *current)
			}
			current = &Section{
				Heading:   title,
				Level:     level,
				StartLine: i + 1,
			}
		}
	}
	if current != nil {
		current.Content = strings.Join(lines[current.StartLine:], "\n")
		sections = append(sections, *current)
	}
	return sections
}

func parseHeading(line string) (int, string) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 || trimmed[0] != '#' {
		return 0, ""
	}
	level := 0
	for _, c := range trimmed {
		if c == '#' {
			level++
		} else {
			break
		}
	}
	if level > 6 {
		return 0, ""
	}
	title := strings.TrimSpace(trimmed[level:])
	if title == "" {
		return 0, ""
	}
	return level, title
}

// Scanner extracts artifact names from section content.
type Scanner interface {
	Scan(content string) []string
}

// NewScanner returns a language-appropriate scanner.
func NewScanner(language string) Scanner {
	switch strings.ToLower(language) {
	case "java":
		return &javaScanner{}
	case "go", "golang":
		return &goScanner{}
	default:
		return &genericScanner{}
	}
}

// CheckCoverage finds sections with artifacts not covered by any pattern.
// A section is a "gap" when it contains artifacts and NONE match patterns.json.
func CheckCoverage(sections []Section, scanner Scanner, patterns *rules.ExtractOutput) *Result {
	covered := buildCoveredSet(patterns)
	result := &Result{TotalSections: len(sections)}

	for _, section := range sections {
		artifacts := scanner.Scan(section.Content)
		if len(artifacts) == 0 {
			continue
		}
		result.SectionsWithContent++

		var uncovered []string
		anyCovered := false
		for _, a := range artifacts {
			if matchesCovered(a, covered) {
				anyCovered = true
			} else {
				uncovered = append(uncovered, a)
			}
		}

		if anyCovered {
			result.CoveredSections++
		} else {
			result.GapCount++
			result.Gaps = append(result.Gaps, Gap{
				Heading:   section.Heading,
				Line:      section.StartLine,
				Artifacts: uncovered,
			})
		}
	}
	return result
}

func buildCoveredSet(patterns *rules.ExtractOutput) []string {
	var covered []string
	for _, p := range patterns.Patterns {
		if p.SourceFQN != "" {
			norm := normalizePattern(p.SourceFQN)
			covered = append(covered, norm)
			if cls := lastDotSegment(norm); cls != "" {
				covered = append(covered, cls)
			}
		}
		if p.SourcePattern != "" {
			covered = append(covered, p.SourcePattern)
		}
		if p.TargetPattern != "" {
			covered = append(covered, p.TargetPattern)
		}
		if p.DependencyName != "" {
			covered = append(covered, p.DependencyName)
			parts := strings.Split(p.DependencyName, ".")
			if len(parts) > 1 {
				covered = append(covered, parts[len(parts)-1])
			}
		}
		if p.XPath != "" {
			covered = append(covered, p.XPath)
		}
		if p.Message != "" {
			covered = append(covered, p.Message)
		}
	}
	return covered
}

func lastDotSegment(s string) string {
	i := strings.LastIndex(s, ".")
	if i < 0 || i == len(s)-1 {
		return ""
	}
	return s[i+1:]
}

// normalizePattern strips common regex escaping so comparison works.
func normalizePattern(s string) string {
	s = strings.ReplaceAll(s, "\\.", ".")
	s = strings.ReplaceAll(s, "\\(", "(")
	s = strings.ReplaceAll(s, "\\)", ")")
	return s
}

func matchesCovered(artifact string, covered []string) bool {
	lower := strings.ToLower(artifact)
	for _, c := range covered {
		cLower := strings.ToLower(c)
		if strings.Contains(cLower, lower) || strings.Contains(lower, cLower) {
			return true
		}
	}
	return false
}

// --- Markdown extraction helpers ---

var inlineCodeRe = regexp.MustCompile("`([^`]+)`")

func extractInlineCode(content string) []string {
	clean := stripCodeBlocks(content)
	var refs []string
	for _, m := range inlineCodeRe.FindAllStringSubmatch(clean, -1) {
		refs = append(refs, m[1])
	}
	return refs
}

func extractCodeBlocks(content string) []string {
	var blocks []string
	lines := strings.Split(content, "\n")
	inBlock := false
	var block []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				blocks = append(blocks, strings.Join(block, "\n"))
				block = nil
			}
			inBlock = !inBlock
			continue
		}
		if inBlock {
			block = append(block, line)
		}
	}
	return blocks
}

func stripCodeBlocks(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inBlock = !inBlock
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

var noise = map[string]bool{
	"true": true, "false": true, "null": true, "void": true,
	"string": true, "int": true, "long": true, "boolean": true,
	"byte": true, "char": true, "double": true, "float": true,
	"class": true, "interface": true, "enum": true, "import": true,
	"public": true, "private": true, "protected": true, "static": true,
	"final": true, "abstract": true, "return": true, "this": true,
	"new": true, "extends": true, "implements": true, "package": true,
	"override": true, "deprecated": true, "test": true,
}

var (
	versionRe  = regexp.MustCompile(`^\d+\.\d+(\.\d+)?[.x-]*$`)
	templateRe = regexp.MustCompile(`<[a-z]`)
)

func isNoise(s string) bool {
	if len(s) < 4 {
		return true
	}
	if noise[strings.ToLower(s)] {
		return true
	}
	if versionRe.MatchString(s) {
		return true
	}
	if templateRe.MatchString(s) {
		return true
	}
	if strings.Contains(s, "(") || strings.Contains(s, ")") {
		return true
	}
	return false
}

// --- Java Scanner ---

type javaScanner struct{}

var (
	javaFQNRe      = regexp.MustCompile(`[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*){2,}\.[A-Z]\w*`)
	javaPropertyRe = regexp.MustCompile(`[a-z][a-z0-9]*(?:\.[a-z][a-z0-9-]*){2,}`)
	mavenArtifactRe = regexp.MustCompile(`<artifactId>([^<]+)</artifactId>`)
	xmlElementRe    = regexp.MustCompile(`<([a-z][a-zA-Z]{2,})>`)
	depCoordRe      = regexp.MustCompile(`["']([a-z][a-z0-9.-]*):([a-z][a-z0-9-]+)["']`)
)

func (s *javaScanner) Scan(content string) []string {
	seen := make(map[string]bool)
	var artifacts []string

	add := func(text string) {
		if !isNoise(text) && !seen[text] {
			seen[text] = true
			artifacts = append(artifacts, text)
		}
	}

	for _, ref := range extractInlineCode(content) {
		if name, ok := strings.CutPrefix(ref, "@"); ok {
			add(name)
		} else if strings.Contains(ref, ".") && strings.Count(ref, ".") >= 2 {
			add(ref)
		}
	}

	blockText := strings.Join(extractCodeBlocks(content), "\n")
	for _, m := range javaFQNRe.FindAllString(blockText, -1) {
		add(m)
	}
	for _, m := range javaPropertyRe.FindAllString(blockText, -1) {
		if !seen[m] {
			add(m)
		}
	}
	for _, m := range mavenArtifactRe.FindAllStringSubmatch(blockText, -1) {
		add(m[1])
	}
	for _, m := range xmlElementRe.FindAllStringSubmatch(blockText, -1) {
		add(m[1])
	}
	for _, m := range depCoordRe.FindAllStringSubmatch(blockText, -1) {
		add(m[2])
	}

	return artifacts
}

// --- Go Scanner ---

type goScanner struct{}

var (
	goModuleRe = regexp.MustCompile(`[a-z][a-z0-9-]*\.[a-z]{2,}(?:/[a-z][a-z0-9._-]*)+`)
	goStdlibRe = regexp.MustCompile(`"([a-z][a-z0-9]*/[a-z][a-z0-9]*)"`)
)

func (s *goScanner) Scan(content string) []string {
	seen := make(map[string]bool)
	var artifacts []string

	add := func(text string) {
		if !isNoise(text) && !seen[text] {
			seen[text] = true
			artifacts = append(artifacts, text)
		}
	}

	for _, ref := range extractInlineCode(content) {
		if strings.Contains(ref, "/") || strings.Contains(ref, ".") {
			add(ref)
		}
	}

	blockText := strings.Join(extractCodeBlocks(content), "\n")
	for _, m := range goModuleRe.FindAllString(blockText, -1) {
		add(m)
	}
	for _, m := range goStdlibRe.FindAllStringSubmatch(blockText, -1) {
		add(m[1])
	}

	return artifacts
}

// --- Generic Scanner ---

type genericScanner struct{}

func (s *genericScanner) Scan(content string) []string {
	seen := make(map[string]bool)
	var artifacts []string

	for _, ref := range extractInlineCode(content) {
		if !isNoise(ref) && !seen[ref] && (strings.Contains(ref, ".") || strings.Contains(ref, "/") || strings.Contains(ref, "-")) {
			seen[ref] = true
			artifacts = append(artifacts, ref)
		}
	}
	return artifacts
}
