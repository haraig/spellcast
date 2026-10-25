//go:build windows

package platform

import (
	"context"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procSetWinEventHook    = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent     = user32.NewProc("UnhookWinEvent")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostThreadMessageW = user32.NewProc("PostThreadMessageW")
)

const (
	eventSystemForeground = 0x0003
	winEventOutOfContext  = 0x0000
	wmQuit                = 0x0012
)

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

// WatchForeground reports foreground window changes until ctx ends. The hook
// callback only queues the change; fn is called from another goroutine, in
// order, so a slow fn cannot make Windows drop the hook.
func (windowsPlatform) WatchForeground(ctx context.Context, fn func(Target)) {
	events := make(chan Target, 32)

	go func() {
		for {
			select {
			case t := <-events:
				fn(t)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		// The hook needs a message loop on the thread that installed it.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		callback := syscall.NewCallback(func(hook, event, hwnd, idObject, idChild, thread, eventTime uintptr) uintptr {
			if hwnd == 0 {
				return 0
			}
			select {
			case events <- Target{Handle: uintptr(rootWindow(windows.HWND(hwnd)))}:
			default: // fn is far behind: drop this change, the next one follows
			}
			return 0
		})
		hook, _, _ := procSetWinEventHook.Call(eventSystemForeground, eventSystemForeground, 0, callback, 0, 0, winEventOutOfContext)
		if hook == 0 {
			return
		}
		defer procUnhookWinEvent.Call(hook)

		tid, _, _ := procGetCurrentThreadIDLib.Call()
		go func() {
			<-ctx.Done()
			procPostThreadMessageW.Call(tid, wmQuit, 0, 0)
		}()

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
}
