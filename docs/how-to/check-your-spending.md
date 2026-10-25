# Check what Spellcast costs

Goal: see what requests used and cost, and compare the estimate with what Anthropic bills.

## In Spellcast

- **Under each answer** a muted line shows the tokens of that request and its estimated cost, for example `312 in · 187 out · ≈ $0.0001`. "in" includes cached input tokens.
- **In the tray menu**, under **Usage**, two read-only entries show the totals for today and this month, for example `Today: 42 requests · ≈ $0.12`. Today starts at midnight local time.
- **In the log** (tray: **Help** > **Open log**) every finished request has one line `request done` with the model, the action, the token counts and the cost. It never contains your text or the key.
- **In `usage.json`** next to `config.yaml` there is one entry per day. See the [configuration reference](../reference/configuration.md#usage-file).

## What the numbers are, and are not

- They are **estimates**: tokens times the prices in the config (`prices`). The API reports tokens, never money.
- A request that was cancelled, or that failed before the end, has no final token count and is not counted. The totals can therefore be a little too low.
- Thinking tokens are part of the output count, so they are included.
- A model without a price shows tokens only. Its requests are counted, and the tray adds "+ unpriced" when some requests had no price.

## Check the real spending

The Claude Console is the authoritative source.

1. Open the Claude Console (the web page where you manage your API key and credit) and go to its usage and cost pages.
2. Compare the day's tokens and cost with `Today` in the tray menu. They should agree to within rounding if you only use Spellcast with that key. Other apps that use the same key add to the Console's numbers.
3. If the prices differ, update `prices` in the config and use **Settings** > **Reload config**. The defaults are the prices of 2026-10-06; Anthropic can change them.

To limit spending, set a monthly limit in the Console and keep the API credit small.
