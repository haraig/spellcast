//go:build windows

package platform

import (
	"bytes"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestKeyInputSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("INPUT is 40 bytes on 64-bit Windows only")
	}
	if got := unsafe.Sizeof(keyInput{}); got != 40 {
		t.Fatalf("sizeof(keyInput) = %d, SendInput needs 40", got)
	}
}

func TestCanReplace(t *testing.T) {
	deny := []string{"WindowsTerminal.exe", "mintty.exe"}
	p := windowsPlatform{denyList: func() []string { return deny }}

	// A real window that exists: the desktop.
	desktop, _, _ := user32.NewProc("GetDesktopWindow").Call()

	tests := []struct {
		name   string
		target Target
		want   bool
		reason string
	}{
		{"ok", Target{App: "NOTEPAD.EXE", Handle: desktop}, true, ""},
		{"no source app known", Target{Handle: desktop}, true, ""},
		{"denied, same case", Target{App: "WindowsTerminal.exe", Handle: desktop}, false, "Replace is disabled in WindowsTerminal.exe."},
		{"denied, other case", Target{App: "MINTTY.EXE", Handle: desktop}, false, "Replace is disabled in MINTTY.EXE."},
		{"no handle", Target{App: "NOTEPAD.EXE"}, false, "The source window is unknown."},
		{"closed window", Target{App: "NOTEPAD.EXE", Handle: 0x7FFFFFF0}, false, "The source window was closed."},
	}
	for _, tt := range tests {
		ok, reason := p.CanReplace(tt.target)
		if ok != tt.want || reason != tt.reason {
			t.Errorf("%s: got (%v, %q), want (%v, %q)", tt.name, ok, reason, tt.want, tt.reason)
		}
	}

	deny = nil // the list is read on every call: a config reload applies at once
	if ok, _ := p.CanReplace(Target{App: "WindowsTerminal.exe", Handle: desktop}); !ok {
		t.Error("an empty deny list must allow everything")
	}
}

func TestReplaceTargetGone(t *testing.T) {
	p := windowsPlatform{}
	if err := p.Replace(t.Context(), Target{Handle: 0x7FFFFFF0}, "x", ""); err != ErrTargetGone {
		t.Errorf("err = %v, want ErrTargetGone", err)
	}
}

// formatsOnClipboard lists the formats on the clipboard.
func formatsOnClipboard(t *testing.T) map[uint32]bool {
	t.Helper()
	if err := openClipboard(); err != nil {
		t.Fatal(err)
	}
	defer closeClipboard()
	got := map[uint32]bool{}
	var f uint32
	for {
		n, _, _ := procEnumClipboardFormats.Call(uintptr(f))
		f = uint32(n)
		if f == 0 {
			return got
		}
		got[f] = true
	}
}

// This test uses the real clipboard: it saves the current content and puts it back.
func TestClipboardRoundTrip(t *testing.T) {
	orig, err := readSnapshot()
	if err != nil {
		t.Skipf("clipboard not available: %v", err)
	}
	t.Cleanup(func() { _ = restoreSnapshot(orig) })

	// The user's content: text plus a private format.
	p, _ := windows.UTF16PtrFromString("SpellcastTestFormat")
	custom, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(p)))
	customData := []byte{1, 2, 3, 4, 5, 0, 255}
	text, _ := utf16Bytes("Grüße 👋")
	if err := openClipboard(); err != nil {
		t.Fatal(err)
	}
	procEmptyClipboard.Call()
	if err := setData(cfUnicodeText, text); err != nil {
		t.Fatal(err)
	}
	if err := setData(uint32(custom), customData); err != nil {
		t.Fatal(err)
	}
	closeClipboard()

	snap, err := readSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incomplete {
		t.Error("snapshot incomplete")
	}
	var gotCustom []byte
	for _, it := range snap.Items {
		if it.Format == uint32(custom) {
			gotCustom = it.Data
		}
		if skipFormat(it.Format) {
			t.Errorf("format %d should not be saved", it.Format)
		}
	}
	if !bytes.Equal(gotCustom, customData) {
		t.Fatalf("custom format in snapshot = %v, want %v", gotCustom, customData)
	}

	// Our result replaces everything and is marked for "no history, no cloud".
	seq, err := writeText("RESULT\r\nline two")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := clipboardText(); !ok || got != "RESULT\r\nline two" {
		t.Errorf("clipboard text = %q, %v", got, ok)
	}
	if seq == 0 || clipboardSequence() != seq {
		t.Errorf("sequence number: stored %d, now %d", seq, clipboardSequence())
	}
	formats := formatsOnClipboard(t)
	if formats[uint32(custom)] {
		t.Error("old private format must be gone after writeText")
	}
	if !formats[fmtCanHistory] || !formats[fmtExcludeMonitor] || !formats[fmtCanCloud] {
		t.Errorf("history/cloud exclusion formats missing: %v", formats)
	}

	// Restore brings the user's content back.
	if err := restoreSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	if got, ok := clipboardText(); !ok || got != "Grüße 👋" {
		t.Errorf("restored text = %q, %v", got, ok)
	}
	back, err := readSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var backCustom []byte
	for _, it := range back.Items {
		if it.Format == uint32(custom) {
			backCustom = it.Data
		}
	}
	if !bytes.Equal(backCustom, customData) {
		t.Errorf("restored custom format = %v, want %v", backCustom, customData)
	}
}

func TestWriteContentWithHTML(t *testing.T) {
	old, err := readSnapshot()
	if err != nil {
		t.Skip("clipboard not available:", err)
	}
	t.Cleanup(func() { _ = restoreSnapshot(old) })

	if _, err := writeContent("**fett**", "<p><strong>fett</strong></p>"); err != nil {
		t.Fatal(err)
	}
	if got, ok := clipboardText(); !ok || got != "**fett**" {
		t.Errorf("plain text = %q, %v", got, ok)
	}
	snap, err := readSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var html []byte
	for _, it := range snap.Items {
		if it.Format == fmtHTML {
			html = it.Data
		}
	}
	if !bytes.Contains(html, []byte("<!--StartFragment--><p><strong>fett</strong></p><!--EndFragment-->")) {
		t.Errorf("HTML Format = %q", html)
	}
	if !formatsOnClipboard(t)[fmtCanHistory] {
		t.Error("history exclusion missing")
	}
}

func TestWriteTextRejectsNUL(t *testing.T) {
	if _, err := writeText("a\x00b"); err == nil {
		t.Error("text with a NUL cannot be put on the clipboard as is")
	}
}

func TestAutostart(t *testing.T) {
	// Another value name: the real "Spellcast" entry is never touched.
	old := autostartKey
	autostartKey = "SpellcastTestAutostart"
	t.Cleanup(func() {
		_ = windowsPlatform{}.SetAutostart(false)
		autostartKey = old
	})

	p := windowsPlatform{}
	if err := p.SetAutostart(false); err != nil { // removing a missing value is fine
		t.Fatalf("disable when absent: %v", err)
	}
	if p.AutostartEnabled() {
		t.Fatal("enabled before it was set")
	}
	if err := p.SetAutostart(true); err != nil {
		t.Fatal(err)
	}
	if !p.AutostartEnabled() {
		t.Error("not enabled after SetAutostart(true)")
	}
	if err := p.SetAutostart(false); err != nil {
		t.Fatal(err)
	}
	if p.AutostartEnabled() {
		t.Error("still enabled after SetAutostart(false)")
	}
}
