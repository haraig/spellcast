package core

import (
	"strings"
	"testing"
)

func TestMarkdownToHTML(t *testing.T) {
	got := markdownToHTML("\n**Hallo** <script>x()</script>\n\n- a\n- b\n")
	for _, want := range []string{"<strong>Hallo</strong>", "<ul>", "<li>a</li>"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "<script") {
		t.Errorf("raw HTML must be dropped: %q", got)
	}
}
