package core

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"spellcast/internal/claude"
	"spellcast/internal/config"
	"spellcast/internal/platform"
)

// Message shown when there is no API key. The tray entry is "Set API key…".
const noKeyMessage = "No API key – use the tray menu: Set API key…"

// StopCancelled is the stop reason shown for a session whose request was
// cancelled before the answer was complete.
const StopCancelled = "cancelled"

// replaceTimeout bounds one Replace (activating the target and pasting).
const replaceTimeout = 5 * time.Second

// View describes a session for the window. The JSON names are the event
// payload the frontend reads.
type View struct {
	ID     uint64 `json:"id"`     // identifies the session; all later events carry it
	Action string `json:"action"` // action name, e.g. "Translate"
	Source string `json:"source"` // source app, e.g. "NOTEPAD.EXE"; may be empty
	Output string `json:"output"` // "plain" or "markdown"
	// Original is the selected text, for "Show changes" (a local diff in the
	// window against the answer).
	Original string `json:"original"`

	// Replace is possible now, or the reason why not (shown as a tooltip).
	CanReplace    bool   `json:"canReplace"`
	ReplaceReason string `json:"replaceReason"`

	// Target is the window the text came from. The UI places the result window
	// next to it and keeps it on top of it.
	Target platform.Target `json:"-"`
	// PlaceAtPointer asks for placement at the mouse pointer instead of the
	// target's text cursor (used by "Show last answer").
	PlaceAtPointer bool `json:"-"`
}

// UI shows a session in the result window, reusing the window if it is open.
// Begin starts (or restarts) a session; the other calls belong to the session
// with the given id.
type UI interface {
	Begin(v View)
	Delta(id uint64, text string)
	// Notes sends a piece of the notes of a two-part answer (action option
	// "notes"). Delta carries only the usable text then.
	Notes(id uint64, text string)
	// Done ends the answer. usage is the short usage line ("312 in · 187 out
	// · ≈ $0.0001"), empty if there is none.
	Done(id uint64, stopReason, usage string)
	Fail(id uint64, message string)
	// SetReplaceable updates whether Replace is possible (e.g. the target was closed).
	SetReplaceable(id uint64, ok bool, reason string)
	// Hide hides the window without ending the session; Reveal shows it again
	// where it is.
	Hide()
	Reveal()
}

// UsageRecorder counts a finished request and returns the usage line for the
// window. usage.Tracker implements it.
type UsageRecorder interface {
	Record(model, action string, u claude.Usage) string
}

// ClientFunc returns the current Claude client. It returns claude.ErrNoAPIKey
// if there is no key. It is called for every request, so a new key applies
// without a restart.
type ClientFunc func() (claude.Streamer, error)

// session is everything about one trigger. The last one is kept in memory
// until the next trigger, so "Show last answer" and Replace can use it.
type session struct {
	id        uint64
	action    config.Action
	selection platform.Selection

	// answer is the usable text; for actions with notes it excludes the notes.
	answer     strings.Builder
	notes      strings.Builder
	split      splitter
	finished   bool   // the Done event arrived
	stopReason string // set when finished or cancelled
	failure    string // message if the request failed
	usageLine  string // shown under the finished answer
}

// Core turns triggers into Claude requests and UI updates.
type Core struct {
	ui       UI
	action   func(key string) (config.Action, bool)
	client   ClientFunc
	platform platform.Platform
	usage    UsageRecorder // optional

	mu      sync.Mutex
	nextID  uint64
	cancel  context.CancelFunc // of the running request
	current *session           // the last session, nil before the first trigger
}

// New creates a Core. action looks up the current config on every trigger,
// so a config reload applies immediately.
func New(ui UI, action func(key string) (config.Action, bool), client ClientFunc, p platform.Platform) *Core {
	return &Core{ui: ui, action: action, client: client, platform: p}
}

// SetUsage sets the recorder for finished requests. Call it before Run.
func (c *Core) SetUsage(u UsageRecorder) { c.usage = u }

// Run handles triggers from in until ctx ends or in is closed.
func (c *Core) Run(ctx context.Context, in <-chan Trigger) {
	defer c.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-in:
			if !ok {
				return
			}
			c.handle(ctx, t)
		}
	}
}

// Cancel stops the running request, if any. The UI calls it when the window
// is closed. The partial answer stays available for "Show last answer".
func (c *Core) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if s := c.current; s != nil && !s.finished && s.failure == "" && s.stopReason == "" {
		s.stopReason = StopCancelled
	}
}

// begin cancels the running request and starts a new session.
func (c *Core) begin(parent context.Context, a config.Action, sel platform.Selection) (*session, context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel
	c.nextID++
	c.current = &session{id: c.nextID, action: a, selection: sel, split: splitter{enabled: a.Notes}}
	return c.current, ctx
}

func (c *Core) view(s *session, placeAtPointer bool) View {
	ok, reason := c.platform.CanReplace(s.selection.Target)
	return View{
		ID:             s.id,
		Action:         s.action.Name,
		Source:         s.selection.Target.App,
		Output:         s.action.Output,
		Original:       s.selection.Text,
		CanReplace:     ok,
		ReplaceReason:  reason,
		Target:         s.selection.Target,
		PlaceAtPointer: placeAtPointer,
	}
}

