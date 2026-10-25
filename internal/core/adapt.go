package core

import "strings"

// whitespace that is carried over from the original selection.
const edgeSpace = " \t\r\n"

// AdaptText fits the model's answer to the selection it replaces:
//   - the answer is trimmed, then the leading and trailing whitespace of the
//     original is put back (e.g. the trailing line break of a full-line
//     selection in VS Code);
//   - all line breaks get the style of the original: "\r\n" if the original
//     contains "\r\n", "\n" if it has any other line break, and defaultEOL if
//     it has none.
func AdaptText(answer, original, defaultEOL string) string {
	body := strings.Trim(answer, edgeSpace)

	var lead, trail string
	if core := strings.Trim(original, edgeSpace); core == "" {
		lead = original
	} else {
		lead = original[:strings.Index(original, core)]
		trail = original[len(lead)+len(core):]
	}

	return withLineEnding(lead+body+trail, lineEnding(original, defaultEOL))
}

// lineEnding returns the line ending style of text, or def if it has no line break.
func lineEnding(text, def string) string {
	switch {
	case strings.Contains(text, "\r\n"):
		return "\r\n"
	case strings.ContainsAny(text, "\r\n"):
		return "\n"
	}
	return def
}

// withLineEnding converts every line break ("\r\n", "\n" or a lone "\r") to eol.
func withLineEnding(text, eol string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if eol == "\n" {
		return text
	}
	return strings.ReplaceAll(text, "\n", eol)
}
