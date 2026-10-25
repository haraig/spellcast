#!/usr/bin/env bash
# Writes THIRD_PARTY_NOTICES.txt: the license texts of everything linked into
# bin/spellcast.exe (Go standard library, Go modules of the Windows build) and
# of the npm packages bundled into the frontend. The output is sorted and has
# no dates, so a rerun without dependency changes gives no diff.
# Needs: the Go module cache (go mod download) and frontend/node_modules (npm ci).
set -euo pipefail
export LC_ALL=C # the same sort order on every machine
cd "$(dirname "$0")/.."

if [ ! -d frontend/node_modules ]; then
	echo "gen-notices: frontend/node_modules is missing; run 'npm --prefix frontend ci' first" >&2
	exit 1
fi
out=THIRD_PARTY_NOTICES.txt
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

# license_files DIR: the license and notice files in the top level of DIR.
license_files() {
	find -L "$1" -maxdepth 1 -type f | { grep -iE '/(licen[cs]e|copying|notice)([._-][^/]*)?$' || true; } | sort
}

# section NAME VERSION DIR [FALLBACK]: FALLBACK is printed when DIR has no license file.
section() {
	local files
	files=$(license_files "$3")
	if [ -z "$files" ] && [ -z "${4:-}" ]; then
		echo "gen-notices: no license file for $1 in $3" >&2
		exit 1
	fi
	{
		printf '%s\n' "================================================================================"
		printf '%s %s\n' "$1" "$2"
		printf '%s\n\n' "================================================================================"
		if [ -z "$files" ]; then
			printf '%s\n\n' "$4"
		fi
		while read -r f; do
			[ -n "$f" ] || continue
			cat "$f"
			printf '\n'
		done <<<"$files"
	} >>"$tmp"
}

{
	echo "Spellcast includes the following third-party software."
	echo "Each entry names the component and its version, followed by its license."
	echo
} >"$tmp"

goroot=$(go env GOROOT)
# Homebrew keeps the Go LICENSE one level above GOROOT.
[ -n "$(license_files "$goroot")" ] || goroot=$(dirname "$goroot")
# The version from go.mod's toolchain line, not the local Go: the same file on every machine.
gover=$(sed -n 's/^toolchain //p' go.mod)
section "Go standard library" "${gover:-$(go env GOVERSION)}" "$goroot"

GOOS=windows go list -deps -tags production \
	-f '{{with .Module}}{{if not .Main}}{{.Path}}	{{.Version}}	{{.Dir}}{{end}}{{end}}' \
	./cmd/spellcast | sort -u |
	while IFS=$'\t' read -r path version dir; do
		section "$path" "$version" "$dir"
	done

npm --prefix frontend ls --omit=dev --all --parseable 2>/dev/null | tail -n +2 | sort |
	while read -r dir; do
		name=$(node -p "require('$dir/package.json').name")
		version=$(node -p "require('$dir/package.json').version")
		license=$(node -p "require('$dir/package.json').license || ''")
		# Some packages (for example @wailsio/runtime) ship no license file, only the field.
		section "$name" "$version" "$dir" "${license:+License: $license (the package ships no license file).}"
	done

mv "$tmp" "$out"
trap - EXIT
echo "wrote $out"
