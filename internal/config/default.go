package config

// DefaultYAML is written on first start when config.yaml does not exist.
// Keep the actions in sync with docs/reference/configuration.md.
const DefaultYAML = `# Spellcast configuration. Use "Reload config" in the tray menu to apply changes.
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
`
