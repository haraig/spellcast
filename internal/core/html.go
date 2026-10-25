package core

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// md renders the Markdown a model produces. Raw HTML in the answer is dropped
// (goldmark's default), so the clipboard never gets script or style tags.
var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

// markdownToHTML converts an answer to an HTML fragment for the clipboard.
// It returns "" if the conversion fails; the caller then pastes plain text.
func markdownToHTML(answer string) string {
	var b strings.Builder
	if err := md.Convert([]byte(strings.Trim(answer, edgeSpace)), &b); err != nil {
		return ""
	}
	return strings.TrimSpace(b.String())
}
