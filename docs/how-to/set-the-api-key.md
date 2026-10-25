# Set the Anthropic API key

Goal: let Spellcast send requests to Claude.

## Steps

1. Create an API key in the [Anthropic Console](https://console.anthropic.com/).
2. Start Spellcast. Click the tray icon and choose **Set API key…**. While no key is set, it is the first entry; later it is under **Settings**.
3. Paste the key and press Enter (or **Save**).
4. Spellcast checks the key with a free request (no tokens are used) and stores it in the Windows Credential Manager under the name `spellcast`. It is used at once, no restart needed.

If the check cannot reach the API (no network), the key is stored anyway and the message says so. A key the API rejects is not stored.

## Alternative: environment variable

Without a stored key, Spellcast uses the environment variable `ANTHROPIC_API_KEY`. Set it for your user and restart Spellcast. If both exist, the stored key wins. **Settings** > **Reload config** in the tray menu re-reads the key.

## Check what is in use

- Open **Settings** > **Set API key…** in the tray menu. It names the source of the current key (credential store or environment).
- To remove the stored key, open the Windows Credential Manager, then Windows Credentials, and delete the generic credential `spellcast:anthropic-api-key`.

## Read the stored key

The Credential Manager window cannot show the key in readable form: Spellcast stores it as UTF-8 text, and the window reads it as UTF-16. Use the script instead:

```bash
powershell.exe -File scripts/windows/show-api-key.ps1          # prints the key
powershell.exe -File scripts/windows/show-api-key.ps1 -Copy    # copies it to the clipboard
```

Run it from the repo root in WSL. In a Windows terminal, use `powershell -File scripts\windows\show-api-key.ps1`.

The script only reads. The key is shown in clear text, so use `-Copy` if others can see your screen or the terminal history is kept. Any program running as your Windows user can read the key the same way. If you have lost the key and it is not stored here, create a new one in the Console: it shows a key only once.

## If something is wrong

| Message in the result window | Meaning |
|---|---|
| No API key – use the tray menu: Set API key… | Neither a stored key nor `ANTHROPIC_API_KEY` exists. |
| The API key was rejected | The key is wrong or revoked. Set a new one. |
| Rate limit reached | Too many requests. Try again in a moment. |
| Model "…" was not found | The `model` in the [config](../reference/configuration.md) is wrong or not available to your account. |

The key is never written to `config.yaml` or the log.
