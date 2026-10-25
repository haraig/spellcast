# How to set up the development environment

Goal: a machine that builds, tests and runs Spellcast. On Windows, you develop in WSL2 (Ubuntu): the Windows app is cross-compiled there and runs on Windows through WSL interop. On Linux and macOS you work in the same way, in a normal terminal. Versions are listed in [dev-tools](../reference/dev-tools.md).

Only the Windows app exists today. The commands below therefore build for Windows (`GOOS=windows`).

## 1. Install WSL2 and Ubuntu (Windows only)

In a Windows terminal:

```
wsl --install -d Ubuntu-22.04
```

Restart if asked, then open Ubuntu and create your Linux user. Everything below runs in the Ubuntu shell (bash or zsh).

## 2. Install Go and Node.js

Any installation works if `go` and `node` are Linux binaries on your PATH. One working option, without sudo:

- Go from [Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux): `brew install go`. The Homebrew installer prints an `eval "$(.../brew shellenv)"` line; put it in `~/.zprofile` (zsh) or `~/.profile` (bash), or `go` is missing in new shells.
- Node.js 24 from [nvm](https://github.com/nvm-sh/nvm): `nvm install 24`.

Make sure `~/go/bin` is on your PATH, before the Windows folders (add `export PATH="$HOME/go/bin:$PATH"` to `~/.zshrc` or `~/.bashrc`).

## 3. Install Task and the Wails CLI

```bash
go install github.com/go-task/task/v3/cmd/task@v3.54.0
CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34
```

- `CGO_ENABLED=0` is required: without it the install fails with `exec: "pkg-config": executable file not found`. The cgo-free `wails3` can still generate icons, the `.syso` resource and bindings, build, and run `wails3 dev`.
- Use the same Wails version as `go.mod`. Wails v3 is still a beta, so versions can change behavior.

Then the check tools (lint, vulnerability scan):

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
```

And the git hooks (lefthook), installed once per clone after step 5:

```bash
go install github.com/evilmartians/lefthook/v2@v2.2.2
lefthook install   # in the repo
```

The hooks are defined in `lefthook.yml`. Before each commit they check gofmt and the Markdown links of the staged files (about a second). Before each push they run `task lint`, `task test` and the frontend tests and type check. Do not bypass them with `--no-verify`; fix the cause instead.

`go.mod` names the Go toolchain (`toolchain go1.27.3`); an older local Go downloads it automatically.

## 4. Configure Git

```bash
git config --global user.name "Your Name"          # required for commits
git config --global user.email "you@example.com"
```

Optional: to keep your private email address out of public commits, use your GitHub noreply address (`<id>+<username>@users.noreply.github.com`, shown under GitHub **Settings → Emails**) for every repo below one folder:

```bash
printf '[user]\n\temail = <id>+<username>@users.noreply.github.com\n' > ~/.gitconfig-github
git config --global 'includeIf.gitdir:~/code/github.com/.path' '~/.gitconfig-github'
git config user.email   # inside a repo below that folder: prints the noreply address
```

On GitHub, also turn on **Keep my email addresses private** and **Block command line pushes that expose my email**.

Line endings are set by the repo's `.gitattributes` (LF everywhere), so no `core.autocrlf` setting is needed.

## 5. Clone the repo

Clone into the Linux filesystem (for example `~/develop/...`), not under `/mnt/c`: builds on `/mnt/c` are much slower.

## 6. Install VS Code extensions (optional)

Open the folder from WSL with `code .` (VS Code installs its WSL server on first use), then install `golang.go` and `editorconfig.editorconfig` in the WSL window. Accept the prompt to install the Go tools (gopls, staticcheck).

## 7. Verify

In a new shell:

```bash
command -v task wails3 go node   # task and wails3 in ~/go/bin; go and node must be Linux paths
task --version; wails3 version; go version; node -v
```

Then build and test (see [Build and run](build-and-run.md)):

```bash
GOOS=windows task build          # bin/spellcast.exe
GOOS=windows go vet ./...
GOOS=windows go test ./...       # the Windows test programs run through interop
(cd frontend && npm test && npx tsc --noEmit)
task check                       # everything CI checks: lint, tests, frontend, links, vulnerabilities
```

## Working with the Windows app from WSL

- **Build, test, dev mode:** always set `GOOS=windows`. `go test` runs the Windows `.test.exe` files directly through interop; no `-exec` wrapper is needed. (`task lint`, `task test` and `task check` set it themselves.)
- **Without WSL (native Linux or macOS):** the Windows test programs cannot run. `task test` then vets the Windows build and tests only the packages that build everywhere; CI runs the Windows tests on `windows-latest`.
- **Dev mode:** `GOOS=windows WSLENV=FRONTEND_DEVSERVER_URL task dev`. Vite runs in WSL, and the Windows dev exe loads the frontend from it (CSS changes reload at once, Go changes rebuild and restart the exe). Stopping `task dev` also stops the Windows process.
- **Environment variables for the exe:** Windows programs see only the variables listed in `WSLENV`, for example `ANTHROPIC_BASE_URL=http://127.0.0.1:18080 WSLENV=ANTHROPIC_BASE_URL ./bin/spellcast.exe`.
- **Stop the app:** use **Quit** in the tray menu, or `taskkill.exe /F /IM spellcast.exe`. Plain `taskkill` without `/F` does not stop it.
- **Windows-side commands:** call them with `.exe` (`taskkill.exe`, `tasklist.exe`, `powershell.exe`).
- **Temp directory:** keep Go's temp dir on Linux (the default). With `GOTMPDIR` on `/mnt/c`, the Windows tests take twice as long.
- **Networking:** WSL runs in the default NAT mode. Windows programs reach services in WSL on `127.0.0.1`, but Linux programs cannot reach Windows services on `127.0.0.1`.

## Troubleshooting
- **A command finds a Windows tool instead of the Linux one:** WSL appends the Windows PATH, so a Windows `go`, `node` or `task` can shadow a missing Linux one. Check with `command -v`.
- **Disk space:** the Go module and build caches grow to several GB (about 7 GB after the first builds), and the WSL disk (`ext4.vhdx` on C:) does not shrink by itself. `go clean -cache` frees the build cache.
