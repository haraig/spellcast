// Package config loads and validates config.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	DefaultModel     = "claude-haiku-5-5"
	DefaultMaxTokens = 4096

	// MaxInstructionsRunes caps "instructions": they are sent with every
	// request, so a paste mistake must not burn tokens.
	MaxInstructionsRunes = 8000

	OutputPlain    = "plain"
	OutputMarkdown = "markdown"
)

// Efforts are the valid values of "effort". Empty means the model's default.
var Efforts = []string{"low", "medium", "high", "xhigh", "max"}

// Price is the price of a model in USD per million tokens. A zero cache price
// means that kind of token is not priced (Spellcast does not use prompt caching).
type Price struct {
	Input      float64 `yaml:"input"`
	Output     float64 `yaml:"output"`
	CacheRead  float64 `yaml:"cache_read"`
	CacheWrite float64 `yaml:"cache_write"`
}

// DefaultPrices are the prices of the default models as of 2026-10-06 (Haiku
// 5.5: for prompts up to 100K tokens). Prices change: check the Claude
// Console and override them in "prices".
var DefaultPrices = map[string]Price{
	"claude-haiku-5-5":  {Input: 0.10, Output: 0.50},
	"claude-sonnet-5-5": {Input: 2, Output: 10, CacheRead: 0.20},
	"claude-opus-5-5":   {Input: 4, Output: 20, CacheRead: 0.20},
}

// Config is the content of config.yaml after defaults are applied.
type Config struct {
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
	// Effort is the default effort level; empty leaves it to the model.
	Effort string `yaml:"effort"`
	// Instructions are general rules added to the system prompt of every
	// action that does not opt out with use_instructions: false.
	Instructions string `yaml:"instructions"`
	// Prices by model id for the cost estimate (USD per million tokens). An
	// entry replaces the built-in default of that model; other defaults stay.
	Prices  map[string]Price  `yaml:"prices"`
	Actions map[string]Action `yaml:"actions"`
	Replace Replace           `yaml:"replace"`
}

