# AGENTS.md

Instructions for coding agents working in this repo. People start with [CONTRIBUTING.md](CONTRIBUTING.md); this file follows the same rules.

## What Spellcast is

A Windows tray app in Go with [Wails v3](https://v3.wails.io/) and a small TypeScript frontend. Select text in any app, press CapsLock + a digit, and Claude processes the text with the prompt of that action. Overview: [Architecture](docs/explanation/architecture.md).

## Commands

The app is Windows-only so far. Develop in WSL2 (or on Linux or macOS) and build for Windows. The Windows exe and the Windows tests run through WSL interop.

```bash
GOOS=windows task build   # bin/spellcast.exe
task lint                 # golangci-lint for the Windows build (includes go vet)
task test                 # Go tests, see scripts/go-test.sh
task check                # everything CI checks: run it before you say a change is done
task notices              # after any change to Go modules or npm packages
```

- Pass environment variables to the Windows exe with `WSLENV` (for example `WSLENV=ANTHROPIC_BASE_URL`).
- Stop the running app with `taskkill.exe /F /IM spellcast.exe`. Plain `taskkill` does not stop it.
- Install Go tools with `go install` and a pinned version. No sudo.

## Layout

| Path | Content |
|---|---|
| `cmd/spellcast` | `main`, the tray menu, wiring |
| `internal/core` | Turns a trigger into a request and a view; no OS code |
| `internal/claude` | Claude API client (official Go SDK, streaming), system prompt, API key |
| `internal/config` | `config.yaml`: parse, defaults, validation; the default file is `default.go` |
| `internal/platform` | **All OS-specific code** (hotkeys, capture, Replace, window placement) |
| `internal/ui` | The Wails side of the windows (result window, API key window) |
| `internal/usage` | Token and cost totals |
| `frontend/` | Result window (TypeScript), with its own tests |
| `docs/` | Documentation, see below |
| `scripts/` | Test, notices and doc-link scripts |

## Rules

- **OS-specific code only in `internal/platform`.** Everything else builds on every OS and is tested with fakes.
- **New behavior comes with a test.**
- **Docs change with the code.** A change to setup, behavior, config or a design decision updates the matching page in `docs/` in the same change. If you install a tool, update [Set up the development environment](docs/how-to/set-up-dev-environment.md) and [Dev tools](docs/reference/dev-tools.md).
- **Default config:** `internal/config/default.go` and the copy in [Configuration](docs/reference/configuration.md#default-config) stay identical.
- **Docs follow [Diátaxis](https://diataxis.fr):** `docs/tutorials/`, `docs/how-to/`, `docs/reference/` and `docs/explanation/` stay separate; never mix two types on one page. Register new pages in [docs/README.md](docs/README.md).
- **Current state only:** docs and code comments describe what is, not what changed ("now", "no longer", "replaces the old"). History belongs in commit messages.
- **Style:** American English, plain and short sentences. Plain ASCII punctuation: no curly quotes, em dashes or arrows. LF line endings. gofmt and golangci-lint for Go.
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/) titles (`fix: ...`, `docs: ...`). Do not bypass the git hooks with `--no-verify`.

## Don'ts

- Do not commit or push unless asked.
- Do not log or store the user's selected text or API key.
- Do not add sampling parameters (`temperature`, `top_p`, `top_k`) to API requests; current models reject them.
