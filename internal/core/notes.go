package core

import "strings"

// notesRule is the line that separates the usable text from the notes.
const notesRule = "---"

// splitter divides a streamed answer into the usable text and the notes. The
// notes start after the first line that contains only "---". Without such a
// line the whole answer is text. A line counts only when it is complete, so
// the splitter holds back a trailing partial line while it could still turn
// into the rule ("-", "--", "---").
type splitter struct {
	enabled bool
	found   bool   // the rule line was seen; everything after it is notes
	pending string // the current incomplete line, held back
}

// Write takes the next piece of the answer and returns what can be shown now.
func (sp *splitter) Write(piece string) (text, notes string) {
	if !sp.enabled {
		return piece, ""
	}
	if sp.found {
		return "", piece
	}
	sp.pending += piece
	var out strings.Builder
	for {
		i := strings.IndexByte(sp.pending, '\n')
		if i < 0 {
			break
		}
		line := sp.pending[:i+1]
		sp.pending = sp.pending[i+1:]
		if isRule(line) {
			sp.found = true
			return out.String(), sp.takePending()
		}
		out.WriteString(line)
	}
	if sp.pending != "" && !couldBeRule(sp.pending) {
		out.WriteString(sp.takePending())
	}
	return out.String(), ""
}

// Flush is called when the answer is complete. It returns the held-back
// partial line: a last line "---" is the rule (with no notes after it), any
// other is text.
func (sp *splitter) Flush() (text string) {
	if sp.pending == "" {
		return ""
	}
	if isRule(sp.pending) {
		sp.found = true
		sp.pending = ""
		return ""
	}
	return sp.takePending()
}

func (sp *splitter) takePending() string {
	p := sp.pending
	sp.pending = ""
	return p
}

// isRule reports whether line (with or without its line break) is exactly "---",
// ignoring trailing blanks.
func isRule(line string) bool {
	return strings.TrimRight(line, " \t\r\n") == notesRule
}

// couldBeRule reports whether an incomplete line may still become the rule.
func couldBeRule(partial string) bool {
	return strings.HasPrefix(notesRule, strings.TrimRight(partial, " \t\r"))
}

// SplitNotes splits a complete answer the same way as the splitter does.
func SplitNotes(answer string) (text, notes string) {
	sp := splitter{enabled: true}
	text, notes = sp.Write(answer)
	return text + sp.Flush(), notes
}
