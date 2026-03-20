package ingestion

import "strings"

const (
	// DefaultMaxChunkSize is the default maximum characters per chunk.
	// Sized to fit within typical LLM context windows with room for prompts.
	DefaultMaxChunkSize = 50000
)

// Chunk splits content into pieces that fit within maxSize characters.
// It preserves section boundaries by splitting on markdown headers (##).
// If a section is too large, it falls back to paragraph splitting.
func Chunk(content string, maxSize int) []string {
	if maxSize <= 0 {
		maxSize = DefaultMaxChunkSize
	}
	if len(content) <= maxSize {
		return []string{content}
	}

	sections := splitSections(content)
	return mergeChunks(sections, maxSize)
}

// splitSections splits content on markdown header boundaries (## and above).
func splitSections(content string) []string {
	lines := strings.Split(content, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isHeader(trimmed) && current.Len() > 0 {
			sections = append(sections, strings.TrimSpace(current.String()))
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(sections, strings.TrimSpace(current.String()))
	}
	return sections
}

func isHeader(line string) bool {
	return strings.HasPrefix(line, "# ") ||
		strings.HasPrefix(line, "## ") ||
		strings.HasPrefix(line, "### ")
}

// mergeChunks combines small sections into chunks up to maxSize,
// splitting oversized sections by paragraphs.
func mergeChunks(sections []string, maxSize int) []string {
	var chunks []string
	var current strings.Builder

	for _, section := range sections {
		if len(section) > maxSize {
			// Flush current buffer
			if current.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
			// Split oversized section by paragraphs
			chunks = append(chunks, splitByParagraphs(section, maxSize)...)
			continue
		}

		if current.Len()+len(section)+2 > maxSize {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(section)
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}

func splitByParagraphs(text string, maxSize int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		if current.Len()+len(para)+2 > maxSize && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}
