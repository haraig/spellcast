package platform

import "testing"

func down(vk uint32) keyEvent      { return keyEvent{VK: vk} }
func up(vk uint32) keyEvent        { return keyEvent{VK: vk, Up: true} }
func shiftDown(vk uint32) keyEvent { return keyEvent{VK: vk, Shift: true} }

type step struct {
	e       keyEvent
	swallow bool
	digit   string
}

func run(t *testing.T, steps []step) {
	t.Helper()
	var f hotkeyFilter
	for i, s := range steps {
		swallow, digit := f.handle(s.e)
		if swallow != s.swallow || digit != s.digit {
			t.Fatalf("step %d %+v: got (swallow=%v, digit=%q), want (%v, %q)", i, s.e, swallow, digit, s.swallow, s.digit)
		}
	}
}

func TestHotkeyChord(t *testing.T) {
	run(t, []step{
		{down(vkCapital), true, ""},
		{down('1'), true, "1"},
		{up('1'), true, ""},
		{down('2'), true, "2"}, // CapsLock still held: a second hotkey
		{up('2'), true, ""},
		{up(vkCapital), true, ""},
	})
}

func TestHotkeyAutoRepeatFiresOnce(t *testing.T) {
	run(t, []step{
		{down(vkCapital), true, ""},
		{down(vkCapital), true, ""}, // CapsLock auto-repeat
		{down('3'), true, "3"},
		{down('3'), true, ""}, // digit auto-repeat
		{down('3'), true, ""},
		{up('3'), true, ""},
		{up(vkCapital), true, ""},
	})
}

func TestHotkeyCapsReleasedBeforeDigit(t *testing.T) {
	run(t, []step{
		{down(vkCapital), true, ""},
		{down('4'), true, "4"},
		{up(vkCapital), true, ""},
		{up('4'), true, ""}, // the digit's key-up is still swallowed
		{down('4'), false, ""},
		{up('4'), false, ""}, // typed normally afterwards
	})
}

func TestHotkeyDigitsWithoutCapsPassThrough(t *testing.T) {
	run(t, []step{
		{down('1'), false, ""},
		{up('1'), false, ""},
		{down('9'), false, ""},
		{up('9'), false, ""},
	})
}

func TestHotkeyOtherKeysPassThrough(t *testing.T) {
	run(t, []step{
		{down(vkCapital), true, ""},
		{down('A'), false, ""},
		{up('A'), false, ""},
		{down('0'), false, ""}, // 0 is not an action key
		{up('0'), false, ""},
		{up(vkCapital), true, ""},
	})
}

func TestHotkeyCapsAloneDoesNothing(t *testing.T) {
	run(t, []step{
		{down(vkCapital), true, ""},
		{up(vkCapital), true, ""},
	})
}

func TestHotkeyShiftCapsTogglesAsUsual(t *testing.T) {
	run(t, []step{
		{shiftDown(vkCapital), false, ""},
		{shiftDown(vkCapital), false, ""}, // auto-repeat goes through too
		{up(vkCapital), false, ""},
		// Plain CapsLock is a modifier again afterwards.
		{down(vkCapital), true, ""},
		{down('1'), true, "1"},
		{up('1'), true, ""},
		{up(vkCapital), true, ""},
	})
}

func TestHotkeyShiftReleasedBeforeCapsUp(t *testing.T) {
	// Shift+CapsLock down, then Shift released, then CapsLock up: the up must
	// still go through, or the toggle would never finish.
	run(t, []step{
		{shiftDown(vkCapital), false, ""},
		{up(vkCapital), false, ""},
	})
}
