package usage

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spellcast/internal/claude"
	"spellcast/internal/config"
)

func TestCost(t *testing.T) {
	p := config.Price{Input: 2, Output: 10, CacheRead: 0.2, CacheWrite: 2.5}
	u := claude.Usage{InputTokens: 1_000_000, OutputTokens: 500_000, CacheReadTokens: 2_000_000, CacheWriteTokens: 400_000}
	// 2 + 5 + 0.4 + 1 = 8.4
	if got := Cost(p, u); math.Abs(got-8.4) > 1e-9 {
		t.Errorf("cost = %v, want 8.4", got)
	}
	if got := Cost(config.Price{}, u); got != 0 {
		t.Errorf("zero price = %v", got)
	}
}

func TestLineAndMoney(t *testing.T) {
	u := claude.Usage{InputTokens: 312, OutputTokens: 187}
	if got := Line(u, 0.000125, true); got != "312 in · 187 out · ≈ $0.0001" {
		t.Errorf("line = %q", got)
	}
	if got := Line(u, 0, false); got != "312 in · 187 out" {
		t.Errorf("unpriced line = %q", got)
	}
	// "in" counts the cached input too.
	if got := Line(claude.Usage{InputTokens: 10, CacheReadTokens: 100, CacheWriteTokens: 5, OutputTokens: 1}, 0, false); got != "115 in · 1 out" {
		t.Errorf("line with cache = %q", got)
	}
	for in, want := range map[float64]string{0: "$0.0000", 0.0042: "$0.0042", 0.01: "$0.01", 0.1234: "$0.12", 12.5: "$12.50"} {
		if got := Money(in); got != want {
			t.Errorf("Money(%v) = %q, want %q", in, got, want)
		}
	}
}

func prices(model string) (config.Price, bool) {
	if model == "m" {
		return config.Price{Input: 1, Output: 2}, true
	}
	return config.Price{}, false
}

func newTracker(t *testing.T, now *time.Time) (*Tracker, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "usage.json")
	tr := NewTracker(path, prices)
	tr.now = func() time.Time { return *now }
	return tr, path
}

func TestRecordTotalsAndFileRoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 18, 23, 59, 0, 0, time.Local)
	tr, path := newTracker(t, &now)

	line := tr.Record("m", "Translate", claude.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000})
	if line != "1000000 in · 1000000 out · ≈ $3.00" {
		t.Errorf("line = %q", line)
	}
	if got := tr.Record("other", "Explain", claude.Usage{InputTokens: 10, OutputTokens: 5}); got != "10 in · 5 out" {
		t.Errorf("unpriced line = %q", got)
	}

	s := tr.Summary()
	if s.Today.Requests != 2 || s.Today.Unpriced != 1 || s.Today.Input != 1_000_010 || s.Today.Output != 1_000_005 || math.Abs(s.Today.Cost-3) > 1e-9 {
		t.Errorf("today = %+v", s.Today)
	}
	today, month := s.Texts()
	if today != "Today: 2 requests · ≈ $3.00 + unpriced" || month != "This month: 2 requests · ≈ $3.00 + unpriced" {
		t.Errorf("texts = %q, %q", today, month)
	}

	// The totals survive a restart.
	again := NewTracker(path, prices)
	again.now = tr.now
	if got := again.Summary(); got != s {
		t.Errorf("after reload: %+v, want %+v", got, s)
	}
}

func TestDayRolloverAndMonth(t *testing.T) {
	now := time.Date(2026, 10, 31, 23, 59, 0, 0, time.Local)
	tr, _ := newTracker(t, &now)
	tr.Record("m", "A", claude.Usage{InputTokens: 1_000_000}) // $1.00 on Oct 31

	now = time.Date(2026, 11, 1, 0, 1, 0, 0, time.Local) // after midnight
	if s := tr.Summary(); s.Today.Requests != 0 || s.Month.Requests != 0 {
		t.Errorf("a new day and month start empty: %+v", s)
	}
	tr.Record("m", "A", claude.Usage{InputTokens: 2_000_000}) // $2.00 on Nov 1
	now = time.Date(2026, 11, 2, 8, 0, 0, 0, time.Local)
	tr.Record("m", "A", claude.Usage{InputTokens: 3_000_000}) // $3.00 on Nov 2

	s := tr.Summary()
	if s.Today.Requests != 1 || math.Abs(s.Today.Cost-3) > 1e-9 {
		t.Errorf("today = %+v", s.Today)
	}
	if s.Month.Requests != 2 || math.Abs(s.Month.Cost-5) > 1e-9 {
		t.Errorf("month = %+v (October must not count)", s.Month)
	}
}

func TestOnChange(t *testing.T) {
	now := time.Now()
	tr, _ := newTracker(t, &now)
	var got []Summary
	tr.OnChange(func(s Summary) { got = append(got, s) })
	tr.Record("m", "A", claude.Usage{InputTokens: 1})
	tr.Record("m", "A", claude.Usage{InputTokens: 1})
	if len(got) != 2 || got[1].Today.Requests != 2 {
		t.Errorf("changes = %+v", got)
	}
}

func TestBrokenFileIsKeptAndIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := NewTracker(path, prices)
	if s := tr.Summary(); s.Month.Requests != 0 {
		t.Errorf("summary = %+v", s)
	}
	if _, err := os.Stat(path + ".bad"); err != nil {
		t.Errorf("the broken file must be kept as .bad: %v", err)
	}
	tr.Record("m", "A", claude.Usage{InputTokens: 1}) // and recording works
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"requests": 1`) {
		t.Errorf("file = %s", data)
	}
}

func TestMissingFile(t *testing.T) {
	tr := NewTracker(filepath.Join(t.TempDir(), "sub", "usage.json"), prices)
	if s := tr.Summary(); s.Month.Requests != 0 {
		t.Errorf("summary = %+v", s)
	}
	tr.Record("m", "A", claude.Usage{}) // creates the directory
}
