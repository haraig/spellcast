# Contributing to Spellcast

Thanks for your interest in Spellcast! Bug reports, ideas and code are all welcome.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://github.com/haraig/.github/blob/main/CODE_OF_CONDUCT.md). By taking part, you agree to uphold it.

## Issue or pull request?

- **Small fixes** (a typo, a docs fix, an obvious bug): open a pull request directly.
- **New features or changes in behavior:** open an issue first, with the [feature request](https://github.com/haraig/spellcast/issues/new?template=feature_request.yaml) or [bug report](https://github.com/haraig/spellcast/issues/new?template=bug_report.yaml) template. We agree on the what and the how before anyone writes code, so nobody's work is wasted.
- **Security problems:** never in a public issue. See [SECURITY.md](SECURITY.md).

## Set up

Follow [Set up the development environment](docs/how-to/set-up-dev-environment.md). On Windows you develop in WSL2; Linux and macOS work the same way, except that the Windows tests run only in CI there. Then, in your clone:

```bash
lefthook install
```

This installs the git hooks from `lefthook.yml`:

- **before each commit:** gofmt and the Markdown link check on the staged files (about a second);
- **before each push:** `task lint`, `task test`, the frontend tests and the type check.

Do not bypass them with `--no-verify`; fix the cause instead.

## Build, test, check

```bash
GOOS=windows task build   # bin/spellcast.exe
task test                 # Go tests (the Windows tests need WSL interop)
task check                # everything CI checks
```

Details: [Build and run Spellcast](docs/how-to/build-and-run.md).

## Pull requests

- **Title:** in [Conventional Commits](https://www.conventionalcommits.org/) form, for example `fix: keep the clipboard when Replace fails` or `docs: explain the deny list`. Types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `style`, `perf`, `chore`. A check enforces it.
- **Merging:** pull requests are squash-merged, so the title becomes the commit message. Your single commits inside the PR can be named freely.
- **CI** must be green: lint, the Windows tests, the portable tests on Linux and macOS, the frontend, and the security scan.
- **Docs change with the code.** A change to setup, behavior, config or a design decision updates the matching page in `docs/` in the same PR. The docs follow [Diátaxis](https://diataxis.fr): tutorials, how-to guides, reference and explanation stay on separate pages. Register new pages in [docs/README.md](docs/README.md).
- **Dependencies:** after a change to Go modules or npm packages (also in Dependabot PRs), run `task notices` and commit `THIRD_PARTY_NOTICES.txt`. CI fails if it is out of date.
- **Tests:** new behavior comes with a test. OS-specific code lives only in `internal/platform`; everything else is tested with fakes and must build on every OS.

## Style

- American English, plain and short sentences, in code, comments and docs.
- Plain ASCII punctuation: no curly quotes, em dashes or arrows.
- Go code is formatted with gofmt and passes golangci-lint (`task lint`).
- Files use LF line endings (see `.gitattributes` and `.editorconfig`).
