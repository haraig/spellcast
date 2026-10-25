package core

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// logBuffer collects log output; the core logs from several goroutines.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLog sends the default logger to a buffer until the test ends.
func captureLog(t *testing.T) *logBuffer {
	t.Helper()
	b := &logBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(b, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return b
}

// logLine returns the first log line that contains all parts, or "".
func logLine(log string, parts ...string) string {
	for _, line := range strings.Split(log, "\n") {
		ok := true
		for _, p := range parts {
			if !strings.Contains(line, p) {
				ok = false
				break
			}
		}
		if ok {
			return line
		}
	}
	return ""
}

func TestReplaceNotPossibleLogs(t *testing.T) {
	p := okPlatform()
	c, ui, _ := finishedSession(t, p, "1", "Hallo", "Hello")
	log := captureLog(t)
	p.mu.Lock()
	p.canReplace, p.reason = false, "The source window is unknown."
	p.mu.Unlock()
	c.Replace(1)
	ui.waitFor(t, "replaceable")
	if logLine(log.String(), "level=WARN", "replace not possible", "The source window is unknown.", "NOTEPAD.EXE") == "" {
		t.Errorf("no warning with reason and app, log:\n%s", log)
	}
}

func TestReplaceSuccessLogs(t *testing.T) {
	p := okPlatform()
	c, _, _ := finishedSession(t, p, "3", "text", "**fett**")
	log := captureLog(t)
	c.Replace(1)
	if logLine(log.String(), "level=INFO", "replace done", "id=1", "app=NOTEPAD.EXE", "html=true") == "" {
		t.Errorf("no success line, log:\n%s", log)
	}
}
