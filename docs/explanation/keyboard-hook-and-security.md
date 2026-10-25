# The keyboard hook and security

Spellcast's built-in hotkeys use a Windows low-level keyboard hook (`WH_KEYBOARD_LL`). That is the same mechanism keyloggers use, so it deserves a clear explanation: what it needs, what it can see, what Spellcast does with it, and where it stops.

## Why a hook

The hotkey is CapsLock plus a digit. Windows' simpler `RegisterHotKey` needs a modifier such as Ctrl or Alt and cannot make CapsLock a modifier. It also cannot swallow the digit. Only a hook sees every key and can drop events before the active app gets them.

## Permissions

None on Windows. Any normal user process can install a low-level hook. There is no prompt, no admin right and no setting, and Windows gives no sign that a hook exists. That is why security software looks at hooks, and why an unsigned program that installs one can be flagged. (On macOS, which is not implemented yet, the user must allow Input Monitoring or Accessibility by hand.)

## What the hook can see

Every key the logged-in user types in a normal window, including passwords. A hook is not limited to the keys it cares about. The limit is in what the program does with them.

## What Spellcast does with it

- The hook callback runs a small filter ([hotkey.go](../../internal/platform/hotkey.go)) that looks at two things: the CapsLock key and the digits 1 to 9. All other keys pass through at once.
- The filter stores nothing: no key log, no buffer, no history. It keeps three flags (CapsLock held, a Shift+CapsLock press passed through, which digits are held).
- The only output is the digit of a hotkey, queued for the core. Nothing else leaves the callback.
- Keys are never written to the log. The log has the action and the size of the selected text, not the text.
- Spellcast marks the keys it sends itself (Ctrl+Insert, Ctrl+C, Ctrl+V) and ignores them in the hook. Keys sent by other programs are treated like real keys.
- There is no network path for key data. Spellcast opens no listening port. Its only network traffic is the request to the Anthropic API with the selected text.
- The code is short enough to read in one sitting. Keep it that way: any change that stores or sends keys is a security change and needs a clear reason.

The selected text itself does go to the Anthropic API when you press a hotkey. That is the purpose of the tool, and it is the user's choice for every selection.

## Limits

- **Elevated windows (UIPI).** A normal process does not receive the keys typed into an administrator window and cannot send input to it. Spellcast does not elevate itself. To use the hotkeys there, run Spellcast as administrator.
- **Secure desktop.** The UAC prompt, the lock screen and Ctrl+Alt+Del are out of reach for hooks.
- **Hook timeout.** Windows silently removes a low-level hook whose callback takes too long. The callback is therefore minimal, and the real work (copying, the API call) happens on other goroutines. If the hook is still lost, restart Spellcast.
- **Games and anti-cheat.** Some anti-cheat software blocks or flags global hooks.
- **Other keyboard tools.** Hotkey scripts, remapping tools and on-screen keyboards sit in the same hook chain. Whichever sees a key first can swallow it.
- **Antivirus.** An unsigned executable that installs a keyboard hook can be flagged. Code signing would help. Until then, if security software blocks the hook, Spellcast cannot be used on that machine.
