# Architecture overview

How the parts of Spellcast fit together. The pages linked at the end explain each part in detail.

## Components

```mermaid
flowchart LR
    hook["Keyboard hook<br/>CapsLock + digit"] --> core
    tray["Tray menu"] --> core
    config[("config.yaml")] --> core
    core["Core<br/>sessions, two-part split"] --> claude["Claude client<br/>streaming"]
    claude --> api(("Claude API"))
    core --> window["Result window<br/>WebView2 frontend"]
    window -- "Replace, Copy, Close" --> core
    core --> usage[("usage.json")]
    subgraph platform["platform (Windows-only code)"]
        capture["Capture selection"]
        replace["Replace / clipboard"]
        place["Window placement"]
    end
    core --> capture
    core --> replace
    window --> place
```

- The **core** is OS-neutral. It talks to the operating system only through the `platform.Platform` interface and to the window only through events. This keeps it testable with fakes.
- Everything that calls Win32 lives in `internal/platform`. A macOS port would add a second implementation of the same interface.
- The **frontend** is one page with three views: the result window, the API key window and the About window. It gets data from Go as events and never calls Go bindings.

## One request

```mermaid
sequenceDiagram
    actor You
    participant Hook as Keyboard hook
    participant Core
    participant Platform
    participant Claude as Claude API
    participant Window
    You->>Hook: CapsLock + 1
    Hook->>Core: trigger "1"
    Core->>Platform: capture the selection (Ctrl+Insert, clipboard restored)
    Platform-->>Core: text and source window
    Core->>Window: new session (action, source)
    Core->>Claude: system prompt + text (streaming)
    loop while the answer arrives
        Claude-->>Core: text pieces
        Core-->>Window: pieces (answer, then notes)
    end
    Claude-->>Core: done (tokens)
    Core-->>Window: done, cost line
    You->>Window: Replace (Ctrl+Enter)
    Window->>Core: replace
    Core->>Platform: paste into the source window, restore the clipboard
```

## Packages

| Package | Responsibility | Details |
|---|---|---|
| `cmd/spellcast` | Wiring: tray menu, notifications, single instance, logging, version, cleanup | |
| `internal/core` | Triggers, sessions, cancelling, two-part answers, Replace text | [How a request works](how-a-request-works.md) |
| `internal/claude` | Claude client, prompts, API key storage, readable errors | [How a request works](how-a-request-works.md) |
| `internal/config` | Loading and validating `config.yaml`, defaults, prices | [Configuration](../reference/configuration.md) |
| `internal/platform` | Keyboard hook, capture, Replace, clipboard, placement, autostart (Windows) | [Keyboard hook](keyboard-hook-and-security.md), [Replace](how-replace-works.md), [Placement](result-window-placement.md) |
| `internal/ui` | The Wails windows and their events | |
| `internal/usage` | Daily totals in `usage.json` | [Check what Spellcast costs](../how-to/check-your-spending.md) |
| `frontend` | TypeScript views: answer rendering, diff, API key, About | |
