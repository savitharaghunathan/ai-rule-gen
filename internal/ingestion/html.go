package ingestion

import (
	"fmt"
	"strings"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// HTMLToMarkdown converts HTML content to clean markdown.
func HTMLToMarkdown(html string) (string, error) {
	md, err := htmltomd.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("converting HTML to markdown: %w", err)
	}
	return strings.TrimSpace(md), nil
}
