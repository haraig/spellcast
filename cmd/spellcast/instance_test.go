package main

import "testing"

func TestSecondInstanceAction(t *testing.T) {
	const exe = `C:\Tools\spellcast.exe`
	tests := []struct {
		args []string
		want secondAction
	}{
		{[]string{exe}, secondNotify},
		{[]string{exe, "--uninstall"}, secondUninstall},
		{[]string{exe, "--uninstall", "--purge"}, secondPurge},
		{[]string{exe, "--purge", "--uninstall"}, secondPurge},
		{[]string{exe, "--purge"}, secondNotify},
		{[]string{exe, "--unistall"}, secondNotify},
		{[]string{exe, `C:\file.txt`}, secondNotify},
		{nil, secondNotify},
	}
	for _, tt := range tests {
		if got := secondInstanceAction(tt.args); got != tt.want {
			t.Errorf("secondInstanceAction(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}
