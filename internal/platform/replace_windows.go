//go:build windows

package platform

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procIsWindow              = user32.NewProc("IsWindow")
	procIsIconic              = user32.NewProc("IsIconic")
	procShowWindow            = user32.NewProc("ShowWindow")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procGetAncestor           = user32.NewProc("GetAncestor")
	procAttachThreadInput     = user32.NewProc("AttachThreadInput")
	procGetAsyncKeyState      = user32.NewProc("GetAsyncKeyState")
	procSendInput             = user32.NewProc("SendInput")
	procGetCurrentThreadIDLib = kernel32.NewProc("GetCurrentThreadId")
	procSetWindowPosZ         = user32.NewProc("SetWindowPos")
)

const (
	swRestore = 9

	hwndTopmost             = ^uintptr(0) // (HWND)-1
	hwndNoTopmost           = ^uintptr(1) // (HWND)-2
	swpNoMoveSizeNoActivate = 0x0001 | 0x0002 | 0x0010
	gaRoot                  = 2

	inputKeyboard    = 1
	keyeventKeyUp    = 0x0002
	keyeventExtended = 0x0001
	vkShift          = 0x10
	vkControl        = 0x11
	vkMenu           = 0x12 // Alt
	vkLWin, vkRWin   = 0x5B, 0x5C
	vkV              = 0x56
	activateTimeout  = 500 * time.Millisecond
	restoreDelay     = 500 * time.Millisecond // some apps read the clipboard lazily
)

// keyInput is a Win32 INPUT of type keyboard (40 bytes on 64-bit Windows).
type keyInput struct {
	Type      uint32
	_         uint32
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	_         uint32
	ExtraInfo uintptr
	_         [8]byte
}

func isWindow(hwnd windows.HWND) bool {
	ret, _, _ := procIsWindow.Call(uintptr(hwnd))
	return ret != 0
}

func rootWindow(hwnd windows.HWND) windows.HWND {
	root, _, _ := procGetAncestor.Call(uintptr(hwnd), gaRoot)
	if root == 0 {
		return hwnd
	}
	return windows.HWND(root)
}

// SetAlwaysOnTop keeps window above everything (onTop), or puts it directly
// behind `behind`. SWP_NOACTIVATE: the focus stays where it is. (Wails'
// own SetAlwaysOnTop would activate the window and steal the focus.)
func (windowsPlatform) SetAlwaysOnTop(window uintptr, onTop bool, behind Target) {
	insertAfter := hwndNoTopmost
	switch {
	case onTop:
		insertAfter = hwndTopmost
	case behind.Handle != 0 && isWindow(windows.HWND(behind.Handle)):
		insertAfter = behind.Handle // behind a non-topmost window: loses "topmost"
	}
	procSetWindowPosZ.Call(window, insertAfter, 0, 0, 0, 0, swpNoMoveSizeNoActivate)
}

// TargetAlive reports whether the target window still exists.
func (windowsPlatform) TargetAlive(t Target) bool {
	return t.Handle != 0 && isWindow(windows.HWND(t.Handle))
}

// CanReplace checks the target window and the deny list (terminals).
func (p windowsPlatform) CanReplace(t Target) (bool, string) {
	if t.Handle == 0 {
		return false, "The source window is unknown."
	}
	if !isWindow(windows.HWND(t.Handle)) {
		return false, "The source window was closed."
	}
	if p.denied(t.App) {
		return false, fmt.Sprintf("Replace is disabled in %s.", t.App)
	}
	return true, ""
}

func (p windowsPlatform) denied(app string) bool {
	if app == "" || p.denyList == nil {
		return false
	}
	for _, d := range p.denyList() {
		if strings.EqualFold(d, app) {
			return true
		}
	}
	return false
}

