# Files and locations

Where Spellcast keeps its data on Windows. **Help** > **Remove from Windows…** with **Remove and delete my settings** (or `spellcast.exe --uninstall --purge`) deletes all of it except the exe. **Remove** alone deletes only the autostart value and the notification registration.

| What | Where | Notes |
|---|---|---|
| Config | `%AppData%\spellcast\config.yaml` | Created with defaults on the first start. [Reference](configuration.md) |
| Usage totals | `%AppData%\spellcast\usage.json` | One entry per day; delete it to reset the totals |
| Window size | `%AppData%\spellcast\window.json` | The last size of the result window |
| Log | `%LocalAppData%\spellcast\spellcast.log` | Requests, errors; never your text or the key |
| WebView2 data | `%LocalAppData%\spellcast\WebView2` | Cache of the window's web view; local, so it does not roam with the profile |
| API key | Windows Credential Manager, generic credential `spellcast` (user `anthropic-api-key`) | Or the environment variable `ANTHROPIC_API_KEY` |
| Autostart | Registry value `Spellcast` under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` | Set by **Settings** > **Start with Windows** |
| Notification registration | `HKCU\Software\Classes\AppUserModelId\Spellcast`, the COM entry under `HKCU\Software\Classes\CLSID` that it names, `HKCU\Software\Spellcast`, and an icon copy `Spellcast{…}.png` in `%TEMP%` | Written by the notification service at every start |
| Program | `spellcast.exe`, wherever you put it | Not deleted by Spellcast |
