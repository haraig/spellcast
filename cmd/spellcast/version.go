package main

import "strings"

// Set at build time with -ldflags "-X main.version=… -X main.commit=…"
// (see build/windows/Taskfile.yml). Empty in a plain go build.
var (
	version string // git describe: a tag, or the short SHA if there is no tag
	commit  string // short SHA
)

// versionText is the version line shown to the user, for example
// "Version v0.1.0 (abc1234)". Without build information it is "Version dev".
func versionText(version, commit string) string {
	if version == "" {
		return "Version dev"
	}
	if commit == "" || strings.HasPrefix(version, commit) {
		return "Version " + version
	}
	return "Version " + version + " (" + commit + ")"
}
