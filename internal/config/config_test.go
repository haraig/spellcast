package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDefaultYAML(t *testing.T) {
	c, err := Parse([]byte(DefaultYAML))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.ActionKeys(); strings.Join(got, "") != "1234" {
		t.Fatalf("action keys = %v", got)
	}
	a, _ := c.Action("1")
	if a.Model != "claude-sonnet-5-5" || a.Effort != "low" || a.MaxTokens != DefaultMaxTokens || a.Output != OutputPlain || !a.Notes {
		t.Errorf("action 1 not resolved: %+v", a)
	}
	if a, _ := c.Action("2"); a.Model != "claude-sonnet-5-5" || a.Effort != "medium" || !a.Notes {
		t.Errorf("action 2 = %+v", a)
	}
	if a, _ := c.Action("3"); a.Model != DefaultModel || a.Effort != "low" || a.Output != OutputMarkdown {
		t.Errorf("action 3 = %+v", a)
	}
	if len(c.Replace.Deny.Windows) != 4 {
		t.Errorf("deny.windows = %v", c.Replace.Deny.Windows)
	}
}

func TestParseEmptyUsesDefaults(t *testing.T) {
	c, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Model != DefaultModel || c.MaxTokens != DefaultMaxTokens {
		t.Errorf("defaults not applied: %+v", c)
	}
	if len(c.Actions) != 0 {
		t.Errorf("actions = %v", c.Actions)
	}
}

func TestParseInvalid(t *testing.T) {
	tests := []struct{ name, yaml, want string }{
		{"unknown key", "modle: x", "modle"},
		{"bad yaml", "model: [", "config:"},
		{"negative max_tokens", "max_tokens: -1", "max_tokens"},
		{"action key 0", `actions: {"0": {name: A, prompt: p}}`, "digit"},
		{"action key 10", `actions: {"10": {name: A, prompt: p}}`, "digit"},
		{"action no name", `actions: {"1": {prompt: p}}`, "name is required"},
		{"action no prompt", `actions: {"1": {name: A}}`, "prompt is required"},
		{"action bad output", `actions: {"1": {name: A, prompt: p, output: html}}`, "output"},
		{"bad effort", "effort: fast", "effort must be one of"},
		{"action bad effort", `actions: {"1": {name: A, prompt: p, effort: Low}}`, "actions.1: effort"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestPrices(t *testing.T) {
	c, err := Parse([]byte(`
prices:
  my-model: {input: 1.5, output: 7.5}
  claude-haiku-5-5: {input: 0.2, output: 1}
`))
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := c.Price("my-model"); !ok || p.Input != 1.5 || p.Output != 7.5 {
		t.Errorf("my-model = %+v, %v", p, ok)
	}
	if p, _ := c.Price("claude-haiku-5-5"); p.Input != 0.2 || p.Output != 1 {
		t.Errorf("the config replaces the default: %+v", p)
	}
	if p, ok := c.Price("claude-sonnet-5-5"); !ok || p.Input != 2 {
		t.Errorf("other defaults stay: %+v, %v", p, ok)
	}
	if _, ok := c.Price("unknown"); ok {
		t.Error("a model without a price must report none")
	}
	if len(DefaultPrices) != 3 {
		t.Error("Parse must not change the defaults")
	}
	for name, in := range map[string]string{
		"negative":   "prices: {m: {input: -1, output: 1}}",
		"empty name": `prices: {"": {input: 1, output: 1}}`,
		"typo":       "prices: {m: {inptu: 1}}",
		"not a map":  "prices: 5",
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestUnknownKeyNamesKeyAndReference(t *testing.T) {
	const ref = "https://github.com/haraig/spellcast/blob/main/docs/reference/configuration.md"
	for in, key := range map[string]string{
		"listen: x\n": `unknown key "listen" (line 1)`,
		"actions:\n  \"1\":\n    name: a\n    prompt: b\n    foo: c\n": `unknown key "foo" (line 5)`,
	} {
		_, err := Parse([]byte(in))
		if err == nil {
			t.Fatalf("%q: no error", in)
		}
		if !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), ref) {
			t.Errorf("%q: err = %q, want %q and the reference link", in, err, key)
		}
	}
}

func TestEffort(t *testing.T) {
	c, err := Parse([]byte(`
effort: medium
actions:
  "1": {name: A, prompt: p, effort: low}
  "2": {name: B, prompt: p}
`))
	if err != nil {
		t.Fatal(err)
	}
	if a, _ := c.Action("1"); a.Effort != "low" {
		t.Errorf("action 1 effort = %q", a.Effort)
	}
	if a, _ := c.Action("2"); a.Effort != "medium" {
		t.Errorf("action 2 effort = %q, want the top-level value", a.Effort)
	}
	e, _ := Parse([]byte(`actions: {"1": {name: A, prompt: p}}`))
	if a, _ := e.Action("1"); a.Effort != "" {
		t.Errorf("effort without a key = %q, want empty", a.Effort)
	}
}

func TestNotesOption(t *testing.T) {
	c, err := Parse([]byte(`
actions:
  "1": {name: A, prompt: p, notes: true}
  "2": {name: B, prompt: p}
`))
	if err != nil {
		t.Fatal(err)
	}
	if a, _ := c.Action("1"); !a.Notes {
		t.Error("notes: true not read")
	}
	if a, _ := c.Action("2"); a.Notes {
		t.Error("notes must be off by default")
	}
	d, _ := Parse([]byte(DefaultYAML))
	if a, _ := d.Action("4"); a.Name != "Look up" || a.Output != OutputMarkdown || a.Notes {
		t.Errorf("default action 4 = %+v", a)
	}
}

func TestInstructions(t *testing.T) {
	c, err := Parse([]byte(`
instructions: "  Never use em-dashes.\n"
actions:
  "1": {name: A, prompt: p}
  "2": {name: B, prompt: p, use_instructions: false}
  "3": {name: C, prompt: p, use_instructions: true}
`))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"1": "Never use em-dashes.", "2": "", "3": "Never use em-dashes."} {
		if a, _ := c.Action(key); a.Instructions != want {
			t.Errorf("action %s instructions = %q, want %q", key, a.Instructions, want)
		}
	}

	// The default file has the example commented out: no instructions.
	d, err := Parse([]byte(DefaultYAML))
	if err != nil {
		t.Fatal(err)
	}
	if a, _ := d.Action("1"); a.Instructions != "" {
		t.Errorf("default config must not set instructions, got %q", a.Instructions)
	}
}

func TestInstructionsLimit(t *testing.T) {
	ok := "instructions: " + strings.Repeat("ä", MaxInstructionsRunes) // runes, not bytes
	if _, err := Parse([]byte(ok)); err != nil {
		t.Errorf("exactly at the limit: %v", err)
	}
	_, err := Parse([]byte(ok + "x"))
	if err == nil || !strings.Contains(err.Error(), "instructions") {
		t.Errorf("over the limit: err = %v", err)
	}
}

func TestParseReportsAllErrors(t *testing.T) {
	_, err := Parse([]byte("max_tokens: -5\nactions: {\"0\": {name: A, prompt: p}}"))
	if err == nil || !strings.Contains(err.Error(), "max_tokens") || !strings.Contains(err.Error(), "digit") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Actions) != 4 {
		t.Errorf("actions = %d", len(c.Actions))
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != DefaultYAML {
		t.Fatalf("default file not written: %v", err)
	}
}

func TestLoadKeepsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("model: claude-opus-5-5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Model != "claude-opus-5-5" {
		t.Errorf("model = %q", c.Model)
	}
}
