//go:build windows

package platform

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	vkInsert = 0x2D
	vkC      = 0x43

	// How long to wait for the clipboard to change after a copy shortcut.
	copyTimeout = 500 * time.Millisecond
	// The clipboard must stay unchanged this long before it is read, because
	// some apps clear it first and then set the text.
	copySettle = 30 * time.Millisecond
)

// processName returns the executable name (e.g. "notepad.exe") of the process
// that owns hwnd, or "" if it cannot be read (for example an elevated process).
func processName(hwnd windows.HWND) string {
	var pid uint32
	procGetWindowThreadProcessID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer func() { _ = windows.CloseHandle(h) }()
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buf[:size]))
}

// CaptureSelection copies the selected text of the foreground window with
// Ctrl+Insert (Ctrl+C where that does not work, except in terminals, where
// Ctrl+C would stop the running program) and restores the clipboard.
func (p windowsPlatform) CaptureSelection(ctx context.Context) (Selection, error) {
	// A key still held from the hotkey would turn the copy into another shortcut.
	waitModifiersReleased(ctx, time.Second)

	fg, _, _ := procGetForegroundWindow.Call()
	if fg == 0 {
		return Selection{}, ErrNoSelection
	}
	hwnd := rootWindow(windows.HWND(fg))
	target := Target{App: processName(hwnd), Handle: uintptr(hwnd)}

	snap, snapErr := readSnapshot()
	if snapErr != nil {
		slog.Warn("cannot back up the clipboard", "err", snapErr)
	}
	seq := clipboardSequence()

	text, ok := copyWith(ctx, seq, func() error { return sendCtrlKey(vkInsert, true) })
	if !ok && ctx.Err() == nil && !p.denied(target.App) {
		// Some apps ignore Ctrl+Insert (for example Google Docs).
		text, ok = copyWith(ctx, seq, func() error { return sendCtrlKey(vkC, false) })
	}

	// Put the old clipboard back, but only if the copy changed it.
	if snapErr == nil && clipboardSequence() != seq {
		if err := restoreSnapshot(snap); err != nil {
			slog.Warn("cannot restore the clipboard", "err", err)
		}
	}

	if !ok || strings.TrimSpace(text) == "" {
		return Selection{}, ErrNoSelection
	}
	return Selection{Text: text, Target: target}, nil
}

// copyWith sends a copy shortcut and waits for new text on the clipboard.
func copyWith(ctx context.Context, seqBefore uint32, send func() error) (string, bool) {
	if err := send(); err != nil {
		slog.Warn("cannot send the copy shortcut", "err", err)
		return "", false
	}
	deadline := time.Now().Add(copyTimeout)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		if now := clipboardSequence(); now != seqBefore {
			time.Sleep(copySettle)
			if clipboardSequence() != now {
				continue // still changing
			}
			text, ok := clipboardText()
			return text, ok
		}
		time.Sleep(10 * time.Millisecond)
	}
	return "", false
}
