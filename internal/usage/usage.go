// Package usage turns the token counts of a response into an estimated cost
// and keeps running totals per day. Every cost here is an estimate: the API
// reports tokens, never money, and the Claude Console stays the authoritative
// source.
package usage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"spellcast/internal/claude"
	"spellcast/internal/config"
)

// Cost returns the estimated cost in USD of u at price p (USD per million tokens).
func Cost(p config.Price, u claude.Usage) float64 {
	const million = 1e6
	return (float64(u.InputTokens)*p.Input +
		float64(u.OutputTokens)*p.Output +
		float64(u.CacheReadTokens)*p.CacheRead +
		float64(u.CacheWriteTokens)*p.CacheWrite) / million
}

// Line is the short text under an answer, e.g. "312 in · 187 out · ≈ $0.0001".
// "in" includes cached input tokens. Without a price only the tokens are shown.
func Line(u claude.Usage, cost float64, priced bool) string {
	s := fmt.Sprintf("%d in · %d out", u.InputTokens+u.CacheReadTokens+u.CacheWriteTokens, u.OutputTokens)
	if priced {
		s += " · ≈ " + Money(cost)
	}
	return s
}

// Money formats a USD amount: four decimals below one cent, else two.
func Money(usd float64) string {
	if usd < 0.01 {
		return fmt.Sprintf("$%.4f", usd)
	}
	return fmt.Sprintf("$%.2f", usd)
}

// Day is the total of one local calendar day.
type Day struct {
	Requests   int     `json:"requests"`
	Unpriced   int     `json:"unpriced,omitempty"` // requests of models without a price
	Input      int64   `json:"input"`
	Output     int64   `json:"output"`
	CacheRead  int64   `json:"cache_read,omitempty"`
	CacheWrite int64   `json:"cache_write,omitempty"`
	Cost       float64 `json:"cost"` // estimated USD, priced requests only
}

func (d *Day) add(o Day) {
	d.Requests += o.Requests
	d.Unpriced += o.Unpriced
	d.Input += o.Input
	d.Output += o.Output
	d.CacheRead += o.CacheRead
	d.CacheWrite += o.CacheWrite
	d.Cost += o.Cost
}

// Summary is what the tray shows.
type Summary struct {
	Today, Month Day
}

// Texts returns the two tray lines, e.g. "Today: 42 requests · ≈ $0.12".
func (s Summary) Texts() (today, month string) {
	return "Today: " + dayText(s.Today), "This month: " + dayText(s.Month)
}

func dayText(d Day) string {
	noun := "requests"
	if d.Requests == 1 {
		noun = "request"
	}
	t := fmt.Sprintf("%d %s · ≈ %s", d.Requests, noun, Money(d.Cost))
	if d.Unpriced > 0 {
		t += " + unpriced"
	}
	return t
}

type file struct {
	Days map[string]Day `json:"days"`
}

// Tracker records finished requests and keeps the daily totals in a file.
type Tracker struct {
	path  string
	price func(model string) (config.Price, bool)
	now   func() time.Time

	mu       sync.Mutex
	days     map[string]Day
	onChange func(Summary)
}

// NewTracker loads the totals from path (a missing file is fine). price looks
// up the current prices on every request, so a config reload applies at once.
func NewTracker(path string, price func(model string) (config.Price, bool)) *Tracker {
	t := &Tracker{path: path, price: price, now: time.Now, days: map[string]Day{}}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		var f file
		if err := json.Unmarshal(data, &f); err != nil {
			// Keep the broken file for inspection, start empty.
			slog.Warn("usage file is not valid, starting empty", "path", path, "err", err)
			_ = os.Rename(path, path+".bad")
		} else if f.Days != nil {
			t.days = f.Days
		}
	case !os.IsNotExist(err):
		slog.Warn("cannot read the usage file", "path", path, "err", err)
	}
	return t
}

// OnChange sets a function that is called after every change with the new
// summary. It is called without any lock held.
func (t *Tracker) OnChange(f func(Summary)) {
	t.mu.Lock()
	t.onChange = f
	t.mu.Unlock()
}

// Summary returns the totals for today and the current month (local time).
func (t *Tracker) Summary() Summary {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.summaryLocked()
}

func (t *Tracker) summaryLocked() Summary {
	now := t.now()
	today := now.Format("2006-01-02")
	month := now.Format("2006-01")
	var s Summary
	for key, d := range t.days {
		if key == today {
			s.Today = d
		}
		if len(key) >= 7 && key[:7] == month {
			s.Month.add(d)
		}
	}
	return s
}

// Record counts one finished request and returns the line for the window.
// It logs one line (model, action, tokens and cost; never the text or the key).
func (t *Tracker) Record(model, action string, u claude.Usage) string {
	p, priced := t.price(model)
	cost := 0.0
	if priced {
		cost = Cost(p, u)
	}
	slog.Info("request done", "model", model, "action", action,
		"input_tokens", u.InputTokens, "output_tokens", u.OutputTokens,
		"cache_read_tokens", u.CacheReadTokens, "cache_write_tokens", u.CacheWriteTokens,
		"cost_usd", cost, "priced", priced)

	entry := Day{Requests: 1, Input: u.InputTokens, Output: u.OutputTokens,
		CacheRead: u.CacheReadTokens, CacheWrite: u.CacheWriteTokens, Cost: cost}
	if !priced {
		entry.Unpriced = 1
	}

	t.mu.Lock()
	key := t.now().Format("2006-01-02")
	d := t.days[key]
	d.add(entry)
	t.days[key] = d
	err := t.saveLocked()
	sum := t.summaryLocked()
	onChange := t.onChange
	t.mu.Unlock()

	if err != nil {
		slog.Warn("cannot save the usage file", "path", t.path, "err", err)
	}
	if onChange != nil {
		onChange(sum)
	}
	return Line(u, cost, priced)
}

// saveLocked writes the file through a temporary file, so a crash cannot
// leave half a file.
func (t *Tracker) saveLocked() error {
	data, err := json.MarshalIndent(file{Days: t.days}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, t.path)
}
