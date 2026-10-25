package ui

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Events of the "Set API key" window.
const (
	EventKeyState  = "apikey:state"  // Go -> frontend, payload: KeyStatePayload
	EventKeySave   = "apikey:save"   // frontend -> Go, payload: {key}
	EventKeyResult = "apikey:result" // Go -> frontend, payload: KeyResultPayload
)

// KeyStatePayload tells the window where the current key comes from.
type KeyStatePayload struct {
	Source string `json:"source"` // "", "credential store" or "ANTHROPIC_API_KEY"
}

// KeyResultPayload is the outcome of a save.
type KeyResultPayload struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// SaveKeyFunc checks and stores a key. The message is shown to the user,
// whether the save worked or not. The key must never appear in it.
type SaveKeyFunc func(key string) (message string, err error)

// APIKeyWindow is the small window with the password field.
type APIKeyWindow struct {
	app *application.App
	win *application.WebviewWindow
}

// NewAPIKeyWindow creates the window, hidden until Show. save runs in the
// background, so the window stays responsive during the check.
func NewAPIKeyWindow(app *application.App, save SaveKeyFunc) *APIKeyWindow {
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "apikey",
		Title:         "Spellcast: API key",
		Width:         460,
		Height:        296,
		DisableResize: true,
		Hidden:        true,
		URL:           "/?view=apikey",
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel() // reused: closing only hides it
		win.Hide()
	})

	app.Event.On(EventKeySave, func(e *application.CustomEvent) {
		key := keyFromEvent(e.Data)
		go func() {
			msg, err := save(key)
			if err != nil {
				slog.Warn("API key not saved", "reason", msg) // never log the key
			}
			app.Event.Emit(EventKeyResult, KeyResultPayload{OK: err == nil, Message: msg})
		}()
	})
	return &APIKeyWindow{app: app, win: win}
}

// Show opens the window. source says where the current key comes from ("" if none).
func (w *APIKeyWindow) Show(source string) {
	w.app.Event.Emit(EventKeyState, KeyStatePayload{Source: source})
	w.win.Center()
	w.win.Show()
	w.win.Focus()
}

// keyFromEvent reads the key from the event payload {key: "..."}.
func keyFromEvent(data any) string {
	key, _ := eventMap(data)["key"].(string)
	return key
}
