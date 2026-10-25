package core

import "testing"

func TestAdaptText(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		original string
		eol      string
		want     string
	}{
		{"plain word", "Hello", "Hallo", "\r\n", "Hello"},
		{"answer is trimmed", "\n  Hello world \n", "Hallo Welt", "\r\n", "Hello world"},
		{"leading and trailing spaces of the original come back", "Hello", "  Hallo \t", "\r\n", "  Hello \t"},
		{"full-line selection keeps its trailing line break (LF)", "Print it", "print it\n", "\r\n", "Print it\n"},
		{"full-line selection keeps its trailing line break (CRLF)", "Print it", "print it\r\n", "\n", "Print it\r\n"},
		{"indentation is kept", "return 1", "    return 0\n", "\r\n", "    return 1\n"},
		{"multi-line LF original: answer gets LF", "one\r\ntwo\r\nthree", "eins\nzwei\ndrei", "\r\n", "one\ntwo\nthree"},
		{"multi-line CRLF original: answer gets CRLF", "one\ntwo", "eins\r\nzwei", "\n", "one\r\ntwo"},
		{"mixed original: CRLF wins", "a\nb", "x\r\ny\nz", "\n", "a\r\nb"},
		{"no line break in the original: OS default (Windows)", "one\ntwo", "eins zwei", "\r\n", "one\r\ntwo"},
		{"no line break in the original: OS default (macOS)", "one\r\ntwo", "eins zwei", "\n", "one\ntwo"},
		{"lone CR counts as a line break", "a\rb", "x\ry", "\r\n", "a\nb"},
		{"blank lines inside are kept", "a\n\nb", "x\n\ny", "\r\n", "a\n\nb"},
		{"unicode", "Grüße 👋", "Hello", "\r\n", "Grüße 👋"},
		{"whitespace-only original", "Hi", "  \n", "\r\n", "  \nHi"},
		{"empty answer", "  \n", "Hallo\n", "\r\n", "\n"},
		{"empty original", "Hi", "", "\r\n", "Hi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AdaptText(tt.answer, tt.original, tt.eol); got != tt.want {
				t.Errorf("AdaptText(%q, %q, %q) = %q, want %q", tt.answer, tt.original, tt.eol, got, tt.want)
			}
		})
	}
}