// Action is one entry under "actions". The map key ("1".."9") is the digit
// that triggers it. Model and MaxTokens are always set after Parse.
type Action struct {
	Name      string `yaml:"name"`
	Prompt    string `yaml:"prompt"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
	Effort    string `yaml:"effort"`
	Output    string `yaml:"output"`
	// UseInstructions opts out of the general instructions when false
	// (default true).
	UseInstructions *bool `yaml:"use_instructions"`
	// Notes switches on two-part answers: the usable text, a line with only
	// "---", then notes for the reader. Off by default.
	Notes bool `yaml:"notes"`

	// Instructions is the general text that applies to this action: the
	// trimmed top-level "instructions", or empty if the action opted out.
	// Set by Parse; not read from the file.
	Instructions string `yaml:"-"`
}

// Replace holds the per-OS deny lists for the Replace feature.
type Replace struct {
	Deny Deny `yaml:"deny"`
}

// Deny lists apps where Replace is disabled. Windows: exe names; macOS: bundle IDs.
type Deny struct {
	Windows []string `yaml:"windows"`
	Darwin  []string `yaml:"darwin"`
}

// Dir returns the spellcast config directory (os.UserConfigDir()/spellcast).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "spellcast"), nil
}

// DefaultPath returns the path of config.yaml.
func DefaultPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config at path. If the file does not exist, it is created
// from DefaultYAML first.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(DefaultYAML), 0o644); err != nil {
			return nil, err
		}
		data = []byte(DefaultYAML)
	} else if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse decodes and validates YAML. Unknown keys are an error, so typos are caught.
func Parse(data []byte) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, decodeError(err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	c.Instructions = strings.TrimSpace(c.Instructions)
	prices := make(map[string]Price, len(DefaultPrices)+len(c.Prices))
	for m, p := range DefaultPrices {
		prices[m] = p
	}
	for m, p := range c.Prices {
		prices[m] = p
	}
	c.Prices = prices
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = DefaultMaxTokens
	}
	if c.Replace.Deny.Windows == nil {
		c.Replace.Deny.Windows = []string{"WindowsTerminal.exe", "conhost.exe", "wezterm-gui.exe", "mintty.exe"}
	}
	if c.Replace.Deny.Darwin == nil {
		c.Replace.Deny.Darwin = []string{"com.apple.Terminal", "com.googlecode.iterm2"}
	}
	for k, a := range c.Actions {
		if a.Model == "" {
			a.Model = c.Model
		}
		if a.MaxTokens == 0 {
			a.MaxTokens = c.MaxTokens
		}
		if a.Effort == "" {
			a.Effort = c.Effort
		}
		if a.Output == "" {
			a.Output = OutputPlain
		}
		if a.UseInstructions == nil || *a.UseInstructions {
			a.Instructions = c.Instructions
		}
		c.Actions[k] = a
	}
}

func (c *Config) validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if c.MaxTokens < 0 {
		add("max_tokens must be positive, got %d", c.MaxTokens)
	}
	if c.Effort != "" && !slices.Contains(Efforts, c.Effort) {
		add("effort must be one of %s, got %q", strings.Join(Efforts, ", "), c.Effort)
	}
	if n := utf8.RuneCountInString(c.Instructions); n > MaxInstructionsRunes {
		add("instructions: %d characters, the limit is %d", n, MaxInstructionsRunes)
	}
	for m, p := range c.Prices {
		if strings.TrimSpace(m) == "" {
			add("prices: model id must not be empty")
		}
		if p.Input < 0 || p.Output < 0 || p.CacheRead < 0 || p.CacheWrite < 0 {
			add("prices.%s: prices must not be negative", m)
		}
	}
	for _, k := range c.ActionKeys() {
		a := c.Actions[k]
		if len(k) != 1 || k[0] < '1' || k[0] > '9' {
			add("actions: key %q must be a digit from 1 to 9", k)
			continue
		}
		if strings.TrimSpace(a.Name) == "" {
			add("actions.%s: name is required", k)
		}
		if strings.TrimSpace(a.Prompt) == "" {
			add("actions.%s: prompt is required", k)
		}
		if a.MaxTokens < 0 {
			add("actions.%s: max_tokens must be positive, got %d", k, a.MaxTokens)
		}
		if a.Effort != "" && !slices.Contains(Efforts, a.Effort) {
			add("actions.%s: effort must be one of %s, got %q", k, strings.Join(Efforts, ", "), a.Effort)
		}
		if a.Output != OutputPlain && a.Output != OutputMarkdown {
			add("actions.%s: output must be %q or %q, got %q", k, OutputPlain, OutputMarkdown, a.Output)
		}
	}
	return errors.Join(errs...)
}

// ActionKeys returns the configured action keys in ascending order.
func (c *Config) ActionKeys() []string {
	keys := make([]string, 0, len(c.Actions))
	for k := range c.Actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Price returns the price of a model, if there is one.
func (c *Config) Price(model string) (Price, bool) {
	p, ok := c.Prices[model]
	return p, ok
}

// Action returns the action for a digit, with model, max_tokens and effort resolved.
func (c *Config) Action(key string) (Action, bool) {
	a, ok := c.Actions[key]
	return a, ok
}

// ReferenceURL is the page that lists every config key.
const ReferenceURL = "https://github.com/haraig/spellcast/blob/main/docs/reference/configuration.md"

// unknownField matches yaml.v3's message for a key that the Config type does not have.
var unknownField = regexp.MustCompile(`^line (\d+): field (\S+) not found in type `)

// decodeError turns yaml.v3's decode errors into messages for users: an
// unknown key is named with its line, followed by a link to the reference.
func decodeError(err error) error {
	var te *yaml.TypeError
	if !errors.As(err, &te) {
		return fmt.Errorf("config: %w", err)
	}
	lines := make([]string, 0, len(te.Errors)+1)
	unknown := false
	for _, e := range te.Errors {
		if m := unknownField.FindStringSubmatch(e); m != nil {
			lines = append(lines, fmt.Sprintf("unknown key %q (line %s)", m[2], m[1]))
			unknown = true
			continue
		}
		lines = append(lines, e)
	}
	if unknown {
		lines = append(lines, "All keys: "+ReferenceURL)
	}
	return fmt.Errorf("config: %s", strings.Join(lines, "\n"))
}
