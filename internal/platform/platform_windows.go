//go:build windows

package platform

import (
	"image"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procGetGUIThreadInfo         = user32.NewProc("GetGUIThreadInfo")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procClientToScreen           = user32.NewProc("ClientToScreen")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procMonitorFromPoint         = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
)

const (
	monitorDefaultToNearest = 2

	swpNoSize     = 0x0001
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
)

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

func (r rect) toImage() image.Rectangle {
	return image.Rect(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom))
}

type guiThreadInfo struct {
	Size      uint32
	Flags     uint32
	Active    windows.HWND
	Focus     windows.HWND
	Capture   windows.HWND
	MenuOwner windows.HWND
	MoveSize  windows.HWND
	Caret     windows.HWND
	CaretRect rect
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

// New returns the Windows implementation. The process is per-monitor DPI
// aware v2 (build/windows/wails.exe.manifest), so all coordinates here are
// physical pixels and no scaling is needed.
//
// denyList returns the executables where Replace is disabled (terminals); it
// is called each time, so a config reload applies at once.
func New(denyList func() []string) Platform { return windowsPlatform{denyList: denyList} }

type windowsPlatform struct {
	denyList func() []string
}

func (windowsPlatform) Anchor(t Target) (image.Rectangle, bool) {
	if r, ok := caretRect(windows.HWND(t.Handle)); ok {
		return r, true
	}
	var p point
	if ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p))); ret == 0 {
		slog.Warn("GetCursorPos failed")
	}
	return image.Rect(int(p.X), int(p.Y), int(p.X)+1, int(p.Y)+1), false
}

// caretRect asks the GUI thread of the target window for its text caret and
// converts it to screen coordinates. Classic Win32 apps (Notepad, Outlook,
// Word) report it; Chrome, VS Code and other Electron apps often do not.
func caretRect(hwnd windows.HWND) (image.Rectangle, bool) {
	if hwnd == 0 {
		return image.Rectangle{}, false
	}
	tid, _, _ := procGetWindowThreadProcessID.Call(uintptr(hwnd), 0)
	if tid == 0 {
		return image.Rectangle{}, false
	}
	info := guiThreadInfo{Size: uint32(unsafe.Sizeof(guiThreadInfo{}))}
	if ret, _, _ := procGetGUIThreadInfo.Call(tid, uintptr(unsafe.Pointer(&info))); ret == 0 {
		return image.Rectangle{}, false
	}
	if info.Caret == 0 || info.CaretRect == (rect{}) {
		return image.Rectangle{}, false
	}
	topLeft := point{info.CaretRect.Left, info.CaretRect.Top}
	bottomRight := point{info.CaretRect.Right, info.CaretRect.Bottom}
	if ret, _, _ := procClientToScreen.Call(uintptr(info.Caret), uintptr(unsafe.Pointer(&topLeft))); ret == 0 {
		return image.Rectangle{}, false
	}
	if ret, _, _ := procClientToScreen.Call(uintptr(info.Caret), uintptr(unsafe.Pointer(&bottomRight))); ret == 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(int(topLeft.X), int(topLeft.Y), int(bottomRight.X), int(bottomRight.Y)), true
}

func (windowsPlatform) PlaceWindow(window uintptr, anchor image.Rectangle) {
	hwnd := windows.HWND(window)
	work := workArea(image.Pt((anchor.Min.X+anchor.Max.X)/2, (anchor.Min.Y+anchor.Max.Y)/2))

	// Moving to a monitor with another scale factor changes the window's
	// pixel size (WM_DPICHANGED). Move, re-read the size, and place again.
	for i := 0; i < 2; i++ {
		r, ok := windowRect(hwnd)
		if !ok {
			return
		}
		pos := Place(anchor, r.Size(), work)
		if pos == r.Min {
			return
		}
		procSetWindowPos.Call(uintptr(hwnd), 0, uintptr(pos.X), uintptr(pos.Y), 0, 0,
			swpNoSize|swpNoZOrder|swpNoActivate)
	}
}

func windowRect(hwnd windows.HWND) (image.Rectangle, bool) {
	var r rect
	if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r))); ret == 0 {
		return image.Rectangle{}, false
	}
	return r.toImage(), true
}

// workArea returns the work area (screen minus taskbar) of the monitor
// containing pt, or the nearest monitor.
func workArea(pt image.Point) image.Rectangle {
	// MonitorFromPoint takes the POINT by value; on amd64 and arm64 it is
	// packed into one register.
	packed := uintptr(uint32(int32(pt.X))) | uintptr(uint32(int32(pt.Y)))<<32
	mon, _, _ := procMonitorFromPoint.Call(packed, monitorDefaultToNearest)
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ret, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&info))); ret == 0 {
		slog.Warn("GetMonitorInfo failed, not limiting the window to a work area")
		return image.Rect(-1<<30, -1<<30, 1<<30, 1<<30)
	}
	return info.Work.toImage()
}
