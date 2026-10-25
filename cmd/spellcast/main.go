package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"spellcast"
	"spellcast/internal/config"
	"spellcast/internal/core"
	"spellcast/internal/platform"
	"spellcast/internal/ui"
	"spellcast/internal/usage"
)

// appName is the Wails app name. The notification registration uses it as
// the app ID and as the name in the header of every notification.
const appName = "Spellcast"

func main() {
	logPath := setupLogging()
	slog.Info("spellcast starting", "version", version, "commit", commit)

	dist, err := fs.Sub(spellcast.Frontend, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	configPath, err := config.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	// --uninstall [--purge] only cleans up (see below): it must not create a
	// config file, so the config is not loaded for it.
	cliAction := secondInstanceAction(os.Args)

	// The very first start (no config yet) shows a hint where Spellcast lives.
	_, statErr := os.Stat(configPath)
	firstStart := errors.Is(statErr, os.ErrNotExist)
	store := newConfigStore(configPath)
	var loadErr error
	if cliAction == secondNotify {
		_, loadErr = store.Reload()
		if loadErr != nil {
			slog.Error("config invalid at startup, using built-in defaults", "path", configPath, "err", loadErr)
			def, err := config.Parse([]byte(config.DefaultYAML))
			if err != nil {
				log.Fatal(err)
			}
			store.cfg = def
		}
	}

	notify := newNotifier()
	// onSecondInstance runs in the first instance when Spellcast is started
	// again. It is set once the app is ready (it needs the platform).
	var onSecondInstance func(secondAction)
	claudeClient := &clientHolder{}
	claudeClient.Reload()

	app := application.New(application.Options{
		Name:        appName,
		Description: "Process selected text with Claude",
		// A second instance exits (exit code 0); the running one gets its command line.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.spellcast.app",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if onSecondInstance != nil {
					onSecondInstance(secondInstanceAction(data.Args))
				}
			},
		},
		Services: []application.Service{
			application.NewService(notify.svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(dist),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		Windows: application.WindowsOptions{
			// The app lives in the tray; closing the result window must not quit it.
			DisableQuitOnLastWindowClosed: true,
		},
	})

	plat := platform.New(func() []string { return store.Get().Replace.Deny.Windows })

	// "spellcast.exe --uninstall [--purge]" with no other instance running:
	// clean up without UI and exit (a second instance was already handed over
	// to the running one inside application.New).
	if cliAction != secondNotify {
		if err := removeFromWindows(plat, cliAction == secondPurge); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	// remove asks for "Remove from Windows": the app quits first, and the
	// cleanup runs after app.Run returns, when no window, WebView2 or
	// notification service can write the files and registry keys again.
	var pendingRemove atomic.Int32 // 0: none, else 1 + purge
	remove := func(purge bool) {
		v := int32(1)
		if purge {
			v = 2
		}
		pendingRemove.Store(v)
		notify.Notify("removed", "Removing Spellcast from Windows", "Spellcast quits now. Then delete spellcast.exe, or run winget uninstall.")
		go app.Quit() // not on the caller's goroutine: Quit waits for the main thread
	}
	onSecondInstance = func(a secondAction) {
		slog.Info("started again", "action", a)
		switch a {
		case secondUninstall:
			remove(false)
		case secondPurge:
			remove(true)
		default:
			notify.Notify("already-running", "Spellcast", "Spellcast is already running")
		}
	}
	var session *core.Core // created below; the tray entries need it

	t := newTray(app)
	t.hasKey = func() bool { _, err := claudeClient.Client(); return err == nil }

	// API key window: hidden until "Set API key…". A saved key changes the menu.
	keyWindow := ui.NewAPIKeyWindow(app, func(key string) (string, error) {
		msg, err := claudeClient.SaveKey(key)
		t.Refresh()
		return msg, err
	})

	t.onShowLast = func() {
		if !session.ShowLast() {
			notify.Notify("no-answer", "Spellcast", "There is no answer yet. Select text and press CapsLock + a digit.")
		}
	}
	t.autostart = plat.AutostartEnabled
	t.onAutostart = func(enabled bool) {
		if err := plat.SetAutostart(enabled); err != nil {
			slog.Error("cannot change autostart", "err", err)
			notify.Notify("autostart-error", "Spellcast: autostart not changed", err.Error())
			t.SetConfig(store.Get()) // show the real state again
		}
	}
	about := ui.NewAboutWindow(app, ui.AboutInfo{
		Version:   versionText(version, commit),
		Copyright: "© 2026 Harald Aigner",
		License:   "MIT License",
		RepoURL:   "https://github.com/haraig/spellcast",
		Notices:   spellcast.Notices,
	})
	t.onAbout = about.Show
	t.onRemove = func() { confirmRemove(app, remove) }
	t.onSetAPIKey = func() { keyWindow.Show(claudeClient.Source()) }
	t.onOpenConfig = func() { openFile(app, configPath) }
	t.onOpenLog = func() { openFile(app, logPath) }
	t.onReload = func() {
		cfg, err := store.Reload()
		if err != nil {
			slog.Error("config reload failed, keeping the previous config", "err", err)
			notify.Notify("config-error", "Spellcast: config not reloaded", err.Error())
			return
		}
		claudeClient.Reload()
		t.SetConfig(cfg)
		slog.Info("config reloaded", "actions", len(cfg.Actions))
		notify.Notify("config-reloaded", "Spellcast", "Config reloaded.")
	}
	t.SetConfig(store.Get())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	triggers := make(chan core.Trigger, 8)

	// Result window: hidden until a selection arrives.
	resultWindow := ui.NewResultWindow(app, plat, filepath.Join(filepath.Dir(configPath), "window.json"))
	session = core.New(resultWindow, func(key string) (config.Action, bool) {
		return store.Get().Action(key)
	}, claudeClient.Client, plat)
	tracker := usage.NewTracker(filepath.Join(filepath.Dir(configPath), "usage.json"), func(model string) (config.Price, bool) {
		return store.Get().Price(model)
	})
	session.SetUsage(tracker)
	tracker.OnChange(t.SetUsage)
	if s := tracker.Summary(); s.Month.Requests > 0 {
		t.SetUsage(s) // totals from earlier runs, before the first request
	}
	go refreshUsageAtMidnight(ctx, tracker, t)
	resultWindow.OnClose(session.Cancel) // closing the window stops the running request
	resultWindow.OnReplace(session.Replace)
	go session.Run(ctx, triggers)

	// Notifications and the hotkeys need the running app.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if firstStart {
			notify.Notify("first-start", "Spellcast", "Spellcast is running in the tray. Select text and press CapsLock+1.")
		}
		if loadErr != nil {
			notify.Notify("config-error", "Spellcast: config invalid, using defaults", loadErr.Error())
		}
		if _, err := claudeClient.Client(); err != nil {
			notify.NotifyNoKey(err)
		}
		resultWindow.WatchForeground(ctx) // on top of the source window only
		startHotkeys(ctx, store, plat, notify, triggers)
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
	if v := pendingRemove.Load(); v != 0 {
		cancel() // stop the request loop, so no usage is written any more
		if err := removeFromWindows(plat, v == 2); err != nil {
			os.Exit(1)
		}
	}
}

func openFile(app *application.App, path string) {
	if path == "" {
		slog.Warn("no path to open")
		return
	}
	if err := app.Browser.OpenFile(path); err != nil {
		slog.Error("cannot open file", "path", path, "err", err)
	}
}
