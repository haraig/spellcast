package ui

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// captureLog sends the default logger to a buffer until the test ends.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	b := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(b, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return b
}

func TestReplaceEventBadPayloadLogs(t *testing.T) {
	log := captureLog(t)
	r := &ResultWindow{onReplace: func(uint64) { t.Error("handler called for a bad payload") }}
	payloads := []any{map[string]any{}, map[string]any{"id": -1.0}, "x"}
	for _, p := range payloads {
		r.handleReplace(p)
	}
	n := 0
	for _, line := range strings.Split(log.String(), "\n") {
		if !strings.Contains(line, "replace event without a valid id") {
			continue
		}
		n++
		if !strings.Contains(line, "level=WARN") || !strings.Contains(line, "data=") {
			t.Errorf("want a WARN line with the payload, got: %s", line)
		}
	}
	if n != len(payloads) {
		t.Errorf("%d warnings, want %d, log:\n%s", n, len(payloads), log)
	}
}

func TestReplaceEventNoHandlerLogs(t *testing.T) {
	log := captureLog(t)
	(&ResultWindow{}).handleReplace(map[string]any{"id": 1.0})
	if !strings.Contains(log.String(), "replace event but no handler") {
		t.Errorf("no warning, log:\n%s", log)
	}
}
