package main

import "slices"

// secondAction is what the running instance does when Spellcast is started
// again, and what a first instance does with its own command line.
type secondAction int

const (
	secondNotify    secondAction = iota // show "already running" (or nothing, for the first instance)
	secondUninstall                     // --uninstall: remove the autostart entry, then quit
	secondPurge                         // --uninstall --purge: also delete the settings, the log and the API key
)

// secondInstanceAction reads a command line (args[0] is the exe path).
// Anything that is not exactly --uninstall [--purge] only shows the notice.
func secondInstanceAction(args []string) secondAction {
	if len(args) < 2 {
		return secondNotify
	}
	flags := args[1:]
	if !slices.Contains(flags, "--uninstall") {
		return secondNotify
	}
	if slices.Contains(flags, "--purge") {
		return secondPurge
	}
	return secondUninstall
}
