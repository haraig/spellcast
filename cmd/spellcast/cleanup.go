package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"spellcast/internal/claude"
	"spellcast/internal/config"
	"spellcast/internal/platform"
)

// cleanupDeps are the side effects of "Remove from Windows" (fakes in tests).
type cleanupDeps struct {
	DisableAutostart func() error
	RemoveNotifier   func() error // the notification registration in HKCU
	DeleteKey        func() error
	CloseLog         func() error
	RemoveAll        func(path string) error
	ConfigDir        string // %AppData%\spellcast: config.yaml, usage.json, window.json
	LogDir           string // %LocalAppData%\spellcast: spellcast.log
	WebViewDir       string // %AppData%\spellcast.exe: the WebView2 user data
}

// cleanup removes the autostart entry and the notification registration and,
// with purge, the settings, the WebView2 data, the log and the stored API key.
// Run it only when no window and no notification service is running (they
// would write some of it again). Every step runs even if an earlier one fails; the
// error names each failed step. The exe itself is never deleted.
func cleanup(purge bool, d cleanupDeps) error {
	var errs []error
	step := func(what string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", what, err))
		}
	}
	step("autostart entry", d.DisableAutostart())
	step("notification registration", d.RemoveNotifier())
	if purge {
		step("API key", d.DeleteKey())
		step("settings folder "+d.ConfigDir, d.RemoveAll(d.ConfigDir))
		step("WebView2 folder "+d.WebViewDir, d.RemoveAll(d.WebViewDir))
		step("closing the log", d.CloseLog())
		step("log folder "+d.LogDir, d.RemoveAll(d.LogDir))
	}
	return errors.Join(errs...)
}

// realCleanupDeps are the real side effects on this machine.
func realCleanupDeps(plat platform.Platform) (cleanupDeps, error) {
	cfgDir, err := config.Dir()
	if err != nil {
		return cleanupDeps{}, err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return cleanupDeps{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return cleanupDeps{}, err
	}
	return cleanupDeps{
		DisableAutostart: func() error { return plat.SetAutostart(false) },
		RemoveNotifier:   func() error { return platform.RemoveNotificationRegistration(appName) },
		DeleteKey:        claude.DeleteKey,
		CloseLog:         closeLog,
		// A second spellcast.exe that handed --uninstall over may hold the log
		// file open for a moment.
		RemoveAll: retryRemoveAll(os.RemoveAll, 10, 200*time.Millisecond),
		ConfigDir: cfgDir,
		LogDir:    filepath.Join(cache, "spellcast"),
		// Wails' default WebView2 user data folder: %AppData%\<exe name>.
		WebViewDir: filepath.Join(filepath.Dir(cfgDir), filepath.Base(exe)),
	}, nil
}

// retryRemoveAll wraps remove so that it tries up to tries times, waiting
// pause between tries.
func retryRemoveAll(remove func(string) error, tries int, pause time.Duration) func(string) error {
	return func(path string) error {
		var err error
		for i := 0; i < tries; i++ {
			if err = remove(path); err == nil {
				return nil
			}
			time.Sleep(pause)
		}
		return err
	}
}

// removeFromWindows runs the cleanup on this machine and logs the result
// (with purge, the log goes to stderr, because its folder is deleted).
func removeFromWindows(plat platform.Platform, purge bool) error {
	d, err := realCleanupDeps(plat)
	if err == nil {
		err = cleanup(purge, d)
	}
	if err != nil {
		slog.Error("remove from Windows failed", "purge", purge, "err", err)
		return err
	}
	slog.Info("removed from Windows", "purge", purge)
	return nil
}

// confirmRemove asks before "Remove from Windows". remove(purge) does the work.
func confirmRemove(app *application.App, remove func(purge bool)) {
	d := app.Dialog.Question().
		SetTitle("Remove Spellcast from Windows").
		SetMessage("Remove: Spellcast does not start with Windows anymore. Your settings and API key stay.\n\n" +
			"Remove and delete my settings: also deletes the config, the usage totals, the log and the stored API key.\n\n" +
			"Spellcast quits afterwards. The exe file itself stays: delete it yourself, or run winget uninstall.")
	d.AddButton("Remove").OnClick(func() { remove(false) })
	d.AddButton("Remove and delete my settings").OnClick(func() { remove(true) })
	cancel := d.AddButton("Cancel")
	d.SetCancelButton(cancel)
	d.SetDefaultButton(cancel)
	d.Show()
}
