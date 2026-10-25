package main

import (
	"context"
	"log/slog"

	"spellcast/internal/core"
	"spellcast/internal/platform"
)

// startHotkeys runs the built-in CapsLock + digit hotkeys in the background
// until ctx ends. If the keyboard hook cannot be installed, the user is told.
func startHotkeys(ctx context.Context, store *configStore, plat platform.Platform, notify *notifier, out chan<- core.Trigger) {
	src := &core.NativeSource{
		Platform: plat,
		HasAction: func(key string) bool {
			_, ok := store.Get().Action(key)
			return ok
		},
	}
	go func() {
		if err := src.Run(ctx, out); err != nil {
			slog.Error("hotkeys not available", "err", err)
			notify.Notify("hotkeys-failed", "Spellcast: hotkeys not available", err.Error())
		}
	}()
}
