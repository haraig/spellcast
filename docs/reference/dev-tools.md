# Development tools and versions

Versions the project is built and tested with (2026-10-25). Newer patch versions usually work. Setup steps: [how-to](../how-to/set-up-dev-environment.md).

| Tool | Version | Source | Needed for |
|---|---|---|---|
| WSL | 2 | Windows | Linux environment on Windows |
| Ubuntu | 22.04 LTS | `wsl --install -d Ubuntu-22.04` | dev shell (zsh) |
| Git | 2.34.1 | Ubuntu package | VCS |
| Go | 1.27.3 | Homebrew on Linux (`brew install go`) | language, build (`go.mod`: `go 1.27.0`) |
| Node.js / npm | 24.16.0 / 11.16.0 | nvm 0.39.1 (`nvm install 24`) | frontend build and tests |
| Task | 3.54.0 | `go install github.com/go-task/task/v3/cmd/task@v3.54.0` | Taskfile runner used by Wails |
| Wails v3 CLI (`wails3`) | v3.0.0-beta.34 | `CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34` | app framework, build |
| golangci-lint | v2.15.0 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0` | Go lint (`task lint`, config `.golangci.yml`) |
| lefthook | v2.2.2 | `go install github.com/evilmartians/lefthook/v2@v2.2.2`, then `lefthook install` | git hooks (`lefthook.yml`) |
| actionlint | v1.7.12 | `go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12` | checks `.github/workflows/*.yml` (run `actionlint` after a workflow change) |
| govulncheck | v1.9.0 | `go install golang.org/x/vuln/cmd/govulncheck@v1.9.0` | vulnerability scan (`task check`) |
| Python | 3.10 or newer (Ubuntu: 3.10.12; Homebrew: 3.15.0) | preinstalled on Ubuntu | `scripts/check-doc-links.py` (standard library only) |
| WebView2 runtime | part of Windows 11 | Windows | result window |
| VS Code | any recent | Windows, opened from WSL with `code .` | editor |

`task`, `wails3`, `golangci-lint`, `govulncheck` and `lefthook` are installed in `~/go/bin`. The Go toolchain is pinned in `go.mod` (`toolchain go1.27.3`).

## Frontend packages
`marked` and `dompurify` (Markdown rendering and sanitizing, bundled by Vite, no CDN) are runtime dependencies. `@wailsio/runtime` is pinned to the exact version of the Wails Go module (`3.0.0-beta.34`); upgrade both together. `frontend/.npmrc` sets `min-release-age=7`: npm installs only versions published at least 7 days ago (Dependabot waits as long). TypeScript is 7.x (the Wails template's tsconfig needs 5 or later); check types with `npx tsc --noEmit` in `frontend/`.

## License notices
`scripts/gen-notices.sh` builds `THIRD_PARTY_NOTICES.txt` from `go list -deps` (Windows build) and `npm ls --omit=dev`, with the license files of the Go module cache and `node_modules`. No extra tool is needed.

## Go packages
`github.com/yuin/goldmark` (Markdown to HTML for the `CF_HTML` clipboard format on Replace) is a direct dependency.

## VS Code extensions (WSL)
`golang.go`, `editorconfig.editorconfig`.

## Git configuration (global)
| Key | Value |
|---|---|
| `user.name`, `user.email` | set |
| `includeIf "gitdir:~/develop/data/git/github.com/"` | `path = ~/.gitconfig-github` (`user.email` = the GitHub noreply address for repos below that folder) |

## Not installed, not needed
C compiler (no cgo for the Windows build), GTK/WebKit dev packages (only for a future Linux build of the app), Docker, pnpm, NSIS/MSIX/SignTool (only for later packaging and signing).
