package core

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"sync"
	"testing"
	"time"

	"spellcast/internal/claude"
	"spellcast/internal/config"
	"spellcast/internal/platform"
)

// fakeUI records what the core tells the UI, as readable strings.
type fakeUI struct {
	mu     sync.Mutex
	log    []string
	views  []View // every Begin
	notify chan struct{}
}

func newFakeUI() *fakeUI { return &fakeUI{notify: make(chan struct{}, 200)} }

func (f *fakeUI) add(s string) {
	f.mu.Lock()
	f.log = append(f.log, s)
	f.mu.Unlock()
	f.notify <- struct{}{}
}

func (f *fakeUI) Begin(v View) {
	f.mu.Lock()
	f.views = append(f.views, v)
	f.mu.Unlock()
	f.add(fmt.Sprintf("begin %d %s/%s/%s replace=%v(%s) pointer=%v", v.ID, v.Action, v.Source, v.Output, v.CanReplace, v.ReplaceReason, v.PlaceAtPointer))
}
func (f *fakeUI) Delta(id uint64, text string) { f.add(fmt.Sprintf("delta %d %s", id, text)) }
func (f *fakeUI) Notes(id uint64, text string) { f.add(fmt.Sprintf("notes %d %s", id, text)) }
func (f *fakeUI) Done(id uint64, stop, usage string) {
	if usage != "" {
		f.add(fmt.Sprintf("done %d %s [%s]", id, stop, usage))
		return
	}
	f.add(fmt.Sprintf("done %d %s", id, stop))
}
func (f *fakeUI) Fail(id uint64, msg string) { f.add(fmt.Sprintf("fail %d %s", id, msg)) }
func (f *fakeUI) SetReplaceable(id uint64, ok bool, reason string) {
	f.add(fmt.Sprintf("replaceable %d %v %s", id, ok, reason))
}
func (f *fakeUI) Hide()   { f.add("hide") }
func (f *fakeUI) Reveal() { f.add("reveal") }

func (f *fakeUI) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.log...)
}

// waitFor blocks until the log contains an entry with the given prefix.
func (f *fakeUI) waitFor(t *testing.T, prefix string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		for _, s := range f.snapshot() {
			if strings.HasPrefix(s, prefix) {
				return
			}
		}
		select {
		case <-f.notify:
		case <-deadline:
			t.Fatalf("timeout waiting for %q; log: %v", prefix, f.snapshot())
		}
	}
}

// fakeStreamer answers from a script.
type fakeStreamer struct {
	mu       sync.Mutex
	requests []claude.Request
	script   func(ctx context.Context, req claude.Request, out chan<- claude.Event)
}

func (f *fakeStreamer) Stream(ctx context.Context, req claude.Request) <-chan claude.Event {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	out := make(chan claude.Event, 8)
	go func() {
		defer close(out)
		f.script(ctx, req, out)
	}()
	return out
}

// fakePlatform records Replace calls.
type fakePlatform struct {
	mu           sync.Mutex
	canReplace   bool
	reason       string
	replaceErr   error
	replaced     []string // "<handle>:<text>"
	replacedHTML []string
	uiAtReplace  func() []string
	uiLog        []string // UI log at the moment of the last Replace
}

func okPlatform() *fakePlatform { return &fakePlatform{canReplace: true} }

func (f *fakePlatform) Anchor(platform.Target) (image.Rectangle, bool) {
	return image.Rectangle{}, false
}
func (f *fakePlatform) PlaceWindow(uintptr, image.Rectangle)                   {}
func (f *fakePlatform) WatchForeground(context.Context, func(platform.Target)) {}
func (f *fakePlatform) SetAlwaysOnTop(uintptr, bool, platform.Target)          {}
func (f *fakePlatform) TargetAlive(platform.Target) bool                       { return true }
func (f *fakePlatform) AutostartEnabled() bool                                 { return false }
func (f *fakePlatform) SetAutostart(bool) error                                { return nil }

func (f *fakePlatform) WatchHotkeys(context.Context, func(string)) error { return nil }
func (f *fakePlatform) CaptureSelection(context.Context) (platform.Selection, error) {
	return platform.Selection{}, platform.ErrNoSelection
}
func (f *fakePlatform) ShowTip(string) {}

