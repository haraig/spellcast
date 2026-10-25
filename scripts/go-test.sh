#!/usr/bin/env sh
# Runs the Go tests. The app is Windows-only so far: with WSL interop the
# Windows test programs run directly (GOOS=windows). Without interop (native
# Linux or macOS, or SPELLCAST_NO_INTEROP=1) it vets the Windows build and
# tests only the packages that build everywhere; CI runs the Windows tests.
set -eu

PORTABLE="./internal/core ./internal/config ./internal/platform ./internal/claude ./internal/usage"

if [ -e /proc/sys/fs/binfmt_misc/WSLInterop ] && [ -z "${SPELLCAST_NO_INTEROP:-}" ]; then
	GOOS=windows go test "$@" ./...
else
	GOOS=windows go vet ./...
	# shellcheck disable=SC2086 # PORTABLE is a list of packages
	GOOS= go test "$@" $PORTABLE
	echo "Windows tests skipped (no WSL interop); CI runs them on windows-latest."
fi
