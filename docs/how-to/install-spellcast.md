# How to install Spellcast

Spellcast is a single `spellcast.exe` with no installer. You need Windows 10 or 11 with the WebView2 runtime (preinstalled on Windows 11) and a Claude API key.

## Download

Open the [latest release](https://github.com/haraig/spellcast/releases/latest) and download one of these:

- `spellcast.exe`: the program itself.
- `spellcast-<version>-windows-amd64.zip`: the same exe, with `LICENSE` and `THIRD_PARTY_NOTICES.txt`.

## Check the download (optional)

Compare the SHA-256 checksum with the line for your file in `SHA256SUMS.txt` from the same release:

```powershell
(Get-FileHash .\spellcast.exe -Algorithm SHA256).Hash.ToLower()
```

Or check that the file was built by this repository's release workflow, with the [GitHub CLI](https://cli.github.com/):

```powershell
gh attestation verify .\spellcast.exe -R haraig/spellcast
```

## Put it in place

1. Create a folder for it, for example `%LocalAppData%\Programs\Spellcast`, and move `spellcast.exe` there (unpack it first if you downloaded the zip).
2. Double-click `spellcast.exe`. The exe is not code-signed, so Windows SmartScreen may show "Windows protected your PC". Choose **More info** > **Run anyway**. Windows asks only once per file.
3. Look for the wand icon in the tray, then set your API key: see [Set the API key](set-the-api-key.md).

Keep the exe where it is once you have turned on **Start with Windows**: the autostart entry stores its full path, see [Start Spellcast with Windows](start-spellcast-with-windows.md).

Next: [Your first 10 minutes with Spellcast](../tutorials/first-ten-minutes.md). To update later, see [Update Spellcast](update-spellcast.md).
