# How to write good actions

An action is a name and a prompt on one digit (CapsLock + 1 to 9). These tips help the answers fit straight into your text.

## Say what to output, and only that

Say "Output only the …" (the translation, the rewritten text, the reply). Otherwise models like to add "Here is the translation:", and Replace would paste that too.

## Keep what must stay

Name what must not change: "Keep line breaks, formatting, names, numbers, URLs and code unchanged." "Keep the language and the tone."

## Do not repeat format rules

Spellcast adds the format instruction itself: plain text for `output: plain`, Markdown for `output: markdown`. Use `markdown` for explanations you read in the window, and `plain` for text you put back with Replace.

## Put extras into notes

With `notes: true`, the model may add notes after a line with only `---`: alternatives, a short reason, a warning. Replace and Copy use only the text before the line. Ask for them in the prompt, for example: "If the text is a single word, add up to three alternatives as notes." Details: [Get alternatives and notes](get-alternatives-and-notes.md).

## Shared rules go into instructions

Rules for every action (your spelling variant, "never use em dashes", a glossary of fixed terms) belong in the top-level `instructions`, not in each prompt. An action can opt out with `use_instructions: false`.

## Translate in two directions: name both

For a two-way action, state both directions as one rule: "Translate English text into German, and German text into English. The language of the whole text counts." A prompt that starts with "Translate the text into English" makes the model more likely to return English text unchanged. Add "never return it unchanged", and say that a few English words in a German text do not make it English.

Write rules, not steps. A step like "First decide the language" can make the model write its reasoning into the answer ("The text is English, so ..."), especially at low effort. If it still happens, add "Start directly with the answer: no reasoning."

## Give notes a line format

With `output: plain`, Spellcast tells the model not to use Markdown, so it may drop the dashes of a list. Give the exact form instead: one per line in the form "- word: when to use it", with no heading or lead-in.

## Pick the model and effort per action

The default model (Haiku) is fast and cheap. Where the wording matters, as in translations and rewrites, set a stronger model on that action only (`model: claude-sonnet-5-5`).

`effort` sets how much the model thinks before it answers. Short texts rarely need much: `low` keeps the answer fast, and `medium` suits rewrites where style matters. Without `effort` the model uses its own default, which is slower (`high` for Sonnet).

## Examples

```yaml
actions:
  "5":
    name: Formal
    prompt: >-
      Rewrite the text in a polite, formal tone. Keep the language and the meaning.
      Output only the rewritten text.
  "6":
    name: Reply
    notes: true
    prompt: >-
      Write a short, friendly reply to this message in its language.
      Output only the reply. Add one alternative reply as a note.
  "7":
    name: Summarize
    output: markdown
    prompt: >-
      Summarize the text in three bullet points in English.
```

All keys: [Configuration](../reference/configuration.md).