// Replace puts text on the clipboard, activates the target and pastes with
// Ctrl+V. The old clipboard comes back after restoreDelay, unless someone
// else changed the clipboard in the meantime.
func (p windowsPlatform) Replace(ctx context.Context, t Target, text, html string) error {
	hwnd := windows.HWND(t.Handle)
	if hwnd == 0 || !isWindow(hwnd) {
		return ErrTargetGone
	}

	// Best effort: if the old clipboard cannot be read, we paste anyway and
	// then leave our text in place rather than clearing the clipboard.
	snap, snapErr := readSnapshot()
	if snapErr != nil {
		slog.Warn("cannot back up the clipboard", "err", snapErr)
	} else if snap.Incomplete {
		slog.Warn("clipboard backup is incomplete (a format was too big or unreadable)")
	}

	seq, err := writeContent(text, html)
	if err != nil {
		return fmt.Errorf("cannot write the clipboard: %w", err)
	}

	restore := func() {
		if snapErr != nil {
			return
		}
		if clipboardSequence() != seq {
			slog.Info("clipboard changed by someone else, not restoring")
			return
		}
		if err := restoreSnapshot(snap); err != nil {
			slog.Warn("cannot restore the clipboard", "err", err)
		}
	}

	pasteErr := pasteInto(ctx, hwnd)
	if pasteErr != nil {
		restore() // nothing was pasted: put the old clipboard back at once
		return pasteErr
	}
	time.AfterFunc(restoreDelay, restore)
	return nil
}

// pasteInto activates the window and sends Ctrl+V.
func pasteInto(ctx context.Context, hwnd windows.HWND) error {
	waitModifiersReleased(ctx, time.Second)
	if err := activate(ctx, hwnd); err != nil {
		return err
	}
	return sendCtrlV()
}

// waitModifiersReleased waits until Ctrl, Shift, Alt and Win are up, so the
// paste is not turned into e.g. Ctrl+Shift+V by a key the user still holds
// from the Replace shortcut. It gives up silently after max.
func waitModifiersReleased(ctx context.Context, max time.Duration) {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		held := false
		for _, vk := range []uintptr{vkShift, vkControl, vkMenu, vkLWin, vkRWin} {
			if state, _, _ := procGetAsyncKeyState.Call(vk); state&0x8000 != 0 {
				held = true
			}
		}
		if !held {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func isForeground(hwnd windows.HWND) bool {
	fg, _, _ := procGetForegroundWindow.Call()
	return rootWindow(windows.HWND(fg)) == rootWindow(hwnd)
}

// activate brings the window to the foreground and waits until it is there.
func activate(ctx context.Context, hwnd windows.HWND) error {
	if ret, _, _ := procIsIconic.Call(uintptr(hwnd)); ret != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	procSetForegroundWindow.Call(uintptr(hwnd))

	deadline := time.Now().Add(activateTimeout)
	triedAttach := false
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isForeground(hwnd) {
			return nil
		}
		if !triedAttach && time.Until(deadline) < activateTimeout*3/4 {
			// SetForegroundWindow was refused: borrow the input queue of the
			// current foreground thread for a moment.
			triedAttach = true
			attachAndActivate(hwnd)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if isForeground(hwnd) {
		return nil
	}
	return ErrActivateFailed
}

func attachAndActivate(hwnd windows.HWND) {
	fg, _, _ := procGetForegroundWindow.Call()
	if fg == 0 {
		return
	}
	fgThread, _, _ := procGetWindowThreadProcessID.Call(fg, 0)
	me, _, _ := procGetCurrentThreadIDLib.Call()
	if fgThread == 0 || fgThread == me {
		return
	}
	procAttachThreadInput.Call(me, fgThread, 1)
	procSetForegroundWindow.Call(uintptr(hwnd))
	procAttachThreadInput.Call(me, fgThread, 0)
}

// injectMarker is put into the extra info of every key we send, so our own
// keyboard hook can tell these events from the user's (and from other
// programs' input, which it handles like real keys).
const injectMarker = 0x5370656C // "Spel"

func newKey(vk uint16, up, extended bool) keyInput {
	in := keyInput{Type: inputKeyboard, Vk: vk, ExtraInfo: injectMarker}
	if up {
		in.Flags |= keyeventKeyUp
	}
	if extended {
		in.Flags |= keyeventExtended
	}
	return in
}

func sendKeys(inputs []keyInput) error {
	n, _, _ := procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	if int(n) != len(inputs) {
		return ErrInputBlocked
	}
	return nil
}

// sendCtrlKey sends Ctrl down, key down, key up, Ctrl up. extended marks keys
// of the extended block (Insert), which Windows tells apart from the numpad.
func sendCtrlKey(vk uint16, extended bool) error {
	return sendKeys([]keyInput{
		newKey(vkControl, false, false), newKey(vk, false, extended),
		newKey(vk, true, extended), newKey(vkControl, true, false),
	})
}

// sendCtrlV pastes.
func sendCtrlV() error { return sendCtrlKey(vkV, false) }
