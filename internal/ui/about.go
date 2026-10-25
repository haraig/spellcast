package ui

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Events of the About window.
const (
	EventAboutInfo     = "about:info"      // Go -> frontend, payload: AboutInfo
	EventAboutOpenRepo = "about:open-repo" // frontend -> Go, no payload
)

// AboutInfo is what the About window shows.
type AboutInfo struct {
	Version   string `json:"version"`   // for example "Version v0.1.0 (abc1234)"
	Copyright string `json:"copyright"` // "© 2026 Harald Aigner"
	License   string `json:"license"`   // "MIT License"
	RepoURL   string `json:"repoUrl"`
	Notices   string `json:"notices"` // third-party licenses, plain text
}

// AboutWindow is the small window with the version, the copyright and the licenses.
type AboutWindow struct {
	app  *application.App
	win  *application.WebviewWindow
	info AboutInfo
}

// NewAboutWindow creates the window, hidden until Show.
func NewAboutWindow(app *application.App, info AboutInfo) *AboutWindow {
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "about",
		Title:         "About Spellcast",
		Width:         420,
		Height:        380,
		DisableResize: true,
		Hidden:        true,
		URL:           "/?view=about",
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel() // reused: closing only hides it
		win.Hide()
	})
	app.Event.On(EventAboutOpenRepo, func(*application.CustomEvent) {
		if err := app.Browser.OpenURL(info.RepoURL); err != nil {
			slog.Error("cannot open the repository page", "err", err)
		}
	})
	return &AboutWindow{app: app, win: win, info: info}
}

// Show opens the window.
func (w *AboutWindow) Show() {
	w.app.Event.Emit(EventAboutInfo, w.info)
	w.win.Center()
	w.win.Show()
	w.win.Focus()
}
