// Package core connects triggers, actions, Claude and the UI (OS-neutral).
package core

import (
	"context"

	"spellcast/internal/platform"
)

// Trigger asks to run the action with the given key on a selection.
type Trigger struct {
	Action    string
	Selection platform.Selection
}

// TriggerSource produces triggers until ctx ends: the native hotkeys (Windows
// today, macOS later).
type TriggerSource interface {
	Run(ctx context.Context, out chan<- Trigger) error
}
