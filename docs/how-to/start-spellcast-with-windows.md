# Start Spellcast with Windows

Goal: Spellcast and the hotkeys are ready after you log in.

## Spellcast itself

Click the tray icon and tick **Settings** > **Start with Windows**. Untick it to remove the entry.

This writes the value `Spellcast` under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, with the full path of the running `spellcast.exe`. If you move or rebuild the file, untick and tick the entry again.

The built-in hotkeys start with Spellcast, so nothing else is needed.

## Check

Log out and in again. The tray icon should appear, and CapsLock + a digit should work on selected text. The entry in **Task Manager**, **Startup apps**, lists `spellcast.exe`.