func (f *fakePlatform) CanReplace(platform.Target) (bool, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.canReplace, f.reason
}

func (f *fakePlatform) Replace(ctx context.Context, t platform.Target, text, html string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replaced = append(f.replaced, fmt.Sprintf("%d:%s", t.Handle, text))
	f.replacedHTML = append(f.replacedHTML, html)
	if f.uiAtReplace != nil {
		f.uiLog = f.uiAtReplace()
	}
	return f.replaceErr
}

func (f *fakePlatform) replacedCopy() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.replaced...)
}

func lookup(key string) (config.Action, bool) {
	switch key {
	case "1":
		return config.Action{Name: "Translate", Prompt: "Translate.", Model: "m1", MaxTokens: 100, Output: config.OutputPlain}, true
	case "3":
		return config.Action{Name: "Explain", Prompt: "Explain.", Model: "m3", MaxTokens: 300, Output: config.OutputMarkdown}, true
	}
	return config.Action{}, false
}

func trigger(action, text, app string) Trigger {
	return Trigger{Action: action, Selection: platform.Selection{Text: text, Target: platform.Target{App: app, Handle: 42}}}
}

// start runs a core on a trigger channel and returns it.
func start(t *testing.T, ui UI, client ClientFunc, p ...platform.Platform) (*Core, chan<- Trigger) {
	t.Helper()
	var plat platform.Platform = okPlatform()
	if len(p) > 0 {
		plat = p[0]
	}
	in := make(chan Trigger, 4)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c := New(ui, lookup, client, plat)
	go c.Run(ctx, in)
	return c, in
}

func TestStreamsAnswerToUI(t *testing.T) {
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "Hel"}
		out <- claude.Event{Kind: claude.KindDelta, Text: "lo"}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	ui := newFakeUI()
	_, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })

	in <- trigger("9", "ignored", "X.EXE") // unknown action: nothing happens
	in <- trigger("3", "Grüße 👋", "NOTEPAD.EXE")
	ui.waitFor(t, "done")

	want := []string{"begin 1 Explain/NOTEPAD.EXE/markdown replace=true() pointer=false", "delta 1 Hel", "delta 1 lo", "done 1 end_turn"}
	if got := ui.snapshot(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("log = %v, want %v", got, want)
	}
	req := fs.requests[0]
	if req.Model != "m3" || req.MaxTokens != 300 || req.Prompt != "Explain." || req.Output != config.OutputMarkdown || req.Text != "Grüße 👋" {
		t.Errorf("request = %+v", req)
	}
	if len(fs.requests) != 1 {
		t.Errorf("%d requests, want 1", len(fs.requests))
	}
}

func TestNewTriggerCancelsRunningRequest(t *testing.T) {
	first := make(chan struct{}) // closed when the first request sees its cancellation
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		if req.Text == "one" {
			out <- claude.Event{Kind: claude.KindDelta, Text: "old"}
			<-ctx.Done()
			close(first)
			// A late event from the cancelled request must not reach the UI.
			select {
			case out <- claude.Event{Kind: claude.KindDelta, Text: "LATE"}:
			default:
			}
			return
		}
		out <- claude.Event{Kind: claude.KindDelta, Text: "new"}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	ui := newFakeUI()
	_, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })

	in <- trigger("1", "one", "A.EXE")
	ui.waitFor(t, "delta 1 old")
	in <- trigger("1", "two", "B.EXE")
	ui.waitFor(t, "done 2")

	select {
	case <-first:
	case <-time.After(3 * time.Second):
		t.Fatal("the first request was not cancelled")
	}
	time.Sleep(50 * time.Millisecond)
	for _, s := range ui.snapshot() {
		if strings.Contains(s, "LATE") || strings.HasPrefix(s, "done 1") || strings.HasPrefix(s, "fail 1") {
			t.Errorf("cancelled request reached the UI: %q (log %v)", s, ui.snapshot())
		}
	}
}

