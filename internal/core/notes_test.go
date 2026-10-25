package core

import (
	"strings"
	"testing"
)

func TestSplitNotes(t *testing.T) {
	tests := []struct{ name, in, text, notes string }{
		{"no rule", "just text\nmore", "just text\nmore", ""},
		{"rule", "Hello\n---\nAlt: Hi", "Hello\n", "Alt: Hi"},
		{"rule with CRLF", "Hello\r\n---\r\nAlt", "Hello\r\n", "Alt"},
		{"rule with trailing blank", "Hello\n--- \nAlt", "Hello\n", "Alt"},
		{"rule at the end", "Hello\n---", "Hello\n", ""},
		{"rule only", "---\nnotes", "", "notes"},
		{"first rule wins", "a\n---\nb\n---\nc", "a\n", "b\n---\nc"},
		{"longer dashes are text", "a\n----\nb", "a\n----\nb", ""},
		{"dashes inside a line", "a --- b\nc", "a --- b\nc", ""},
		{"indented is text", "a\n  ---\nb", "a\n  ---\nb", ""},
		{"single dash at the end", "a\n-", "a\n-", ""},
		{"two dashes at the end", "a\n--", "a\n--", ""},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, notes := SplitNotes(tt.in)
			if text != tt.text || notes != tt.notes {
				t.Errorf("SplitNotes(%q) = (%q, %q), want (%q, %q)", tt.in, text, notes, tt.text, tt.notes)
			}
		})
	}
}

// The result must not depend on how the stream is cut into pieces.
func TestSplitterStreamingEqualsWhole(t *testing.T) {
	inputs := []string{
		"Hello\n---\nAlt: Hi\n",
		"a\n----\nb\n---\nc",
		"x\n--",
		"x\n---",
		"--- \r\nnotes",
		"plain text without a rule",
	}
	for _, in := range inputs {
		wantText, wantNotes := SplitNotes(in)
		for size := 1; size <= len(in); size++ {
			sp := splitter{enabled: true}
			var text, notes strings.Builder
			for i := 0; i < len(in); i += size {
				end := min(i+size, len(in))
				tx, nt := sp.Write(in[i:end])
				text.WriteString(tx)
				notes.WriteString(nt)
			}
			text.WriteString(sp.Flush())
			if text.String() != wantText || notes.String() != wantNotes {
				t.Fatalf("%q in pieces of %d: (%q, %q), want (%q, %q)", in, size, text.String(), notes.String(), wantText, wantNotes)
			}
		}
	}
}

func TestSplitterHoldsBackOnlyPossibleRule(t *testing.T) {
	sp := splitter{enabled: true}
	if text, _ := sp.Write("abc\n--"); text != "abc\n" {
		t.Errorf("a partial rule line must be held back, got %q", text)
	}
	if text, _ := sp.Write("x"); text != "--x" {
		t.Errorf("when it turns out to be text, it is released, got %q", text)
	}
	if text, _ := sp.Write("def"); text != "def" {
		t.Errorf("other partial lines are shown at once, got %q", text)
	}
}

func TestSplitterDisabled(t *testing.T) {
	sp := splitter{}
	if text, notes := sp.Write("a\n---\nb"); text != "a\n---\nb" || notes != "" {
		t.Errorf("(%q, %q)", text, notes)
	}
	if sp.Flush() != "" {
		t.Error("nothing to flush")
	}
}
