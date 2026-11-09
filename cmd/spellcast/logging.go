package main

import (
	"log/slog"
	"os"
	"path/filepath"
)

// logFile is the open log file (nil if logging goes to stderr).
var logFile *os.File

// localDir is the folder for what the machine can recreate: the log and the
// WebView2 user data (<UserCacheDir>/spellcast, Windows: %LocalAppData%\spellcast).
// The settings stay in config.Dir.
func localDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "spellcast"), nil
}

// webviewDataDir is the WebView2 user data folder, inside localDir. Without it,
// Wails would use %AppData%\<exe name>: a roaming folder named after the exe.
func webviewDataDir() (string, error) {
	dir, err := localDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "WebView2"), nil
}

// setupLogging writes slog output to spellcast.log in localDir. The GUI exe
// has no console. It returns the log path; on failure logging stays on stderr.
func setupLogging() string {
	dir, err := localDir()
	if err != nil {
		slog.Error("no log directory", "err", err)
		return ""
	}
	path := filepath.Join(dir, "spellcast.log")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("cannot create log directory", "err", err)
		return ""
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Error("cannot open log file", "err", err)
		return path
	}
	logFile = f
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	return path
}

// closeLog closes the log file and sends further log output to stderr, so the
// log folder can be deleted (Windows cannot delete an open file).
func closeLog() error {
	if logFile == nil {
		return nil
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	err := logFile.Close()
	logFile = nil
	return err
}
