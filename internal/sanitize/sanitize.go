package sanitize

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reXMLComment = regexp.MustCompile(`(?s)<!--(.*?)-->`)

// XMLComments removes "--" sequences inside XML comments, which are
// illegal in XML and break Maven's POM parser. LLMs frequently generate
// comments like <!-- --add-opens flag --> which is invalid XML.
func XMLComments(content string) string {
	return reXMLComment.ReplaceAllStringFunc(content, func(match string) string {
		inner := match[4 : len(match)-3]
		inner = strings.ReplaceAll(inner, "--", "  ")
		return "<!--" + inner + "-->"
	})
}

// Dir walks a directory and sanitizes all XML files (.xml, .csproj, .pom).
func Dir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".xml" && ext != ".csproj" && ext != ".pom" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		cleaned := XMLComments(string(data))
		if cleaned != string(data) {
			if err := os.WriteFile(path, []byte(cleaned), info.Mode()); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}
		}
		return nil
	})
}
