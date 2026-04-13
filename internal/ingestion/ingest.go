package ingestion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// InputType classifies the input source.
type InputType int

const (
	InputURL  InputType = iota
	InputFile
	InputText
)

// Fetch limits.
const (
	fetchTimeout    = 30 * time.Second
	maxResponseBody = 10 << 20 // 10 MB
	maxRedirects    = 5
)

// Result holds ingested and cleaned content.
type Result struct {
	Content   string
	Source    InputType
	Chunks   []string
	ChunkSize int
}

// Ingest detects the input type and returns cleaned content.
// For URLs, it fetches and converts HTML to markdown.
// For file paths, it reads the file.
// For anything else, it treats the input as raw text.
func Ingest(ctx context.Context, input string, maxChunkSize int) (*Result, error) {
	inputType := DetectType(input)
	var content string
	var err error

	switch inputType {
	case InputURL:
		content, err = fetchURL(ctx, input)
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

// DetectType classifies input as URL, file path, or raw text.
func DetectType(input string) InputType {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return InputURL
	}
	if _, err := os.Stat(input); err == nil {
		return InputFile
	}
	return InputText
}

// httpClient is a hardened HTTP client for URL fetching.
var httpClient = &http.Client{
	Timeout: fetchTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		return nil
	},
}

func fetchURL(ctx context.Context, rawURL string) (string, error) {
	// Block private/loopback IPs to mitigate SSRF
	if err := validateURLHost(rawURL); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %s: %w", rawURL, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching %s: HTTP %d", rawURL, resp.StatusCode)
	}

	// Cap response body to prevent OOM on large responses
	limited := io.LimitReader(resp.Body, maxResponseBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("reading response from %s: %w", rawURL, err)
	}
	if len(body) > maxResponseBody {
		return "", fmt.Errorf("response from %s exceeds %d byte limit", rawURL, maxResponseBody)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return HTMLToMarkdown(string(body))
	}

	return string(body), nil
}

// validateURLHost blocks requests to private/loopback addresses to mitigate SSRF.
func validateURLHost(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL %q has no hostname", rawURL)
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		// DNS resolution failed — let the HTTP client handle it
		// (could be a transient error or the host may resolve later via different DNS)
		return nil
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.New("URL resolves to a private/loopback address; blocked for security")
		}
	}

	return nil
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	return string(data), nil
}
