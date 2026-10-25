package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSizeDefaults(t *testing.T) {
	dir := t.TempDir()
	def := windowSize{defaultWidth, defaultHeight}

	if got := loadSize(filepath.Join(dir, "missing.json")); got != def {
		t.Errorf("missing file: %+v", got)
	}
	for name, content := range map[string]string{
		"garbage": "not json",
		"tiny":    `{"width":10,"height":10}`,
		"huge":    `{"width":99999,"height":300}`,
		"zero":    `{}`,
	} {
		path := filepath.Join(dir, name+".json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := loadSize(path); got != def {
			t.Errorf("%s: %+v, want default", name, got)
		}
	}
}

func TestSaveAndLoadSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "window.json")
	want := windowSize{700, 450}
	saveSize(path, want)
	if got := loadSize(path); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSaveSizeIgnoresInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.json")
	saveSize(path, windowSize{0, 0}) // e.g. a minimized window
	if _, err := os.Stat(path); err == nil {
		t.Error("invalid size was saved")
	}
}