func TestCancelStopsRequest(t *testing.T) {
	cancelled := make(chan struct{})
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "x"}
		<-ctx.Done()
		close(cancelled)
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })

	in <- trigger("1", "text", "A.EXE")
	ui.waitFor(t, "delta 1 x")
	c.Cancel() // the window was closed
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("Cancel did not stop the request")
	}
	time.Sleep(50 * time.Millisecond)
	for _, s := range ui.snapshot() {
		if strings.HasPrefix(s, "fail") || strings.HasPrefix(s, "done") {
			t.Errorf("unexpected %q after Cancel", s)
		}
	}
}

func TestNoAPIKey(t *testing.T) {
	ui := newFakeUI()
	_, in := start(t, ui, func() (claude.Streamer, error) { return nil, claude.ErrNoAPIKey })
	in <- trigger("1", "text", "A.EXE")
	ui.waitFor(t, "fail")
	if got := ui.snapshot()[1]; !strings.Contains(got, "No API key") || !strings.Contains(got, "Set API key") {
		t.Errorf("message = %q", got)
	}
}

func TestKeyStoreFailure(t *testing.T) {
	ui := newFakeUI()
	_, in := start(t, ui, func() (claude.Streamer, error) { return nil, errors.New("store locked") })
	in <- trigger("1", "text", "A.EXE")
	ui.waitFor(t, "fail")
	if got := ui.snapshot()[1]; !strings.Contains(got, "store locked") {
		t.Errorf("message = %q", got)
	}
}

func TestRequestError(t *testing.T) {
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindError, Err: errors.New("boom")}
	}}
	ui := newFakeUI()
	_, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })
	in <- trigger("1", "text", "A.EXE")
	ui.waitFor(t, "fail")
	if got := ui.snapshot()[1]; !strings.HasPrefix(got, "fail 1 ") || !strings.Contains(got, "boom") {
		t.Errorf("message = %q", got)
	}
}

func TestStreamEndingWithoutDone(t *testing.T) {
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "partial"}
	}}
	ui := newFakeUI()
	_, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })
	in <- trigger("1", "text", "A.EXE")
	ui.waitFor(t, "fail")
}

func TestRunStopsOnContextAndClosedChannel(t *testing.T) {
	c := New(newFakeUI(), lookup, nil, okPlatform())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.Run(ctx, make(chan Trigger)); close(done) }()
	cancel()
	waitDone(t, done)

	in := make(chan Trigger)
	done = make(chan struct{})
	go func() { c.Run(context.Background(), in); close(done) }()
	close(in)
	waitDone(t, done)
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
}

// finishedSession runs one trigger to the end.
func finishedSession(t *testing.T, p *fakePlatform, action, selection, answer string) (*Core, *fakeUI, chan<- Trigger) {
	t.Helper()
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: answer}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, p)
	in <- trigger(action, selection, "NOTEPAD.EXE")
	ui.waitFor(t, "done")
	return c, ui, in
}

func TestReplaceAdaptsTextAndHidesWindowFirst(t *testing.T) {
	p := okPlatform()
	var ui *fakeUI
	p.uiAtReplace = func() []string { return ui.snapshot() }
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "\nPrint it\nagain \n"}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	ui = newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, p)
	in <- trigger("1", "print it\r\nagain\r\n", "CODE.EXE") // CRLF, trailing line break
	ui.waitFor(t, "done")

	c.Replace(1)

	if got := p.replacedCopy(); len(got) != 1 || got[0] != "42:Print it\r\nagain\r\n" {
		t.Fatalf("replaced = %q", got)
	}
	if last := p.uiLog[len(p.uiLog)-1]; last != "hide" {
		t.Errorf("the window must be hidden before pasting; UI log at that moment: %v", p.uiLog)
	}
	for _, s := range ui.snapshot() {
		if s == "reveal" || strings.HasPrefix(s, "fail") {
			t.Errorf("unexpected %q after a successful replace", s)
		}
	}
}

