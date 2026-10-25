# How to build and run spellcast

Prerequisite: [the development environment](set-up-dev-environment.md). Run all commands in the repo root, in WSL (or on Linux or macOS).

## Build

```bash
GOOS=windows task build
```

This runs `go mod tidy`, generates icons and bindings, builds the frontend (`npm install`, `vite build`), generates the Windows resource file (`.syso`) and compiles `cmd/spellcast` to `bin/spellcast.exe`. The `.syso` file is removed again after the build.

The build writes the version into the exe: `git describe --tags --always --dirty` (a tag, or the short commit SHA if there is no tag) and the short SHA. **Help** > **About Spellcast** and the first log line show them. Without Git, the version is `dev`. Release builds are made by the release workflow from a tag, see [Releases](../reference/releases.md).

`THIRD_PARTY_NOTICES.txt` (the licenses of all bundled Go modules and npm packages) is embedded in the exe and shown in the About window. After a dependency change, regenerate it with `task notices` (`scripts/gen-notices.sh`) and commit it.

## Run

```bash
GOOS=windows task run                               # runs task build, then starts bin/spellcast.exe
GOOS=windows WSLENV=FRONTEND_DEVSERVER_URL task dev   # dev mode with frontend hot reload (wails3 dev)
```

In dev mode, `wails3 dev` builds a dev exe and starts it with `task run:built` (run without a new build). The exe loads the frontend from the Vite dev server in WSL; `WSLENV` passes the server's address to the Windows process.

spellcast has no console window. It shows a tray icon; use the tray menu entry **Quit** to stop it, or `taskkill.exe /F /IM spellcast.exe`.

## Test

```bash
GOOS=windows go vet ./...
GOOS=windows go test ./...   # the Windows test programs run through WSL interop
```

## Check before you push

```bash
task check
```

It runs what CI runs: golangci-lint (includes `go vet`), the Go tests, the frontend tests and type check, the Markdown link check (`scripts/check-doc-links.py`), govulncheck, `npm audit` and `go mod tidy -diff`. `task lint` and `task test` run single parts. All of them build `frontend/dist` first if it is missing.

## Notes
- `frontend/dist` is embedded by `assets.go`. On a fresh clone `go vet` and `go build` fail until the frontend was built once (`GOOS=windows task build`, or only the frontend with `task frontend:dist`; both need `wails3`, which generates `frontend/bindings` first).
- `frontend/bindings` and `bin/` are generated and not committed.
- Frontend unit tests (the word diff): `npm test` in `frontend/` (Node's built-in test runner, Node 24). Type check: `npx tsc --noEmit`.
- Always build with `task build` (or `task run`), never with plain `go build -o bin/...`: only `task build` embeds the icon and manifest resource. Without it, notifications fail with `Error saving icon`.
