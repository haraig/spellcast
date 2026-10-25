package platform

// Virtual-key codes used by the hotkey filter (same values on Windows).
const (
	vkCapital = 0x14
	vkDigit1  = 0x31 // '1'
	vkDigit9  = 0x39 // '9'
)

// keyEvent is one keyboard event as the low-level hook sees it.
type keyEvent struct {
	VK    uint32
	Up    bool
	Shift bool // a Shift key is held
}

// hotkeyFilter implements "CapsLock + digit" as a pure function of the key
// events, so it can be tested without a keyboard hook. It is used from one
// thread only (the hook's).
//
//   - CapsLock is a pure modifier: its down and up events are swallowed, so
//     the caps lock state never changes by accident.
//   - Shift+CapsLock goes through to Windows and toggles caps lock as usual.
//   - A digit 1-9 pressed while CapsLock is held is swallowed (it must not be
//     typed) and reported once; its key-up is swallowed too, even if CapsLock
//     was released first.
//   - Every other key passes through untouched. Nothing is stored or logged.
type hotkeyFilter struct {
	capsDown bool // CapsLock is held and swallowed
	capsPass bool // a Shift+CapsLock press that was let through
	held     map[uint32]bool
}

// handle takes one event. swallow tells the hook to drop the event; digit is
// "1".."9" when the event completes a hotkey, else "".
func (f *hotkeyFilter) handle(e keyEvent) (swallow bool, digit string) {
	switch {
	case e.VK == vkCapital:
		if !e.Up {
			switch {
			case f.capsPass: // auto-repeat of a press that goes through
				return false, ""
			case f.capsDown: // auto-repeat of a swallowed press
				return true, ""
			case e.Shift:
				f.capsPass = true
				return false, ""
			}
			f.capsDown = true
			return true, ""
		}
		switch {
		case f.capsPass:
			f.capsPass = false
			return false, ""
		case f.capsDown:
			f.capsDown = false
			return true, ""
		}
		return false, ""

	case e.VK >= vkDigit1 && e.VK <= vkDigit9:
		if !e.Up {
			if !f.capsDown {
				return false, ""
			}
			if f.held == nil {
				f.held = make(map[uint32]bool)
			}
			if f.held[e.VK] { // auto-repeat
				return true, ""
			}
			f.held[e.VK] = true
			return true, string(rune(e.VK))
		}
		if f.held[e.VK] {
			delete(f.held, e.VK)
			return true, ""
		}
	}
	return false, ""
}
