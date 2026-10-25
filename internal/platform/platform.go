// Package platform holds the OS-specific parts behind OS-neutral types.
// Windows is implemented in platform_windows.go and the files next to it;
// other systems get a stub until the macOS implementation exists.
package platform

import (
	"context"
	"errors"
	"image"
	"runtime"
)

// Target identifies the window or app that the selection came from.
type Target struct {
	App    string  // "OUTLOOK.EXE" or "com.microsoft.Outlook"
	Handle uintptr // HWND on Windows, PID on macOS
}

// Selection is the captured text and where it came from.
type Selection struct {
	Text   string
	Target Target
}

// Errors of Replace. The messages are shown to the user.
var (
	ErrTargetGone     = errors.New("the source window was closed")
	ErrActivateFailed = errors.New("could not switch back to the source window")
	ErrInputBlocked   = errors.New("Windows blocked the paste (the source app may run as administrator)") //nolint:staticcheck // ST1005: "Windows" is a proper noun; the message is shown to users as is.
	ErrUnsupported    = errors.New("not supported on this system")
	ErrNoSelection    = errors.New("no text selected")
)

// Platform is the OS-specific functionality.
type Platform interface {
	// Anchor returns where the result window should appear, in physical
	// screen pixels: the text cursor (caret) rectangle if the target reports
	// one, else a 1×1 rectangle at the mouse pointer.
	Anchor(t Target) (r image.Rectangle, fromCaret bool)

	// PlaceWindow moves the native window (HWND on Windows) next to anchor,
	// fully inside the work area of the anchor's monitor.
	PlaceWindow(window uintptr, anchor image.Rectangle)

	// Replace pastes text into the target's current selection or cursor
	// position: it puts the text on the clipboard, activates the target,
	// sends the paste shortcut and restores the previous clipboard shortly
	// afterwards. It cannot tell whether the target accepted the paste
	// (read-only views ignore it).
	//
	// If html is not empty it is offered as rich text next to the plain text
	// (Outlook and Word prefer it).
	Replace(ctx context.Context, t Target, text, html string) error

	// CanReplace tells whether Replace is possible. If not, reason explains
	// why in one sentence (shown as a tooltip).
	CanReplace(t Target) (ok bool, reason string)

	// SetAlwaysOnTop puts the native window on top of all windows (onTop) or
	// into normal z-order, directly behind the window `behind` if it is given
	// and exists. It never takes the focus.
	SetAlwaysOnTop(window uintptr, onTop bool, behind Target)

	// TargetAlive tells whether the target window still exists.
	TargetAlive(t Target) bool

	// WatchForeground reports every change of the foreground window until
	// ctx ends. The Target carries the top-level window handle.
	WatchForeground(ctx context.Context, fn func(Target))

	// CaptureSelection copies the selected text of the foreground window and
	// restores the clipboard. It returns ErrNoSelection if there is none.
	CaptureSelection(ctx context.Context) (Selection, error)

	// WatchHotkeys reports CapsLock + digit ("1".."9") until ctx ends. The
	// keys are swallowed, so they are not typed. It returns once the hook is
	// installed. fn is called from another goroutine, in order.
	WatchHotkeys(ctx context.Context, fn func(key string)) error

	// ShowTip shows a short message next to the mouse pointer. It does not
	// take the focus and returns at once.
	ShowTip(text string)

	// AutostartEnabled and SetAutostart manage "start with the OS".
	AutostartEnabled() bool
	SetAutostart(enabled bool) error
}

// DefaultLineEnding is the line ending used when the original selection has
// no line break to copy the style from.
func DefaultLineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
