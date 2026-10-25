package core

import (
	"context"
	"errors"
	"log/slog"

	"spellcast/internal/platform"
)

// NativeSource is the TriggerSource for the built-in hotkeys: CapsLock + digit
// copies the selection and queues a trigger. It needs no script.
type NativeSource struct {
	Platform  platform.Platform
	HasAction func(key string) bool // asked on every hotkey, so a config reload applies at once
}

var _ TriggerSource = (*NativeSource)(nil)

// Run installs the hotkeys and returns when ctx ends. It returns an error if
// the hotkeys cannot be installed.
func (n *NativeSource) Run(ctx context.Context, out chan<- Trigger) error {
	err := n.Platform.WatchHotkeys(ctx, func(key string) { n.handle(ctx, key, out) })
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (n *NativeSource) handle(ctx context.Context, key string, out chan<- Trigger) {
	if !n.HasAction(key) {
		n.Platform.ShowTip("No action configured for " + key)
		return
	}
	sel, err := n.Platform.CaptureSelection(ctx)
	switch {
	case errors.Is(err, platform.ErrNoSelection):
		n.Platform.ShowTip("No text selected")
		return
	case err != nil:
		// The selected text is private: the error never contains it.
		slog.Error("cannot capture the selection", "err", err)
		n.Platform.ShowTip("Spellcast could not copy the selection")
		return
	}
	select {
	case out <- Trigger{Action: key, Selection: sel}:
		slog.Info("trigger accepted", "action", key, "source", sel.Target.App, "bytes", len(sel.Text))
	default:
		n.Platform.ShowTip("Spellcast is busy")
	}
}
