// Package ui holds the Wails side of the windows.
package ui

import (
	"context"
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"spellcast/internal/core"
	"spellcast/internal/platform"
)

// Events between Go and the result window's frontend. All of them carry the
// session id, so the frontend can drop events of a session that was replaced.
const (
	EventBegin        = "session:begin"         // Go -> frontend, payload: core.View
	EventDelta        = "session:delta"         // Go -> frontend, payload: DeltaPayload
	EventNotes        = "session:notes"         // Go -> frontend, payload: DeltaPayload
	EventDone         = "session:done"          // Go -> frontend, payload: DonePayload
	EventFail         = "session:fail"          // Go -> frontend, payload: FailPayload
	EventReplaceState = "session:replace-state" // Go -> frontend, payload: ReplaceStatePayload
	EventReplace      = "session:replace"       // frontend -> Go, payload: {id}
)

// DeltaPayload is a piece of the answer.
type DeltaPayload struct {
	ID   uint64 `json:"id"`
	Text string `json:"text"`
}

// DonePayload ends the answer. StopReason is e.g. "end_turn", "max_tokens",
// "refusal" or "cancelled".
type DonePayload struct {
	ID         uint64 `json:"id"`
	StopReason string `json:"stopReason"`
	Usage      string `json:"usage"` // e.g. "312 in · 187 out · ≈ $0.0001"; may be empty
}

// FailPayload ends the session with a message for the user.
type FailPayload struct {
	ID      uint64 `json:"id"`
	Message string `json:"message"`
}

// ReplaceStatePayload tells the frontend whether Replace is possible.
type ReplaceStatePayload struct {
	ID         uint64 `json:"id"`
	CanReplace bool   `json:"canReplace"`
	Reason     string `json:"reason"`
}

// ResultWindow is the single, reused result window. It implements core.UI.
type ResultWindow struct {
	app      *application.App
	win      *application.WebviewWindow
	platform platform.Platform

	onClose   func()
	onReplace func(id uint64)

	mu          sync.Mutex
	target      platform.Target // the window the current session came from
	id          uint64          // the current session
	visible     bool
	lastReplace ReplaceStatePayload // last state sent, to send changes only
}

var _ core.UI = (*ResultWindow)(nil)

// NewResultWindow creates the window, hidden until the first Begin.
// sizePath is the file that remembers the window size between sessions.
func NewResultWindow(app *application.App, p platform.Platform, sizePath string) *ResultWindow {
	size := loadSize(sizePath)
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:        "result",
		Title:       "Spellcast",
		Width:       size.Width,
		Height:      size.Height,
		MinWidth:    minWidth,
		MinHeight:   minHeight,
		AlwaysOnTop: true,
		Hidden:      true,
		URL:         "/",
	})
	r := &ResultWindow{app: app, win: win, platform: p}

	// The window is reused: closing it (X button, or Esc in the frontend)
	// only hides it. The size is remembered and a running request stops.
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		w, h := win.Size()
		saveSize(sizePath, windowSize{w, h})
		r.setVisible(false)
		win.Hide()
		if r.onClose != nil {
			r.onClose()
		}
	})

	app.Event.On(EventReplace, func(e *application.CustomEvent) { r.handleReplace(e.Data) })
	return r
}

// handleReplace starts a Replace for the session in the event payload. A
// payload or setup that cannot work is logged, so a silent failure can be
// found in the log.
func (r *ResultWindow) handleReplace(data any) {
	id, ok := idFromEvent(data)
	if !ok {
		slog.Warn("replace event without a valid id", "data", data)
		return
	}
	if r.onReplace == nil {
		slog.Warn("replace event but no handler", "id", id)
		return
	}
	go r.onReplace(id)
}

// OnClose sets a function that is called when the user closes the window.
func (r *ResultWindow) OnClose(f func()) { r.onClose = f }

// OnReplace sets the function that is called when the user asks for Replace.
func (r *ResultWindow) OnReplace(f func(id uint64)) { r.onReplace = f }

func (r *ResultWindow) setVisible(v bool) {
	r.mu.Lock()
	r.visible = v
	r.mu.Unlock()
}