func (c *Core) handle(parent context.Context, t Trigger) {
	a, ok := c.action(t.Action)
	if !ok {
		// The config changed between the request and now.
		slog.Warn("trigger for an action that is no longer configured", "action", t.Action)
		return
	}

	s, ctx := c.begin(parent, a, t.Selection)
	c.ui.Begin(c.view(s, false))

	streamer, err := c.client()
	if err != nil {
		if errors.Is(err, claude.ErrNoAPIKey) {
			c.finish(s, func() { s.failure = noKeyMessage }, func() { c.ui.Fail(s.id, noKeyMessage) })
		} else {
			slog.Error("cannot create the Claude client", "err", err)
			msg := "Cannot read the API key: " + err.Error()
			c.finish(s, func() { s.failure = msg }, func() { c.ui.Fail(s.id, msg) })
		}
		return
	}

	req := claude.Request{
		Model:        a.Model,
		MaxTokens:    a.MaxTokens,
		Effort:       a.Effort,
		Prompt:       a.Prompt,
		Instructions: a.Instructions,
		Notes:        a.Notes,
		Output:       a.Output,
		Text:         t.Selection.Text,
	}
	go c.stream(ctx, s, streamer, req)
}

// finish updates the session and informs the UI, as one step, so that
// "Show last answer" never sees a state in between. It does nothing if the
// session was replaced by a newer one.
func (c *Core) finish(s *session, update func(), notify func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current != s {
		return
	}
	update()
	notify()
}

// stream forwards the events of one request to the UI and the session. Once
// ctx is cancelled (a newer trigger, or the window was closed), nothing more
// is forwarded.
func (c *Core) stream(ctx context.Context, s *session, streamer claude.Streamer, req claude.Request) {
	done := false
	for ev := range streamer.Stream(ctx, req) {
		if ctx.Err() != nil {
			return
		}
		switch ev.Kind {
		case claude.KindDelta:
			var text, notes string
			c.finish(s, func() {
				text, notes = s.split.Write(ev.Text)
				s.answer.WriteString(text)
				s.notes.WriteString(notes)
			}, func() { c.sendPieces(s.id, text, notes) })
		case claude.KindDone:
			done = true
			// Only a finished request is counted: a cancelled or failed one
			// has no final usage.
			var line string
			if c.usage != nil {
				line = c.usage.Record(req.Model, s.action.Name, ev.Usage)
			}
			var rest string
			c.finish(s, func() {
				rest = s.split.Flush()
				s.answer.WriteString(rest)
				s.finished, s.stopReason, s.usageLine = true, ev.StopReason, line
			}, func() {
				c.sendPieces(s.id, rest, "")
				c.ui.Done(s.id, ev.StopReason, line)
			})
		case claude.KindError:
			done = true
			slog.Error("request failed", "model", req.Model, "err", ev.Err)
			msg := claude.Describe(ev.Err, req.Model)
			c.finish(s, func() { s.failure = msg }, func() { c.ui.Fail(s.id, msg) })
		}
	}
	if !done && ctx.Err() == nil {
		msg := "The answer ended unexpectedly."
		c.finish(s, func() { s.failure = msg }, func() { c.ui.Fail(s.id, msg) })
	}
}

// sendPieces forwards non-empty pieces of text and notes to the UI.
func (c *Core) sendPieces(id uint64, text, notes string) {
	if text != "" {
		c.ui.Delta(id, text)
	}
	if notes != "" {
		c.ui.Notes(id, notes)
	}
}

// ShowLast reopens the window with the last session: its answer so far, or
// its error. It reports false if there is no session yet. The window appears
// at the mouse pointer. If the request is still running, new pieces keep
// arriving in the reopened window.
func (c *Core) ShowLast() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.current
	if s == nil {
		return false
	}
	// Under the lock, so no piece of a running answer is lost or doubled.
	c.ui.Begin(c.view(s, true))
	c.sendPieces(s.id, s.answer.String(), s.notes.String())
	switch {
	case s.failure != "":
		c.ui.Fail(s.id, s.failure)
	case s.finished:
		c.ui.Done(s.id, s.stopReason, s.usageLine)
	case s.stopReason == StopCancelled:
		c.ui.Done(s.id, StopCancelled, "")
	}
	return true
}

// Replace pastes the answer of session id into the window the selection came
// from, replacing the selection there. It does nothing for an old session or
// an unfinished answer. Failures are shown in the window.
func (c *Core) Replace(id uint64) {
	c.mu.Lock()
	s := c.current
	if s == nil || s.id != id || !s.finished || s.failure != "" || strings.TrimSpace(s.answer.String()) == "" {
		c.mu.Unlock()
		slog.Warn("replace ignored: no finished answer for this session", "id", id)
		return
	}
	target := s.selection.Target
	text := AdaptText(s.answer.String(), s.selection.Text, platform.DefaultLineEnding())
	var html string
	if s.action.Output == config.OutputMarkdown {
		html = markdownToHTML(s.answer.String())
	}
	c.mu.Unlock()

	if ok, reason := c.platform.CanReplace(target); !ok {
		slog.Warn("replace not possible", "id", id, "app", target.App, "reason", reason)
		c.ui.SetReplaceable(id, false, reason)
		return
	}

	// The window must be out of the way before the target is activated.
	c.ui.Hide()
	ctx, cancel := context.WithTimeout(context.Background(), replaceTimeout)
	defer cancel()
	if err := c.platform.Replace(ctx, target, text, html); err != nil {
		slog.Error("replace failed", "err", err)
		c.ui.Reveal()
		c.ui.Fail(id, "Replace failed: "+err.Error()+".")
		return
	}
	slog.Info("replace done", "id", id, "app", target.App, "html", html != "")
}
