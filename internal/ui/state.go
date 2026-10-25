package ui

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	defaultWidth, defaultHeight = 480, 320
	minWidth, minHeight         = 240, 140
	maxSize                     = 10000
)

// windowSize is the remembered size of the result window (logical pixels,
// the unit of Wails' Size/SetSize).
type windowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (s windowSize) valid() bool {
	return s.Width >= minWidth && s.Height >= minHeight && s.Width <= maxSize && s.Height <= maxSize
}

// loadSize reads the remembered size from path. Missing or invalid data
// gives the default size.
func loadSize(path string) windowSize {
	def := windowSize{defaultWidth, defaultHeight}
	data, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	var s windowSize
	if err := json.Unmarshal(data, &s); err != nil || !s.valid() {
		slog.Warn("ignoring invalid window size file", "path", path)
		return def
	}
	return s
}

// saveSize writes the size to path. Failures are logged, never fatal.
func saveSize(path string, s windowSize) {
	if !s.valid() {
		return
	}
	data, err := json.Marshal(s)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
	}
	if err == nil {
		err = os.WriteFile(path, data, 0o644)
	}
	if err != nil {
		slog.Warn("cannot save window size", "path", path, "err", err)
	}
}
