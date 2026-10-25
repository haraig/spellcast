# How to troubleshoot Spellcast

Start with the log: tray menu, **Help** > **Open log**. It has one line per request and the reason for every error. It never contains your text or your API key.

| Problem | What to do |
|---|---|
| No tray icon after the start | Windows may hide new icons: click the small arrow in the notification area, or drag the wand icon onto the taskbar. Starting `spellcast.exe` again shows "Spellcast is already running" if it runs. |
| Windows SmartScreen warns about the exe | The exe is not code-signed. Choose **More info** > **Run anyway** if you trust the download. [Install Spellcast](install-spellcast.md#check-the-download-optional) shows how to check it, or build it yourself. |
| "No text selected" in a terminal | The program in the terminal uses the mouse (for example Claude Code or vim), so the terminal has no selection. **Hold Shift while you drag.** See [Use the built-in hotkeys](use-the-built-in-hotkeys.md#terminal-apps-that-use-the-mouse-select-with-shift). |
| "No text selected" in another app | The app ignores Ctrl+Insert, and Ctrl+C did not work. Try selecting again; see [Use the built-in hotkeys](use-the-built-in-hotkeys.md). |
| Nothing happens in one app only | The app runs as administrator. Windows keeps other programs from reading its keys (UIPI). Run Spellcast as administrator too. |
| VS Code: the whole line is processed | With nothing selected, VS Code copies the current line. Select the text first. |
| Replace does nothing | The field is read-only, or the app runs as administrator. See [Replace the selected text](replace-the-selected-text.md). |
| "No API key" | Set the key: tray menu, **Set API key…**. See [Set the API key](set-the-api-key.md). |
| "Config invalid, using defaults" | The notification names the key and the line. Fix it with **Settings** > **Open config**, then **Settings** > **Reload config**. See [Configuration](../reference/configuration.md). |
| The answer is cut off | Raise `max_tokens` for that action in the config. |

Where Spellcast keeps its files: [Files and locations](../reference/files-and-locations.md).
