package spellcast

import (
	"strings"
	"testing"
)

func TestNoticesCoverBundledDependencies(t *testing.T) {
	for _, want := range []string{
		"github.com/wailsapp/wails/v3",
		"github.com/yuin/goldmark",
		"github.com/anthropics/anthropic-sdk-go",
		"Go standard library",
		"marked",
		"dompurify",
		"@wailsio/runtime",
		"Permission is hereby granted",
	} {
		if !strings.Contains(Notices, want) {
			t.Errorf("THIRD_PARTY_NOTICES.txt does not mention %q", want)
		}
	}
}
