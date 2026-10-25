//go:build !windows

package platform

import (
	"context"
	"image"
)

// New returns a stub until the macOS implementation exists.
// denyList returns the apps where Replace is disabled.
func New(denyList func() []string) Platform { return stub{} }

type stub struct{}

func (stub) Anchor(Target) (image.Rectangle, bool) { return image.Rectangle{}, false }

func (stub) PlaceWindow(uintptr, image.Rectangle) {}

func (stub) Replace(context.Context, Target, string, string) error { return ErrUnsupported }

func (stub) CanReplace(Target) (bool, string) { return false, ErrUnsupported.Error() }

func (stub) SetAlwaysOnTop(uintptr, bool, Target) {}

func (stub) TargetAlive(Target) bool { return false }

func (stub) WatchForeground(context.Context, func(Target)) {}

func (stub) CaptureSelection(context.Context) (Selection, error) {
	return Selection{}, ErrUnsupported
}

func (stub) WatchHotkeys(context.Context, func(string)) error { return ErrUnsupported }

func (stub) ShowTip(string) {}

func (stub) AutostartEnabled() bool { return false }

func (stub) SetAutostart(bool) error { return ErrUnsupported }

// RemoveNotificationRegistration has nothing to remove on this system.
func RemoveNotificationRegistration(string) error { return nil }
