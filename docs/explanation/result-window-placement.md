# How the result window is placed

The result window appears next to the text the selection came from. This page explains how and why.

## Where it appears

1. **Anchor:** the text cursor (caret) of the source window, if the app reports one. Classic Win32 apps (Notepad, Outlook, Word) do. Chrome, VS Code and other Electron apps often do not. Then the anchor is the mouse pointer.
2. **Position:** just below the anchor. If it does not fit below, above it. It is then clamped to the work area (screen minus taskbar) of the anchor's monitor.
3. The position is computed fresh on every trigger, while the source window still has its caret.

The placement rule is a pure function (`platform.Place`) with unit tests. Reading the caret and moving the window are Windows code in `internal/platform`.

## Why physical pixels

The app is per-monitor DPI aware (v2, from `build/windows/wails.exe.manifest`). Caret, mouse and monitor coordinates from Win32 are therefore physical pixels. The window is moved with `SetWindowPos`, which also takes physical pixels. So no conversion to Wails' logical coordinates is needed, and a 150 % monitor needs no special case.

Moving a window to a monitor with another scale factor changes its pixel size. So the window is moved, its size is read again, and the position is checked a second time.

## Size memory

The window is resizable. Its size is saved to `window.json` next to `config.yaml` when the window is closed (X button or Esc) and restored at the next start. Invalid or missing data gives the default size (480×320).

## Closing and focus

- The window is created once and reused. Closing it only hides it.
- "Show last answer" in the tray menu reopens the window with the last answer, placed at the mouse pointer (there is no text cursor to follow).
- While it is open, it stays on top of its source window only, see [How Replace works](how-replace-works.md).
- Esc closes it. It does not close when it loses focus, because the user often switches back to the source app while the answer stays visible.
- The window takes keyboard focus when it opens. Windows only allows the foreground process to hand over focus. With the built-in hotkeys Spellcast has just sent the copy shortcut itself, so it may take the focus.