func TestSessionNotes(t *testing.T) {
	pieces := []string{"Hello wor", "ld\n-", "-", "-\n", "Alt", ": Hi\n"}
	var req claude.Request
	fs := &fakeStreamer{script: func(ctx context.Context, r claude.Request, out chan<- claude.Event) {
		req = r
		for _, p := range pieces {
			out <- claude.Event{Kind: claude.KindDelta, Text: p}
		}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	p := okPlatform()
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, p)
	c.action = func(string) (config.Action, bool) {
		return config.Action{Name: "A", Prompt: "p", Notes: true}, true
	}
	in <- trigger("1", "orig", "NOTEPAD.EXE")
	ui.waitFor(t, "done")

	if !req.Notes {
		t.Error("request must carry Notes")
	}
	var text, notes string
	for _, e := range ui.snapshot() {
		switch {
		case strings.HasPrefix(e, "delta 1 "):
			text += strings.TrimPrefix(e, "delta 1 ")
		case strings.HasPrefix(e, "notes 1 "):
			notes += strings.TrimPrefix(e, "notes 1 ")
		}
	}
	if text != "Hello world\n" || notes != "Alt: Hi\n" {
		t.Errorf("text = %q, notes = %q", text, notes)
	}

	c.Replace(1)
	if got := p.replacedCopy(); len(got) != 1 || got[0] != "42:Hello world" {
		t.Errorf("Replace must paste only the text part: %q", got)
	}
}

func TestShowLastReplaysNotes(t *testing.T) {
	fs := &fakeStreamer{script: func(ctx context.Context, r claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "A\n---\nB"}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, okPlatform())
	c.action = func(string) (config.Action, bool) {
		return config.Action{Name: "A", Prompt: "p", Notes: true}, true
	}
	in <- trigger("1", "x", "NOTEPAD.EXE")
	ui.waitFor(t, "done")
	c.ShowLast()
	log := strings.Join(ui.snapshot(), "|")
	if strings.Count(log, "delta 1 A\n") != 2 || strings.Count(log, "notes 1 B") != 2 {
		t.Errorf("log = %q", log)
	}
}

func TestRequestCarriesInstructions(t *testing.T) {
	var got claude.Request
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		got = req
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, okPlatform())
	_ = c
	c.action = func(string) (config.Action, bool) {
		return config.Action{Name: "A", Prompt: "p", Instructions: "Be brief."}, true
	}
	in <- trigger("1", "x", "NOTEPAD.EXE")
	ui.waitFor(t, "done")
	if got.Instructions != "Be brief." {
		t.Errorf("request instructions = %q", got.Instructions)
	}
}

func TestReplaceSendsHTMLOnlyForMarkdownActions(t *testing.T) {
	p := okPlatform()
	c, _, _ := finishedSession(t, p, "3", "text", "**fett**")
	c.Replace(1)
	p2 := okPlatform()
	c2, _, _ := finishedSession(t, p2, "1", "text", "**fett**")
	c2.Replace(1)

	if len(p.replacedHTML) != 1 || !strings.Contains(p.replacedHTML[0], "<strong>fett</strong>") {
		t.Errorf("markdown action: html = %q", p.replacedHTML)
	}
	if len(p.replaced) != 1 || p.replaced[0] != "42:**fett**" {
		t.Errorf("markdown action: plain text must stay the source: %q", p.replaced)
	}
	if len(p2.replacedHTML) != 1 || p2.replacedHTML[0] != "" {
		t.Errorf("plain action: html = %q, want empty", p2.replacedHTML)
	}
}

func TestReplaceFailureShowsWindowAndMessage(t *testing.T) {
	p := okPlatform()
	p.replaceErr = platform.ErrActivateFailed
	c, ui, _ := finishedSession(t, p, "1", "Hallo", "Hello")
	c.Replace(1)
	ui.waitFor(t, "fail")
	log := ui.snapshot()
	if got := log[len(log)-2]; got != "reveal" {
		t.Errorf("expected reveal before fail, log: %v", log)
	}
	if got := log[len(log)-1]; !strings.Contains(got, "Replace failed") || !strings.Contains(got, "could not switch back") {
		t.Errorf("message = %q", got)
	}
}

