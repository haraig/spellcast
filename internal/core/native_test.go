package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"spellcast/internal/platform"
)

// hotkeyPlatform is a fake platform whose hotkeys the test presses by hand.
type hotkeyPlatform struct {
	*fakePlatform

	mu       sync.Mutex
	press    func(key string)
	captured platform.Selection
	capErr   error
	tips     []string
}

func (h *hotkeyPlatform) WatchHotkeys(ctx context.Context, fn func(string)) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.press = fn
	return nil
}

func (h *hotkeyPlatform) CaptureSelection(context.Context) (platform.Selection, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.captured, h.capErr
}

func (h *hotkeyPlatform) ShowTip(text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tips = append(h.tips, text)
}

func (h *hotkeyPlatform) tipsCopy() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.tips...)
}

func startNative(t *testing.T, h *hotkeyPlatform, has func(string) bool) chan Trigger {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	out := make(chan Trigger, 1)
	src := &NativeSource{Platform: h, HasAction: has}
	go func() { _ = src.Run(ctx, out) }()
	for i := 0; i < 200; i++ {
		h.mu.Lock()
		ready := h.press != nil
		h.mu.Unlock()
		if ready {
			return out
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("hotkeys were not installed")
	return nil
}

func TestNativeSourceSendsTrigger(t *testing.T) {
	h := &hotkeyPlatform{fakePlatform: okPlatform(), captured: platform.Selection{Text: "Hallo", Target: platform.Target{App: "NOTEPAD.EXE", Handle: 7}}}
	out := startNative(t, h, func(k string) bool { return k == "1" })
	h.press("1")
	select {
	case tr := <-out:
		if tr.Action != "1" || tr.Selection.Text != "Hallo" || tr.Selection.Target.Handle != 7 {
			t.Errorf("trigger = %+v", tr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no trigger")
	}
}

func TestNativeSourceTips(t *testing.T) {
	h := &hotkeyPlatform{fakePlatform: okPlatform()}
	out := startNative(t, h, func(k string) bool { return k == "1" })

	h.press("7") // no such action: nothing is copied
	h.capErr = platform.ErrNoSelection
	h.press("1")
	h.capErr = errors.New("clipboard busy")
	h.press("1")

	want := []string{"No action configured for 7", "No text selected", "Spellcast could not copy the selection"}
	got := h.tipsCopy()
	if len(got) != len(want) {
		t.Fatalf("tips = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tip %d = %q, want %q", i, got[i], want[i])
		}
	}
	select {
	case tr := <-out:
		t.Errorf("unexpected trigger %+v", tr)
	default:
	}
}

func TestNativeSourceBusy(t *testing.T) {
	h := &hotkeyPlatform{fakePlatform: okPlatform(), captured: platform.Selection{Text: "x"}}
	out := startNative(t, h, func(string) bool { return true })
	h.press("1") // fills the channel (size 1)
	h.press("1") // full: tip, no block
	if tips := h.tipsCopy(); len(tips) != 1 || tips[0] != "Spellcast is busy" {
		t.Errorf("tips = %q", tips)
	}
	<-out
}

func TestNativeSourceHotkeyError(t *testing.T) {
	h := &failingHotkeys{hotkeyPlatform: &hotkeyPlatform{fakePlatform: okPlatform()}}
	src := &NativeSource{Platform: h, HasAction: func(string) bool { return true }}
	if err := src.Run(context.Background(), make(chan Trigger)); err == nil {
		t.Error("an error from the hook must be returned")
	}
}

type failingHotkeys struct{ *hotkeyPlatform }

func (failingHotkeys) WatchHotkeys(context.Context, func(string)) error {
	return errors.New("hook failed")
}
