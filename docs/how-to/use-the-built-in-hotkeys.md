# Use the built-in hotkeys

Goal: select text in any app, press CapsLock + a digit, and see Claude's answer in the Spellcast window, with nothing but `spellcast.exe` running.

Prerequisites: an [API key](set-the-api-key.md) and Spellcast [built](build-and-run.md) and running (tray icon visible).

## Steps

1. Select some text in any app.
2. Hold CapsLock and press `1` (or another digit with an action in the [config](../reference/configuration.md)).
3. The result window opens next to the text cursor (or the mouse pointer) and shows Claude's answer as it arrives.
4. **Replace** (or Ctrl+Enter) puts the answer in place of your selection, see [Replace the selected text](replace-the-selected-text.md). Ctrl+D (or **Show changes**) highlights what the answer changed, see [See what the answer changed](see-what-changed.md). Ctrl+C copies the answer. Esc closes the window.

The hotkeys are always on while Spellcast runs. No other tool is needed.

## What the hotkeys change

- CapsLock is a pure modifier: it does not toggle caps lock. Shift + CapsLock still does.
- If caps lock is on when Spellcast starts, Spellcast switches it off once.
- CapsLock + 1 to 9 is swallowed: the digit is not typed.
- Spellcast copies the selection with Ctrl+Insert, then with Ctrl+C if that copied nothing, and restores your clipboard afterwards. In terminals only Ctrl+Insert is used, because Ctrl+C would stop the running program (see `replace.deny.windows` in the [configuration](../reference/configuration.md)).
- The copied text briefly appears in the Windows clipboard history (Win+V), because the app you copy from puts it there. Spellcast cannot hide that. Your old clipboard content is put back afterwards.

A digit without an action shows the tip "No action configured for <n>". No selection shows "No text selected". Both appear next to the mouse pointer for a moment.

## Terminal apps that use the mouse: select with Shift

Some programs in a terminal take over the mouse, for example Claude Code, vim, htop or tmux with mouse mode. When you drag there, the program draws its own highlight, and the terminal itself has no selection. Ctrl+Insert then copies nothing and Spellcast shows "No text selected", even if the program has put the text on the clipboard.

**Hold Shift while you drag.** Windows Terminal then makes its own selection, and CapsLock + digit works as usual.

## If it does not work

| Symptom | Cause and fix |
|---|---|
| Notification "Spellcast: hotkeys not available" | Windows refused the keyboard hook. The log (tray: **Help** > **Open log**) has the reason. Check whether antivirus or security software blocks keyboard hooks, and allow `spellcast.exe` there. |
| Nothing happens in one app only | The app runs as administrator and Spellcast does not. Windows hides the keys of elevated windows from other programs (UIPI). Run Spellcast as administrator too. |
| Nothing happens anywhere, but the tray icon is there | Windows removes a keyboard hook whose callback was too slow, or after some power events. Restart Spellcast. If it happens often, please report it with the log. |
| CapsLock + digit still types the digit | Another keyboard tool (a hotkey script, a remapping tool) sees the key first. Quit it. |
| Tip "No text selected" in a terminal | The program in the terminal uses the mouse (Claude Code, vim, tmux). Hold **Shift** while you drag, see [above](#terminal-apps-that-use-the-mouse-select-with-shift). |
| Tip "No text selected" in another app | The app ignores Ctrl+Insert and Ctrl+C. Copy by hand and use another route. |

Why a keyboard hook, and what it can see: [The keyboard hook and security](../explanation/keyboard-hook-and-security.md).
