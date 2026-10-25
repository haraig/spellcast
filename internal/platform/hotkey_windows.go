//go:build windows

package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procGetKeyState         = user32.NewProc("GetKeyState")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
)

const (
	whKeyboardLL = 13

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105
)

// kbdLLHookStruct is KBDLLHOOKSTRUCT.
type kbdLLHookStruct struct {
	VkCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

// WatchHotkeys installs the keyboard hook for CapsLock + digit and calls fn
// with "1".."9" for every hotkey, in order, from its own goroutine until ctx
// ends. It returns once the hook is installed (or failed).
//
// The hook callback is kept minimal: it runs the pure filter and queues the
// digit. Windows silently removes a low-level hook whose callback is too
// slow, so nothing else happens there. The hook sees all keys of the
// session; the filter looks at CapsLock and the digits only and stores nothing.
func (windowsPlatform) WatchHotkeys(ctx context.Context, fn func(key string)) error {
	keys := make(chan string, 8)
	ready := make(chan error, 1)

	go func() {
		for {
			select {
			case k := <-keys:
				fn(k)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		// The hook needs a message loop on the thread that installed it.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		var filter hotkeyFilter
		callback := syscall.NewCallback(func(nCode, wParam, lParam uintptr) uintptr {
			next := func() uintptr {
				ret, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
				return ret
			}
			if int32(nCode) < 0 {
				return next()
			}
			info := (*kbdLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam)))
			if info.ExtraInfo == injectMarker { // sent by Spellcast itself
				return next()
			}
			shift, _, _ := procGetAsyncKeyState.Call(vkShift)
			e := keyEvent{
				VK:    info.VkCode,
				Up:    wParam == wmKeyUp || wParam == wmSysKeyUp,
				Shift: shift&0x8000 != 0,
			}
			if wParam != wmKeyDown && wParam != wmKeyUp && wParam != wmSysKeyDown && wParam != wmSysKeyUp {
				return next()
			}
			swallow, digit := filter.handle(e)
			if digit != "" {
				select {
				case keys <- digit:
				default: // far behind: drop this one
				}
			}
			if swallow {
				return 1
			}
			return next()
		})

		hmod, _, _ := procGetModuleHandleW.Call(0)
		hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, callback, hmod, 0)
		if hook == 0 {
			ready <- fmt.Errorf("cannot install the keyboard hook: %w", err)
			return
		}
		defer procUnhookWindowsHookEx.Call(hook)

		// CapsLock is a pure modifier now. If it is on from before, switch it
		// off once, so typing is not in capitals.
		if state, _, _ := procGetKeyState.Call(vkCapital); state&1 != 0 {
			_ = sendKeys([]keyInput{newKey(vkCapital, false, false), newKey(vkCapital, true, false)})
		}

		tid, _, _ := procGetCurrentThreadIDLib.Call()
		go func() {
			<-ctx.Done()
			procPostThreadMessageW.Call(tid, wmQuit, 0, 0)
		}()
		ready <- nil
		slog.Info("keyboard hook installed")

		var m msg
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(ret) <= 0 { // WM_QUIT or error
				return
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}()

	select {
	case err := <-ready:
		return err
	case <-ctx.Done():
		return errors.New("stopped before the keyboard hook was installed")
	}
}
