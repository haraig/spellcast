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

# license_id FILE...: the SPDX IDs of the licenses in FILE..., joined with " AND ".
# An unknown license fails the script, so a person checks it before it ships.
license_id() {
	local text ids=()
	text=$(cat "$@" | tr -s '[:space:]' ' ')
	case $text in *"Apache License"*"Version 2.0"*) ids+=(Apache-2.0) ;; esac
	case $text in *"Mozilla Public License"*"2.0"*) ids+=(MPL-2.0) ;; esac
	case $text in
	*"Redistribution and use in source and binary forms"*"Neither the name"*) ids+=(BSD-3-Clause) ;;
	*"Redistribution and use in source and binary forms"*) ids+=(BSD-2-Clause) ;;
	esac
	case $text in *"Permission is hereby granted, free of charge"*) ids+=(MIT) ;; esac
	case $text in *"Permission to use, copy, modify, and/or distribute"*) ids+=(ISC) ;; esac
	if [ ${#ids[@]} -eq 0 ]; then
		echo "gen-notices: unknown license in $*" >&2
		exit 1
	fi
	local IFS=' '
	echo "${ids[*]}" | sed 's/ / AND /g'
}

# section NAME DIR [LICENSE]: LICENSE is the SPDX expression from the package
# metadata; without it, license_id reads it from the license files. A package
# without license files gets a note in their place.
# No version in the header: a version bump leaves the file unchanged.
section() {
	local files license
	files=$(license_files "$2")
	if [ -z "$files" ] && [ -z "${3:-}" ]; then
		echo "gen-notices: no license file for $1 in $2" >&2
		exit 1
	fi
	license=${3:-}
	if [ -z "$license" ]; then
		mapfile -t list <<<"$files"
		license=$(license_id "${list[@]}")
	fi
	{
		printf '%s\n' "================================================================================"
		printf '%s (%s)\n' "$1" "$license"
		printf '%s\n\n' "================================================================================"
		if [ -z "$files" ]; then
			printf '%s\n\n' "The package ships no license file, only the license field \"$license\"."
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
	echo "Each entry names the component and its license (SPDX ID), followed by the license text."
	echo
} >"$tmp"

goroot=$(go env GOROOT)
# Homebrew keeps the Go LICENSE one level above GOROOT.
[ -n "$(license_files "$goroot")" ] || goroot=$(dirname "$goroot")
section "Go standard library" "$goroot"

GOOS=windows go list -deps -tags production \
	-f '{{with .Module}}{{if not .Main}}{{.Path}}	{{.Dir}}{{end}}{{end}}' \
	./cmd/spellcast | sort -u |
	while IFS=$'\t' read -r path dir; do
		section "$path" "$dir"
	done

npm --prefix frontend ls --omit=dev --all --parseable 2>/dev/null | tail -n +2 | sort |
	while read -r dir; do
		name=$(node -p "require('$dir/package.json').name")
		# The SPDX expression without outer parentheses: "(MPL-2.0 OR Apache-2.0)".
		license=$(node -p "(require('$dir/package.json').license || '').replace(/^\\((.*)\\)$/, '\$1')")
		section "$name" "$dir" "$license"
	done

mv "$tmp" "$out"
trap - EXIT
echo "wrote $out"