func TestReplaceNotPossible(t *testing.T) {
	p := okPlatform()
	c, ui, _ := finishedSession(t, p, "1", "Hallo", "Hello")
	p.mu.Lock()
	p.canReplace, p.reason = false, "The source window was closed."
	p.mu.Unlock()
	c.Replace(1)
	ui.waitFor(t, "replaceable")
	if got := p.replacedCopy(); len(got) != 0 {
		t.Errorf("Replace must not run when CanReplace says no: %v", got)
	}
	for _, s := range ui.snapshot() {
		if s == "hide" {
			t.Error("the window must stay open when Replace is not possible")
		}
	}
}

func TestReplaceIgnoredForStaleOrUnfinishedSession(t *testing.T) {
	p := okPlatform()
	block := make(chan struct{})
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "partial"}
		select {
		case <-block:
		case <-ctx.Done():
		}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, p)
	t.Cleanup(func() { close(block) })
	in <- trigger("1", "x", "A.EXE")
	ui.waitFor(t, "delta 1 partial")

	c.Replace(1) // not finished
	c.Replace(7) // unknown session
	if got := p.replacedCopy(); len(got) != 0 {
		t.Errorf("replaced = %v", got)
	}
}

func TestReplaceIgnoredWithoutAnswer(t *testing.T) {
	p := okPlatform()
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDone, StopReason: "refusal"}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, p)
	in <- trigger("1", "x", "A.EXE")
	ui.waitFor(t, "done")
	c.Replace(1)
	if got := p.replacedCopy(); len(got) != 0 {
		t.Errorf("replaced = %v", got)
	}
}

func TestBeginCarriesReplaceState(t *testing.T) {
	p := &fakePlatform{canReplace: false, reason: "Replace is disabled in WindowsTerminal.exe."}
	_, ui, _ := finishedSession(t, p, "1", "Hallo", "Hello")
	if got := ui.snapshot()[0]; !strings.Contains(got, "replace=false(Replace is disabled in WindowsTerminal.exe.)") {
		t.Errorf("begin = %q", got)
	}
}

func TestShowLast(t *testing.T) {
	p := okPlatform()
	c, ui, _ := finishedSession(t, p, "3", "Hallo", "The answer")
	before := len(ui.snapshot())

	if !c.ShowLast() {
		t.Fatal("ShowLast = false, want true")
	}
	got := ui.snapshot()[before:]
	want := []string{"begin 1 Explain/NOTEPAD.EXE/markdown replace=true() pointer=true", "delta 1 The answer", "done 1 end_turn"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("replayed = %v, want %v", got, want)
	}

	// Replace works from the reopened window.
	c.Replace(1)
	if got := p.replacedCopy(); len(got) != 1 || got[0] != "42:The answer" {
		t.Errorf("replaced = %q", got)
	}
}

func TestShowLastWithoutSession(t *testing.T) {
	c := New(newFakeUI(), lookup, nil, okPlatform())
	if c.ShowLast() {
		t.Error("ShowLast = true without a session")
	}
}

func TestShowLastAfterFailure(t *testing.T) {
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return nil, claude.ErrNoAPIKey })
	in <- trigger("1", "x", "A.EXE")
	ui.waitFor(t, "fail")
	before := len(ui.snapshot())
	c.ShowLast()
	got := ui.snapshot()[before:]
	if len(got) != 2 || !strings.HasPrefix(got[0], "begin 1") || !strings.Contains(got[1], "No API key") {
		t.Errorf("replayed = %v", got)
	}
}

func TestShowLastWhileRunning(t *testing.T) {
	release := make(chan struct{})
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "one "}
		select {
		case <-release:
			out <- claude.Event{Kind: claude.KindDelta, Text: "two"}
			out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn"}
		case <-ctx.Done():
		}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })
	in <- trigger("1", "x", "A.EXE")
	ui.waitFor(t, "delta 1 one")

	// The partial answer is replayed; later pieces still arrive.
	before := len(ui.snapshot())
	c.ShowLast()
	close(release)
	ui.waitFor(t, "done 1")
	got := strings.Join(ui.snapshot()[before:], "|")
	for _, part := range []string{"begin 1", "delta 1 one ", "delta 1 two", "done 1 end_turn"} {
		if !strings.Contains(got, part) {
			t.Errorf("missing %q in %q", part, got)
		}
	}
}

