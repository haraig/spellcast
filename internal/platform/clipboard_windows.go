//go:build windows

package platform

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalSize   = kernel32.NewProc("GlobalSize")

	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procEnumClipboardFormats       = user32.NewProc("EnumClipboardFormats")
	procRegisterClipboardFormatW   = user32.NewProc("RegisterClipboardFormatW")
	procGetClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
)

const (
	gmemMoveable = 0x0002

	// Standard clipboard formats.
	cfText         = 1
	cfBitmap       = 2
	cfMetafilePic  = 3
	cfOEMText      = 7
	cfPalette      = 9
	cfUnicodeText  = 13
	cfEnhMetafile  = 14
	cfOwnerDisplay = 0x0080
	cfDspFirst     = 0x0081 // CF_DSPTEXT .. CF_DSPENHMETAFILE
	cfDspLast      = 0x008E
	cfGDIObjFirst  = 0x0300 // private GDI object formats
	cfGDIObjLast   = 0x03FF

	// Limits for the best-effort backup of the old clipboard.
	maxItemBytes  = 64 << 20
	maxTotalBytes = 128 << 20
)

// Formats that keep text out of the Windows clipboard history and cloud sync.
// Set on the temporary result text, and when the old content is restored, so
// neither of them shows up there again.
var (
	historyFormatsOnce sync.Once
	fmtExcludeMonitor  uint32 // ExcludeClipboardContentFromMonitorProcessing
	fmtCanHistory      uint32 // CanIncludeInClipboardHistory
	fmtCanCloud        uint32 // CanUploadToCloudClipboard
	fmtHTML            uint32 // "HTML Format" (CF_HTML)
)

func registerHistoryFormats() {
	historyFormatsOnce.Do(func() {
		reg := func(name string) uint32 {
			p, _ := windows.UTF16PtrFromString(name)
			id, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(p)))
			return uint32(id)
		}
		fmtExcludeMonitor = reg("ExcludeClipboardContentFromMonitorProcessing")
		fmtCanHistory = reg("CanIncludeInClipboardHistory")
		fmtCanCloud = reg("CanUploadToCloudClipboard")
		fmtHTML = reg("HTML Format")
	})
}

// clipboardItem is one format with its bytes.
type clipboardItem struct {
	Format uint32
	Data   []byte
}

// clipboardSnapshot is a best-effort copy of the clipboard. Formats that are
// GDI handles (bitmaps, metafiles) or can be derived from other formats are
// not saved.
type clipboardSnapshot struct {
	Items      []clipboardItem
	Incomplete bool // some format was too big or could not be read
}

// openClipboard opens the clipboard, retrying for a short time because
// another program may have it open.
func openClipboard() error {
	deadline := time.Now().Add(time.Second)
	for {
		if ret, _, _ := procOpenClipboard.Call(0); ret != 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("the clipboard is in use by another program")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func closeClipboard() { procCloseClipboard.Call() }

func clipboardSequence() uint32 {
	n, _, _ := procGetClipboardSequenceNumber.Call()
	return uint32(n)
}

// skipFormat reports formats that are not saved.
func skipFormat(f uint32) bool {
	switch f {
	case cfText, cfOEMText: // derived from CF_UNICODETEXT
		return true
	case cfBitmap, cfMetafilePic, cfPalette, cfEnhMetafile, cfOwnerDisplay: // not global memory
		return true
	}
	return (f >= cfDspFirst && f <= cfDspLast) || (f >= cfGDIObjFirst && f <= cfGDIObjLast)
}

// readSnapshot copies the clipboard. The clipboard must be open.
func readSnapshotOpen() clipboardSnapshot {
	var snap clipboardSnapshot
	var total int
	var format uint32
	for {
		next, _, _ := procEnumClipboardFormats.Call(uintptr(format))
		format = uint32(next)
		if format == 0 {
			break
		}
		if skipFormat(format) {
			continue
		}
		h, _, _ := procGetClipboardData.Call(uintptr(format))
		if h == 0 {
			snap.Incomplete = true
			continue
		}
		size, _, _ := procGlobalSize.Call(h)
		if size == 0 {
			continue
		}
		if size > maxItemBytes || total+int(size) > maxTotalBytes {
			snap.Incomplete = true
			continue
		}
		ptr, _, _ := procGlobalLock.Call(h)
		if ptr == 0 {
			snap.Incomplete = true
			continue
		}
		data := make([]byte, size)
		copy(data, unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&ptr))), size))
		procGlobalUnlock.Call(h)
		snap.Items = append(snap.Items, clipboardItem{Format: format, Data: data})
		total += int(size)
	}
	return snap
}

