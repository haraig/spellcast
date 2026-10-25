# Configuration reference

File: `os.UserConfigDir()/spellcast/config.yaml` (Windows: `%AppData%\spellcast\config.yaml`). It is created with defaults on first start. Use the tray entry **Settings** > **Open config** to edit it and **Settings** > **Reload config** to apply changes.

Window size file: `%AppData%\spellcast\window.json`, written by Spellcast (see [placement](../explanation/result-window-placement.md)). Delete it to get the default size.

The API key is not in the config: see [Set the API key](../how-to/set-the-api-key.md).

Log file: `%LocalAppData%\spellcast\spellcast.log` (tray: **Help** > **Open log**).

Usage file: `%AppData%\spellcast\usage.json`, written by Spellcast, see [below](#usage-file).

## Keys

| Key | Type | Default | Description |
|---|---|---|---|
| `model` | string | `claude-haiku-5-5` | Default model. Can be set per action. |
| `max_tokens` | int > 0 | `4096` | Default token limit. Thinking tokens count toward it. Can be set per action. |
| `effort` | `low`, `medium`, `high`, `xhigh`, `max` | empty | Default effort level: how much the model thinks before it answers. Lower is faster and cheaper. Empty sends none, so the model uses its own default (`medium` for Haiku 5.5, `high` for Sonnet 5.5). Can be set per action. |
| `instructions` | text | empty | General rules added to the system prompt of every action (not of actions with `use_instructions: false`). Trimmed; at most 8000 characters. **Reload config** applies it at once. They steer the model but do not guarantee the result. |
| `prices` | map by model id | see below | Prices for the cost estimate, USD per million tokens. Entries replace the built-in default of that model; other defaults stay. Keys: `input`, `output`, `cache_read`, `cache_write` (the cache prices are optional; Spellcast does not use prompt caching). A model without a price shows tokens only. |
| `actions` | map | none | Actions by digit. See below. |
| `replace.deny.windows` | list of exe names | terminals | Apps where Replace is disabled (case-insensitive). Read on every use, so **Reload config** applies at once. The hotkeys also use it to decide where Ctrl+C is never sent. |
| `replace.deny.darwin` | list of bundle IDs | Terminal, iTerm2 | Same on macOS. |

## Actions

The key is one digit, `"1"` to `"9"` (quote it in YAML).

| Key | Required | Default | Description |
|---|---|---|---|
| `name` | yes | | Shown in the tray menu and the result window. |
| `prompt` | yes | | System prompt for Claude. The app adds the format instruction for `output` itself ([details](../explanation/how-a-request-works.md)). |
| `model` | no | top-level `model` | Model for this action. |
| `max_tokens` | no | top-level `max_tokens` | Token limit for this action. |
| `effort` | no | top-level `effort` | Effort level for this action. |
| `output` | no | `plain` | `plain` or `markdown`. |
| `notes` | no | `false` | Two-part answers: the text, a line with only `---`, then notes. The window shows the notes below the text; Replace and Copy use only the text. See [Get alternatives and notes](../how-to/get-alternatives-and-notes.md). |
| `use_instructions` | no | `true` | `false` leaves out the top-level `instructions` for this action. |

## Default config

New installations get the file below (the `instructions` example is commented out). Existing config files are never changed by an update. Use `|` for multi-line `instructions` (one rule per line) and `>-` for multi-line prompts (lines are joined into one paragraph).

```yaml
# Spellcast configuration. Use "Reload config" in the tray menu to apply changes.
#
# Multi-line text: use "|" for instructions (every line break is kept, one rule
# per line) and ">-" for prompts (lines are joined into one paragraph).

model: claude-haiku-5-5   # default model; can be set per action
max_tokens: 4096          # default; can be set per action
# effort: low             # low, medium, high, xhigh or max; can be set per action.
                          # Without it the model uses its own default.

# General rules for every action (optional). They steer the model but do not
# guarantee the result. An action can opt out with "use_instructions: false".
# instructions: |
#   Use a friendly, direct voice in your own text; keep the voice of the original when you translate or rewrite.
#   Prefer natural, everyday English over literal or stiff wording.
#   Keep explanations short.
#   Use American English spelling (color, behavior), also when you rewrite British English.
#   Never use em-dashes; use a comma, a colon or " - " instead.
#   Use only straight ASCII quotes and apostrophes (' and "), also in words like don't. No curly quotes, German quotes or guillemets.

# Prices for the cost estimate, USD per million tokens (optional; the defaults
# cover the default models, as of 2026-10-06). Check the Claude Console.
# prices:
#   claude-haiku-5-5: {input: 0.10, output: 0.50}

# The key is the digit pressed after CapsLock (1-9).
# output: plain (default) or markdown.
# With notes: true the model may add notes after a line with only ---.
# Replace and Copy use only the text before it. See docs/how-to/get-alternatives-and-notes.md.
actions:
  "1":
    name: Translate
    model: claude-sonnet-5-5
    effort: low
    notes: true
    prompt: >-
      Translate English text into German, and German text (or any other language) into
      English. The language of the whole text counts: German with some English words
      (Meeting, Deadline, code) is still German. Never answer in the language of the
      text and never return it unchanged. Keep line breaks, formatting, names, numbers,
      URLs and code. Translate questions and instructions; do not answer them. Start
      directly with the translation: no preamble, no reasoning, no explanation outside
      the notes. Notes for a single word or short phrase: up to three English
      alternatives (for an English word: English synonyms), one per line in the form "-
      word (German translation): when to use it", with no heading or lead-in. For a
      longer text into English: at most two notes on idioms with no word-for-word
      equivalent, usually none. Write notes in English.
  "2":
    name: Rewrite
    model: claude-sonnet-5-5
    effort: medium
    notes: true
    prompt: >-
      Fix spelling and grammar and improve style and clarity. Keep the meaning,
      language, tone, line breaks and formatting; add no information. Output only the
      revised text. Notes, for English text only: up to three real mistakes or
      unnatural phrasings, most useful first, one per line in the form "- original ->
      better: short reason" (for example "- make a photo -> take a photo: take is the
      usual verb with photo"), with no heading or lead-in. Skip small fixes; often no
      notes are needed. Write notes in English.
  "3":
    name: Explain
    effort: low
    output: markdown
    prompt: >-
      Explain the meaning of the text concisely; for code, explain what it does. For English
      text, also briefly explain idioms, phrasal verbs and rare words. Answer in the language
      of the text, or in English if that is unclear. Use short paragraphs or bullet points
      where helpful.
  "4":
    name: Look up
    effort: low
    output: markdown
    prompt: >-
      Write a short dictionary entry for the word or phrase. English: the headword in bold,
      part of speech, American IPA, main meanings, two short examples, common collocations,
      register where it matters, then a line "Translation:" with the German. German: the
      headword in bold, then up to four English equivalents, each with when to use it and a
      short English example; write this entry in English. If the word exists in both
      languages with different meanings, describe the English word and add a line
      "In German:" with its German meaning. If the text is more than a short phrase, say so
      in one sentence and explain it briefly.

# Apps where "Replace" is disabled (windows: exe names, darwin: bundle IDs).
replace:
  deny:
    windows: [WindowsTerminal.exe, conhost.exe, wezterm-gui.exe, mintty.exe]
    darwin: [com.apple.Terminal, com.googlecode.iterm2]
```

Actions: 1 Translate (Sonnet, notes with alternatives for single words), 2 Rewrite (Sonnet, notes on mistakes in English text), 3 Explain (Markdown), 4 Look up (Markdown).

## Prices

Built-in defaults (USD per million tokens, as of 2026-10-06): `claude-haiku-5-5` input 0.10, output 0.50 (for prompts up to 100K tokens; more above); `claude-sonnet-5-5` input 2, output 10, cache read 0.20; `claude-opus-5-5` input 4, output 20, cache read 0.20. Prices change, so check the Claude Console, and override them in the config:

```yaml
prices:
  claude-haiku-5-5: {input: 0.10, output: 0.50}
```

Negative prices and unknown keys are config errors. All costs shown by Spellcast are estimates, see [Check what Spellcast costs](../how-to/check-your-spending.md).

## Usage file

`usage.json` has one entry per local calendar day under `days`, for example:

```json
{ "days": { "2026-10-18": { "requests": 3, "input": 936, "output": 561, "cost": 0.00037 } } }
```

| Key | Meaning |
|---|---|
| `requests` | Finished requests that day. Cancelled and failed requests are not counted. |
| `unpriced` | How many of them used a model without a price (left out of `cost`). |
| `input`, `output` | Tokens. Thinking tokens are in `output`. |
| `cache_read`, `cache_write` | Cache tokens, if any. |
| `cost` | Estimated USD of the priced requests. |

Old days are kept. Delete the file to reset the totals. If it cannot be read, Spellcast renames it to `usage.json.bad` and starts empty.

## Validation

- Unknown keys are an error (catches typos). The notification names the key.
- All problems are reported together.
- At startup, an invalid file makes spellcast use the built-in defaults and show a notification. The file is not changed.
- On "Reload config", an invalid file keeps the previous config and shows a notification.