func TestShowLastAfterCancel(t *testing.T) {
	p := okPlatform()
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "half"}
		<-ctx.Done()
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil }, p)
	in <- trigger("1", "x", "A.EXE")
	ui.waitFor(t, "delta 1 half")
	c.Cancel()

	before := len(ui.snapshot())
	c.ShowLast()
	got := strings.Join(ui.snapshot()[before:], "|")
	if !strings.Contains(got, "delta 1 half") || !strings.Contains(got, "done 1 cancelled") {
		t.Errorf("replayed = %q", got)
	}
	// A cancelled, incomplete answer cannot be pasted.
	c.Replace(1)
	if got := p.replacedCopy(); len(got) != 0 {
		t.Errorf("a cancelled answer was pasted: %v", got)
	}
}

func TestNewTriggerReplacesLastSession(t *testing.T) {
	c, ui, in := finishedSession(t, okPlatform(), "1", "eins", "one")
	in <- trigger("1", "zwei", "B.EXE")
	ui.waitFor(t, "done 2")
	before := len(ui.snapshot())
	c.ShowLast()
	if got := ui.snapshot()[before]; !strings.HasPrefix(got, "begin 2 ") {
		t.Errorf("ShowLast replayed %q, want session 2", got)
	}
}

// fakeUsage records what the core counts.
type fakeUsage struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeUsage) Record(model, action string, u claude.Usage) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("%s/%s/%d/%d", model, action, u.InputTokens, u.OutputTokens))
	return fmt.Sprintf("%d in · %d out", u.InputTokens, u.OutputTokens)
}

func (f *fakeUsage) callsCopy() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func TestFinishedRequestIsCountedAndShown(t *testing.T) {
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "Hallo"}
		out <- claude.Event{Kind: claude.KindDone, StopReason: "end_turn", Usage: claude.Usage{InputTokens: 312, OutputTokens: 187}}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })
	rec := &fakeUsage{}
	c.SetUsage(rec)
	in <- trigger("1", "Hello", "NOTEPAD.EXE")
	ui.waitFor(t, "done 1")

	if got := rec.callsCopy(); len(got) != 1 || got[0] != "m1/Translate/312/187" {
		t.Errorf("recorded = %q", got)
	}
	if got := strings.Join(ui.snapshot(), "|"); !strings.Contains(got, "done 1 end_turn [312 in · 187 out]") {
		t.Errorf("log = %q", got)
	}

	// Show last answer replays the line, without counting again.
	c.ShowLast()
	if got := strings.Join(ui.snapshot(), "|"); strings.Count(got, "[312 in · 187 out]") != 2 {
		t.Errorf("log after ShowLast = %q", got)
	}
	if len(rec.callsCopy()) != 1 {
		t.Error("Show last answer must not count a request twice")
	}
}

func TestCancelledAndFailedRequestsAreNotCounted(t *testing.T) {
	fs := &fakeStreamer{script: func(ctx context.Context, req claude.Request, out chan<- claude.Event) {
		out <- claude.Event{Kind: claude.KindDelta, Text: "x"}
		out <- claude.Event{Kind: claude.KindError, Err: errors.New("boom")}
	}}
	ui := newFakeUI()
	c, in := start(t, ui, func() (claude.Streamer, error) { return fs, nil })
	rec := &fakeUsage{}
	c.SetUsage(rec)
	in <- trigger("1", "Hello", "NOTEPAD.EXE")
	ui.waitFor(t, "fail 1")
	if got := rec.callsCopy(); len(got) != 0 {
		t.Errorf("a failed request must not be counted: %q", got)
	}
}

// "Show changes" compares the answer with the selection, so the window gets
// the selected text, also when the window is reopened with "Show last answer".
func TestViewCarriesTheSelectedText(t *testing.T) {
	c, ui, _ := finishedSession(t, okPlatform(), "3", "Hallo Welt", "The answer")
	if !c.ShowLast() {
		t.Fatal("ShowLast = false, want true")
	}
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if len(ui.views) != 2 {
		t.Fatalf("got %d views, want 2", len(ui.views))
	}
	for i, v := range ui.views {
		if v.Original != "Hallo Welt" {
			t.Errorf("view %d: Original = %q, want %q", i, v.Original, "Hallo Welt")
		}
	}
}
