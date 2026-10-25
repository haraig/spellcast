<h1><img src="build/appicon.png" width="48" alt="" align="top"> Spellcast</h1>

[![CI](https://github.com/haraig/spellcast/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/haraig/spellcast/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/haraig/spellcast)](https://github.com/haraig/spellcast/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Claude, right where you are typing.**

Select text in any app, press <kbd>CapsLock</kbd> + <kbd>1</kbd>, and the answer appears next to your text cursor. Translate, rewrite, explain or look up a word, then put the result back in place with one click. No copy and paste, no switching to another window.

<!-- demo GIF: select a sentence in Outlook or Notepad, press CapsLock+1, the answer streams in next to the cursor, click Replace -->

- **Works in every app:** mail, browser, editor, chat, terminal.
- **Your own actions:** up to nine prompts on <kbd>CapsLock</kbd> + <kbd>1</kbd> to <kbd>9</kbd>, in a plain config file.
- **Replace in place**, or copy. **Show changes** highlights what was edited.
- **More than a translator app:** anything you can describe in a prompt.

> **You need your own Claude API key.** Spellcast calls the Claude API with your key and you pay per request. You can start with as little as $5 of credit (at the time of writing). With the default model, a request usually costs well under a cent, so that lasts for thousands of requests. The tray menu shows what you have spent, see [Check what Spellcast costs](docs/how-to/check-your-spending.md).

## Install

**Windows 10 and 11.** Download [`spellcast.exe`](https://github.com/haraig/spellcast/releases/latest/download/spellcast.exe) from the [latest release](https://github.com/haraig/spellcast/releases/latest), put it in a folder of your choice, and start it. There is no installer. The exe is not code-signed, so Windows SmartScreen may warn on the first start: choose **More info** > **Run anyway**. Details, including how to check the download: [Install Spellcast](docs/how-to/install-spellcast.md). To update, see [Update Spellcast](docs/how-to/update-spellcast.md).

Installing with winget/UniGetUI is planned. You can also [build Spellcast from source](docs/how-to/build-and-run.md).

**macOS:** planned (Homebrew).

Requirements: Windows 10 or 11 with the WebView2 runtime (preinstalled on Windows 11), and a Claude API key.

## Quick start

1. **Get an API key.** Create an account in the [Claude Console](https://console.anthropic.com/), buy credit ([how billing works](https://support.claude.com/en/articles/8977456-how-do-i-pay-for-my-api-usage)) and create an API key.
2. **Start Spellcast.** It has no window of its own: look for the wand icon in the tray. On the first start, a notification says that Spellcast is running. Starting it again while it runs shows "Spellcast is already running".
3. **Set the key.** Click the tray icon, choose **Set API key…** (the first entry while no key is set), paste the key. See [Set the API key](docs/how-to/set-the-api-key.md).
4. **Try it.** Select a sentence in any app and press <kbd>CapsLock</kbd> + <kbd>1</kbd> (Translate). Click **Replace** to put the answer in place of your selection, or **Copy**. <kbd>Esc</kbd> closes the window.
5. **Optional:** tick **Settings** > **Start with Windows** in the tray menu.

New to Spellcast? Follow [Your first 10 minutes](docs/tutorials/first-ten-minutes.md). More: [Keyboard shortcuts](docs/reference/keyboard-shortcuts.md), [Troubleshooting](docs/how-to/troubleshoot.md), [Use the built-in hotkeys](docs/how-to/use-the-built-in-hotkeys.md), [Replace the selected text](docs/how-to/replace-the-selected-text.md), [See what the answer changed](docs/how-to/see-what-changed.md), [Check what Spellcast costs](docs/how-to/check-your-spending.md).

## Configure your actions

Each digit is one action: a name and a prompt. The defaults are Translate, Rewrite, Explain and Look up. Tray menu, **Settings** > **Open config**, change or add actions, then **Settings** > **Reload config**:

```yaml
actions:
  "5":
    name: Formal
    prompt: >-
      Rewrite the text in a polite, formal tone. Keep the language and the meaning.
      Output only the rewritten text.
```

All keys, models and prices: [Configuration reference](docs/reference/configuration.md).

## Privacy and security

- **What leaves your computer:** only the selected text and the action's prompt, sent to the Claude API when you press a hotkey. Nothing else. Spellcast opens no network port.
- **What Anthropic does with it:** Spellcast uses your own API key, so your text is covered by Anthropic's Commercial Terms for your API account. By default, Anthropic does not use API inputs and outputs to train its models, and it deletes them within 30 days. Longer retention applies if a request is flagged for a Usage Policy violation, or if the law requires it. The author of Spellcast never sees your text. Details: [Is my data used for model training?](https://privacy.claude.com/en/articles/7996868-is-my-data-used-for-model-training) and [How long do you store my organization's data?](https://privacy.claude.com/en/articles/7996866-how-long-do-you-store-my-organization-s-data) (Anthropic Privacy Center; check there for the current terms).
- **The keyboard hook** only reacts to <kbd>CapsLock</kbd> + digit. It does not store or log keys. Details: [The keyboard hook and security](docs/explanation/keyboard-hook-and-security.md).
- **Your API key** is stored in the Windows Credential Manager, not in a file. The log never contains the key or your text.
- Found a security problem? See [SECURITY.md](SECURITY.md).

## For developers

Spellcast is written in Go with [Wails v3](https://v3.wails.io/) (system tray and a WebView2 window) and a small TypeScript frontend. It talks to Claude through the official Anthropic Go SDK.

- How it works: [Architecture overview](docs/explanation/architecture.md), [How a request works](docs/explanation/how-a-request-works.md), [How Replace works](docs/explanation/how-replace-works.md), [How the result window is placed](docs/explanation/result-window-placement.md)
- Build from source: [Set up the development environment](docs/how-to/set-up-dev-environment.md) (WSL2 on Windows), then [Build and run](docs/how-to/build-and-run.md)
- All documentation: [docs/](docs/README.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)

## License

[MIT](LICENSE). The licenses of the bundled third-party software are in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt) and in the app: tray menu, **Help** > **About Spellcast**, **Third-party licenses**.
