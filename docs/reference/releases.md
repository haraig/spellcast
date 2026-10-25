# Releases

How Spellcast versions and releases are made. The workflow is `.github/workflows/release.yml`; release-please is configured in `release-please-config.json` and `.release-please-manifest.json`.

## Flow

1. Pull requests are squash-merged, so each PR title becomes one Conventional Commit on `main`.
2. On every push to `main`, [release-please](https://github.com/googleapis/release-please) checks the commits since the last release. If one of them is releasable (see below), it opens or updates the release PR `chore(main): release <version>`. That PR updates `CHANGELOG.md`, `.release-please-manifest.json` and the file and product version in `build/windows/info.json`.
3. A maintainer merges the release PR. release-please then tags the merge commit (`v<version>`) and creates the GitHub release with the changelog entry as its notes.
4. The `build` job builds the app on `windows-latest` from the tag, checks that `git describe` gives the tag, and uploads the assets.

Nothing else is done by hand.

## Version numbers

[Semantic Versioning](https://semver.org/), from the commit types:

| Commit | Release? | Bump from 1.0.0 on | Bump before 1.0.0 |
|---|---|---|---|
| `fix:`, `perf:`, `revert:`, `deps:` | yes | patch | patch |
| `feat:` | yes | minor | minor |
| breaking change (`feat!:`, `fix!:` or a `BREAKING CHANGE:` footer) | yes | major | minor (`bump-minor-pre-major`) |
| `docs:`, `chore:`, `ci:`, `build:`, `test:`, `refactor:`, `style:` | no | - | - |

The first release is `0.1.0` (`initial-version`). The largest bump of all pending commits wins. A commit starts a release only if its type has a visible section in the changelog (`changelog-sections` in `release-please-config.json`). Commits that start no release are included in the next one, but not listed in the changelog.

Dependabot uses `build(deps):` and `ci(deps):`, so dependency updates start no release of their own. To release one at once (for example a security fix), change the PR title to `fix(deps): ...` before merging. A `Release-As: x.y.z` footer in a commit message sets the version of the next release, but does not start one.

## Release notes

`CHANGELOG.md` and the GitHub release notes have one entry per commit, grouped by type: Features, Bug Fixes, Performance, Reverts, Dependencies. Each entry links its PR and commit. To change an entry, edit the PR title before merging. After merging, edit the body of the merged PR and add:

```text
BEGIN_COMMIT_OVERRIDE
fix: the corrected message
END_COMMIT_OVERRIDE
```

release-please uses the override the next time it updates the release PR.

## Assets

| File | Content |
|---|---|
| `spellcast.exe` | The program. The link `https://github.com/haraig/spellcast/releases/latest/download/spellcast.exe` always points to the latest one |
| `spellcast-<version>-windows-amd64.zip` | `spellcast.exe`, `LICENSE`, `THIRD_PARTY_NOTICES.txt` |
| `SHA256SUMS.txt` | SHA-256 of the exe and the zip, in `sha256sum` format |

The exe and the zip have a [build provenance attestation](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations) (`gh attestation verify <file> -R haraig/spellcast`). The exe is not code-signed.

## Token

release-please runs with a token of a GitHub App, because PRs and tags created with the default `GITHUB_TOKEN` trigger no workflows: the release PR would get no checks and could not be merged. The App needs the repository permissions Contents, Pull requests and Issues (read and write) and is installed on this repository only.

| Name | Kind | Value |
|---|---|---|
| `RELEASE_APP_CLIENT_ID` | repository variable | the App's client ID |
| `RELEASE_APP_PRIVATE_KEY` | repository secret | a private key of the App (PEM) |

The `build` job uploads the assets with the workflow's own `GITHUB_TOKEN`.
