package ingestion

import (
	"bytes"
	"fmt"
	"strings"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLToMarkdown converts HTML content to clean markdown.
func HTMLToMarkdown(htmlContent string) (string, error) {
	md, err := htmltomd.ConvertString(htmlContent)
	if err != nil {
		return "", fmt.Errorf("converting HTML to markdown: %w", err)
	}
	return strings.TrimSpace(md), nil
}

// stripTags are HTML elements removed before markdown conversion.
var stripTags = map[atom.Atom]bool{
	atom.Nav:    true,
	atom.Footer: true,
	atom.Aside:  true,
	atom.Script: true,
	atom.Style:  true,
	atom.Noscript: true,
}

// ExtractArticle strips navigation, footer, sidebar, and script elements
// from HTML. If an <article> or <main> element exists, only its content
// is returned. Otherwise the full <body> is returned with noise stripped.
func ExtractArticle(rawHTML string) string {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return rawHTML
	}

	if node := findElement(doc, atom.Article); node != nil {
		return renderNode(node)
	}
	if node := findElement(doc, atom.Main); node != nil {
		return renderNode(node)
	}

	body := findElement(doc, atom.Body)
	if body == nil {
		return rawHTML
	}
	removeElements(body, stripTags)
	return renderNode(body)
}

func findElement(n *html.Node, tag atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func removeElements(n *html.Node, tags map[atom.Atom]bool) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type == html.ElementNode && tags[c.DataAtom] {
			n.RemoveChild(c)
		} else {
			removeElements(c, tags)
		}
	}
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	html.Render(&buf, n)
	return buf.String()
}
