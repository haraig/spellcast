package main

import (
	"reflect"
	"testing"
)

func labels(items []menuItem) []string {
	var out []string
	for _, it := range items {
		if !it.Separator {
			out = append(out, it.Label)
		}
	}
	return out
}

func find(items []menuItem, label string) (menuItem, bool) {
	for _, it := range items {
		if it.Label == label {
			return it, true
		}
	}
	return menuItem{}, false
}

func findID(items []menuItem, id string) bool {
	for _, it := range items {
		if it.ID == id || findID(it.Sub, id) {
			return true
		}
	}
	return false
}

func TestMenuWithoutKey(t *testing.T) {
	m := buildMenu(menuState{})
	if m[0].ID != "set-key-top" || m[0].Label != "Set API key…" || !m[0].Bold {
		t.Fatalf("first item = %+v, want the bold top Set API key entry", m[0])
	}
	if !m[1].Separator {
		t.Errorf("second item = %+v, want a separator", m[1])
	}
}

func TestMenuWithKey(t *testing.T) {
	m := buildMenu(menuState{HasKey: true})
	for _, it := range m {
		if it.ID == "set-key-top" {
			t.Fatal("top Set API key entry shown although a key is set")
		}
	}
	settings, _ := find(m, "Settings")
	if !findID(settings.Sub, "set-key") {
		t.Error("Settings has no Set API key entry")
	}
}

func TestMenuKeyFromEnvironmentHasNoTopEntry(t *testing.T) {
	// main sets HasKey for a key from ANTHROPIC_API_KEY too (the client works).
	if m := buildMenu(menuState{HasKey: true}); m[0].ID == "set-key-top" {
		t.Fatal("top Set API key entry shown for a working key")
	}
}

func TestMenuActionLines(t *testing.T) {
	m := buildMenu(menuState{HasKey: true, Actions: []actionLine{{Key: "1", Name: "Proofread"}}})
	if m[0].Label != "CapsLock+1   Proofread" || !m[0].Disabled {
		t.Errorf("action line = %+v", m[0])
	}
	m = buildMenu(menuState{HasKey: true})
	if m[0].Label != "No actions configured" || !m[0].Disabled {
		t.Errorf("empty action list = %+v", m[0])
	}
}

func TestMenuUsageOnlyWithUsage(t *testing.T) {
	if _, ok := find(buildMenu(menuState{HasKey: true}), "Usage"); ok {
		t.Error("Usage shown without usage")
	}
	u, ok := find(buildMenu(menuState{HasKey: true, UsageToday: "Today: 1", UsageMonth: "Month: 2"}), "Usage")
	if !ok {
		t.Fatal("no Usage submenu")
	}
	if got := labels(u.Sub); !reflect.DeepEqual(got, []string{"Today: 1", "Month: 2"}) {
		t.Errorf("Usage sub = %v", got)
	}
	for _, it := range u.Sub {
		if !it.Disabled {
			t.Errorf("usage line %q is clickable", it.Label)
		}
	}
}

func TestMenuOrder(t *testing.T) {
	on := true
	m := buildMenu(menuState{
		HasKey:     true,
		Actions:    []actionLine{{Key: "1", Name: "Translate"}, {Key: "2", Name: "Rewrite"}},
		UsageToday: "Today", UsageMonth: "Month",
		Autostart: &on,
	})
	want := []string{"CapsLock+1   Translate", "CapsLock+2   Rewrite", "Show last answer", "Usage", "Settings", "Help", "Quit"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("top level = %v, want %v", got, want)
	}
	settings, _ := find(m, "Settings")
	if got := labels(settings.Sub); !reflect.DeepEqual(got, []string{"Set API key…", "Open config", "Reload config", "Start with Windows"}) {
		t.Errorf("Settings = %v", got)
	}
	if it, _ := find(settings.Sub, "Start with Windows"); !it.Checkbox || !it.Checked {
		t.Errorf("Start with Windows = %+v, want a checked checkbox", it)
	}
	help, _ := find(m, "Help")
	if got := labels(help.Sub); !reflect.DeepEqual(got, []string{"Open log", "About Spellcast", "Remove from Windows…"}) {
		t.Errorf("Help = %v", got)
	}
}

func TestMenuWithoutAutostartSupport(t *testing.T) {
	settings, _ := find(buildMenu(menuState{HasKey: true}), "Settings")
	if _, ok := find(settings.Sub, "Start with Windows"); ok {
		t.Error("Start with Windows shown although the platform cannot do it")
	}
}
