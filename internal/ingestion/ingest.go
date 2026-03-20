package ingestion

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// InputType classifies the input source.
type InputType int

const (
	InputURL  InputType = iota
	InputFile
	InputText
)

// Result holds ingested and cleaned content.
type Result struct {
	Content  string
	Source   InputType
	Chunks   []string
	ChunkSize int
}

// Ingest detects the input type and returns cleaned content.
// For URLs, it fetches and converts HTML to markdown.
// For file paths, it reads the file.
// For anything else, it treats the input as raw text.
func Ingest(input string, maxChunkSize int) (*Result, error) {
	inputType := detectType(input)
	var content string
	var err error

	switch inputType {
	case InputURL:
		content, err = fetchURL(input)
	case InputFile:
		content, err = readFile(input)
	case InputText:
		content = input
	}
	if err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty content after ingestion")
	}

	chunks := Chunk(content, maxChunkSize)

	return &Result{
		Content:   content,
		Source:    inputType,
		Chunks:   chunks,
		ChunkSize: maxChunkSize,
	}, nil
}

func detectType(input string) InputType {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return InputURL
	}
	if _, err := os.Stat(input); err == nil {
		return InputFile
	}
	return InputText
}

func fetchURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response from %s: %w", url, err)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return HTMLToMarkdown(string(body))
	}

	return string(body), nil
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	return string(data), nil
}