// readSnapshot opens the clipboard and copies it.
func readSnapshot() (clipboardSnapshot, error) {
	if err := openClipboard(); err != nil {
		return clipboardSnapshot{}, err
	}
	defer closeClipboard()
	return readSnapshotOpen(), nil
}

// setData puts bytes on the open clipboard in the given format.
func setData(format uint32, data []byte) error {
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(data)))
	if h == 0 {
		return errors.New("out of memory")
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		procGlobalFree.Call(h)
		return errors.New("cannot lock memory")
	}
	copy(unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&ptr))), len(data)), data)
	procGlobalUnlock.Call(h)
	if ret, _, err := procSetClipboardData.Call(uintptr(format), h); ret == 0 {
		procGlobalFree.Call(h) // on success, the system owns the memory
		return fmt.Errorf("SetClipboardData(%d): %v", format, err)
	}
	return nil
}

// setHistoryFlags marks the clipboard content as not for the history or the cloud.
func setHistoryFlags() {
	registerHistoryFormats()
	one := []byte{1, 0, 0, 0}
	zero := []byte{0, 0, 0, 0}
	_ = setData(fmtExcludeMonitor, one)
	_ = setData(fmtCanHistory, zero)
	_ = setData(fmtCanCloud, zero)
}

func utf16Bytes(text string) ([]byte, error) {
	u, err := windows.UTF16FromString(text) // includes the terminating zero
	if err != nil {
		return nil, err // text contains a NUL
	}
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i] = byte(c)
		b[2*i+1] = byte(c >> 8)
	}
	return b, nil
}

// writeText replaces the clipboard with text (CF_UNICODETEXT) and returns the
// clipboard sequence number right after. If the number differs later, someone
// else has changed the clipboard.
func writeText(text string) (uint32, error) { return writeContent(text, "") }

// writeContent is writeText plus, if html is not empty, the HTML fragment as
// "HTML Format" (CF_HTML).
func writeContent(text, html string) (uint32, error) {
	data, err := utf16Bytes(text)
	if err != nil {
		return 0, err
	}
	var htmlData []byte
	if html != "" {
		htmlData = append(CFHTML(html), 0)
	}
	if err := func() error {
		if err := openClipboard(); err != nil {
			return err
		}
		defer closeClipboard()
		procEmptyClipboard.Call()
		if err := setData(cfUnicodeText, data); err != nil {
			return err
		}
		if htmlData != nil {
			registerHistoryFormats()
			if err := setData(fmtHTML, htmlData); err != nil {
				return err
			}
		}
		setHistoryFlags()
		return nil
	}(); err != nil {
		return 0, err
	}
	return clipboardSequence(), nil
}

// restoreSnapshot puts the saved content back. An empty snapshot clears the clipboard.
func restoreSnapshot(snap clipboardSnapshot) error {
	if err := openClipboard(); err != nil {
		return err
	}
	defer closeClipboard()
	procEmptyClipboard.Call()
	var firstErr error
	for _, it := range snap.Items {
		if err := setData(it.Format, it.Data); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if len(snap.Items) > 0 {
		setHistoryFlags() // do not record the old content a second time
	}
	return firstErr
}

// clipboardText reads CF_UNICODETEXT (for tests and checks).
func clipboardText() (string, bool) {
	if err := openClipboard(); err != nil {
		return "", false
	}
	defer closeClipboard()
	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", false
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return "", false
	}
	defer procGlobalUnlock.Call(h)
	return windows.UTF16PtrToString((*uint16)(*(*unsafe.Pointer)(unsafe.Pointer(&ptr)))), true
}
