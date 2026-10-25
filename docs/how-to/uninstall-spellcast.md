# How to uninstall Spellcast

Spellcast has no installer. Uninstalling means: remove the autostart entry, optionally delete your settings and API key, then delete `spellcast.exe`.

## From the tray menu

1. Click the tray icon and choose **Help** > **Remove from Windows…**.
2. Choose one:
   - **Remove**: removes the autostart entry and the notification registration. Your settings and API key stay, so a later install picks them up.
   - **Remove and delete my settings**: also deletes the config folder (`%AppData%\spellcast`), the WebView2 data (`%AppData%\spellcast.exe`), the log folder (`%LocalAppData%\spellcast`) and the API key in the Windows Credential Manager.
   - **Cancel**: changes nothing.
3. Spellcast quits and then does the cleanup. Delete `spellcast.exe` yourself, or run `winget uninstall` if you installed it with winget.

## From the command line

```powershell
spellcast.exe --uninstall           # same as "Remove"
spellcast.exe --uninstall --purge   # same as "Remove and delete my settings"
```

If Spellcast is running, the running instance does the cleanup and quits. The exit code is 0 when the cleanup worked (or was handed to the running instance) and 1 when a step failed. Spellcast has no console output: the result is in the log, or in a notification when it was running. With `--purge`, the log is deleted too.

The exe is never deleted by Spellcast.