// Begin moves the window next to the text the selection came from, tells the
// frontend to start a session and brings the window to the front.
func (r *ResultWindow) Begin(v core.View) {
	r.mu.Lock()
	r.target, r.id, r.visible = v.Target, v.ID, true
	r.lastReplace = ReplaceStatePayload{ID: v.ID, CanReplace: v.CanReplace, Reason: v.ReplaceReason}
	r.mu.Unlock()

	// Placement is computed fresh on every trigger, while the target window
	// still has its caret. "Show last answer" asks for the mouse pointer.
	placeAt := v.Target
	if v.PlaceAtPointer {
		placeAt = platform.Target{}
	}
	anchor, fromCaret := r.platform.Anchor(placeAt)
	slog.Info("placing result window", "anchor", anchor, "fromCaret", fromCaret)
	r.platform.PlaceWindow(uintptr(r.win.NativeWindow()), anchor)

	r.app.Event.Emit(EventBegin, v)
	r.win.Show()
	r.win.Focus()
	r.setOnTop(true, platform.Target{})
}

// Delta sends a piece of the answer.
func (r *ResultWindow) Delta(id uint64, text string) {
	r.app.Event.Emit(EventDelta, DeltaPayload{ID: id, Text: text})
}

// Notes sends a piece of the notes of a two-part answer.
func (r *ResultWindow) Notes(id uint64, text string) {
	r.app.Event.Emit(EventNotes, DeltaPayload{ID: id, Text: text})
}

// Done marks the answer as complete.
func (r *ResultWindow) Done(id uint64, stopReason, usage string) {
	r.app.Event.Emit(EventDone, DonePayload{ID: id, StopReason: stopReason, Usage: usage})
}

// Fail shows an error instead of an answer.
func (r *ResultWindow) Fail(id uint64, message string) {
	r.app.Event.Emit(EventFail, FailPayload{ID: id, Message: message})
}

// SetReplaceable tells the frontend whether Replace is possible.
func (r *ResultWindow) SetReplaceable(id uint64, ok bool, reason string) {
	r.mu.Lock()
	r.lastReplace = ReplaceStatePayload{ID: id, CanReplace: ok, Reason: reason}
	r.mu.Unlock()
	r.app.Event.Emit(EventReplaceState, ReplaceStatePayload{ID: id, CanReplace: ok, Reason: reason})
}

// Hide hides the window without ending the session.
func (r *ResultWindow) Hide() {
	r.setVisible(false)
	r.win.Hide()
}

// Reveal shows the window again, where it is.
func (r *ResultWindow) Reveal() {
	r.setVisible(true)
	r.win.Show()
	r.win.Focus()
	r.setOnTop(true, platform.Target{})
}

// WatchForeground keeps the window on top of its target only. While the
// window is open it is always-on-top whenever the target (or the result
// window itself) is the foreground window. Otherwise it is a normal window,
// so it drops behind other apps and comes back when the target is activated.
// If the target window is closed, the window stays in normal z-order and
// Replace is disabled.
func (r *ResultWindow) WatchForeground(ctx context.Context) {
	r.platform.WatchForeground(ctx, r.onForeground)
}

func (r *ResultWindow) onForeground(fg platform.Target) {
	r.mu.Lock()
	visible, target, id, last := r.visible, r.target, r.id, r.lastReplace
	r.mu.Unlock()
	if !visible || target.Handle == 0 {
		return
	}

	own := uintptr(r.win.NativeWindow())
	if !r.platform.TargetAlive(target) {
		r.setOnTop(false, platform.Target{}) // normal z-order
		if last.CanReplace || last.Reason == "" {
			r.SetReplaceable(id, false, "The source window was closed.")
		}
		return
	}
	if fg.Handle == target.Handle || fg.Handle == own {
		r.setOnTop(true, platform.Target{})
	} else {
		r.setOnTop(false, fg) // drop directly behind the app in front
	}
}

// setOnTop changes the z-order without taking the focus.
func (r *ResultWindow) setOnTop(onTop bool, behind platform.Target) {
	r.platform.SetAlwaysOnTop(uintptr(r.win.NativeWindow()), onTop, behind)
}

// idFromEvent reads the session id from an event payload {id: n}.
func idFromEvent(data any) (uint64, bool) {
	n, ok := eventMap(data)["id"].(float64)
	if !ok || n < 0 {
		return 0, false
	}
	return uint64(n), true
}

// eventMap reads an event payload that is a JSON object. Depending on the
// Wails version it arrives as the map itself or wrapped in a one-element list.
func eventMap(data any) map[string]any {
	if list, ok := data.([]any); ok && len(list) > 0 {
		data = list[0]
	}
	m, _ := data.(map[string]any)
	return m
}
