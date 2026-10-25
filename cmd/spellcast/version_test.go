package main

import "testing"

func TestVersionText(t *testing.T) {
	tests := []struct{ version, commit, want string }{
		{"v0.1.0", "abc1234", "Version v0.1.0 (abc1234)"},
		{"", "", "Version dev"},
		{"abc1234-dirty", "abc1234", "Version abc1234-dirty"},
		{"v0.1.0", "", "Version v0.1.0"},
	}
	for _, tt := range tests {
		if got := versionText(tt.version, tt.commit); got != tt.want {
			t.Errorf("versionText(%q, %q) = %q, want %q", tt.version, tt.commit, got, tt.want)
		}
	}
}
