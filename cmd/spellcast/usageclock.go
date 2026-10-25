package main

import (
	"context"
	"time"

	"spellcast/internal/usage"
)

// refreshUsageAtMidnight updates the tray totals when the local day (or month)
// changes without a request, so "Today" does not show yesterday's numbers.
func refreshUsageAtMidnight(ctx context.Context, tr *usage.Tracker, t *tray) {
	last, _ := tr.Summary().Texts()
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s := tr.Summary()
			if today, _ := s.Texts(); today != last && s.Month.Requests > 0 {
				last = today
				t.SetUsage(s)
			}
		}
	}
}
