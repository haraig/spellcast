//go:build windows

package platform

import (
	"image"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procDestroyWindow         = user32.NewProc("DestroyWindow")
	procPeekMessageW          = user32.NewProc("PeekMessageW")
	procGetDC                 = user32.NewProc("GetDC")
	procReleaseDC             = user32.NewProc("ReleaseDC")
	procSendMessageW          = user32.NewProc("SendMessageW")
	procGetStockObject        = gdi32.NewProc("GetStockObject")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
)

const (
	wsPopup                = 0x80000000
	wsBorder               = 0x00800000
	wsExTopmost            = 0x00000008
	wsExToolWindow         = 0x00000080
	wsExNoActivate         = 0x08000000
	ssCenter               = 0x00000001
	ssCenterImage          = 0x00000200
	swShowNoActivate       = 4
	wmSetFont              = 0x0030
	defaultGUIFont         = 17
	pmRemove               = 0x0001
	tipDuration            = 1500 * time.Millisecond
	tipPadX, tipPadY       = 20, 10
	tipOffsetX, tipOffsetY = 16, 22
)

func pointToImage(p point) image.Point { return image.Pt(int(p.X), int(p.Y)) }

type size struct{ CX, CY int32 }

// ShowTip shows a small message next to the mouse pointer for a moment. It
// never takes the focus. It returns at once.
func (windowsPlatform) ShowTip(text string) {
	go showTip(text)
}

func showTip(text string) {
	// A window belongs to the thread that created it, and that thread must
	// pump messages: use a thread of its own.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	class, _ := windows.UTF16PtrFromString("STATIC")
	title, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	font, _, _ := procGetStockObject.Call(defaultGUIFont)

	// Measure the text with the font the window will use.
	hdc, _, _ := procGetDC.Call(0)
	old, _, _ := procSelectObject.Call(hdc, font)
	var sz size
	utf16, _ := windows.UTF16FromString(text)
	procGetTextExtentPoint32W.Call(hdc, uintptr(unsafe.Pointer(&utf16[0])), uintptr(len(utf16)-1), uintptr(unsafe.Pointer(&sz)))
	procSelectObject.Call(hdc, old)
	procReleaseDC.Call(0, hdc)

	w, h := int(sz.CX)+tipPadX, int(sz.CY)+tipPadY
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	work := workArea(pointToImage(p))
	x, y := int(p.X)+tipOffsetX, int(p.Y)+tipOffsetY
	if x+w > work.Max.X {
		x = work.Max.X - w
	}
	if y+h > work.Max.Y {
		y = int(p.Y) - h - tipOffsetY
	}
	if x < work.Min.X {
		x = work.Min.X
	}
	if y < work.Min.Y {
		y = work.Min.Y
	}

	hwnd, _, _ := procCreateWindowExW.Call(
		wsExTopmost|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)),
		wsPopup|wsBorder|ssCenter|ssCenterImage,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, 0, 0)
	if hwnd == 0 {
		return
	}
	defer procDestroyWindow.Call(hwnd)
	procSendMessageW.Call(hwnd, wmSetFont, font, 1)
	procShowWindow.Call(hwnd, swShowNoActivate)

	var m msg
	for end := time.Now().Add(tipDuration); time.Now().Before(end); {
		for {
			ret, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
			if ret == 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
		time.Sleep(15 * time.Millisecond)
	}
}
