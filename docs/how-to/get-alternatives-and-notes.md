# Get alternatives and notes next to the answer

Goal: look up a word, or see alternatives to a translation, and still paste only the text you want.

Prerequisites: a working Spellcast with an [API key](set-the-api-key.md).

## Look up a single word

New config files contain the action **Look up** on digit 4. Select a word and press CapsLock + 4. For an English word you get a dictionary entry: part of speech, pronunciation, meanings, examples, common word combinations, register and the German translation. For a German word you get its English equivalents, each with when to use it and an example.

If your config file already exists, it is not changed. Add the action yourself (copy it from [Configuration](../reference/configuration.md#default-config)), then use **Settings** > **Reload config** in the tray menu.

## Show alternatives without pasting them

Turn on `notes: true` for an action and tell the prompt what goes into the notes. In new config files, Translate and Rewrite have it on: Translate lists English alternatives for a single word (English synonyms with their German translation if the word is English), Rewrite lists the main mistakes it fixed in an English text. A small example:

```yaml
"5":
  name: Synonyms
  notes: true
  prompt: >-
    Give the best synonym for the word in its language. Output only the synonym.
    Notes: up to three more synonyms, each with a few words on when to use it.
```

The window shows the answer, then the notes below it in a muted style. **Replace**, **Copy** and Ctrl+C (with nothing selected) use only the answer, never the notes.

## Rules

- Keep `notes` off for actions that do not need it. It is off by default.
- The prompt only says what the notes contain. Spellcast adds one sentence that tells the model to put them after a line with only `---`.
- If the model writes no such line, the whole answer is the text, as before.
- Only the first line that is exactly `---` counts. Longer lines such as `----` or `--- x` are normal text.
- With `output: markdown` the notes are rendered as Markdown too. In an action without `notes: true`, `---` stays a horizontal rule.

## Rules for every action

To set a tone or a glossary once, use the top-level `instructions` ([Configuration](../reference/configuration.md)). These rules steer the model but do not guarantee the result: a rule like "never use em-dashes" can still slip through now and then.
