# See what the answer changed

Goal: find out which words Claude changed in your text, for example after a proofreading or rewrite action.

Prerequisites: a [hotkey or trigger](use-the-built-in-hotkeys.md) that has produced a complete answer.

## Steps

1. Select some text and run an action (for example CapsLock + `1`).
2. When the answer is complete, click **Show changes** or press Ctrl+D.
3. The window shows your text and the answer in one view:
   - ~~struck-through red~~ words were in your text and are gone,
   - green words are new in the answer,
   - everything else is unchanged.
4. Click **Show changes** or press Ctrl+D again to return to the normal answer.

The setting is kept while the app runs: if you turn it on, the next answers show their changes as soon as they are complete.

## Good to know

- The comparison is local. It sends nothing to the API and costs nothing.
- It compares words and punctuation. Different spaces and line breaks are not changes. Upper and lower case are changes.
- For `output: markdown` actions the view shows the Markdown source instead of the formatted text, because the comparison is made on the raw answer.
- If the answer has the same words as your text, a note says "No changes".
- The button is disabled until the answer is complete, and for a cancelled answer.
- **Copy**, Ctrl+C without a selection and **Replace** always use the plain answer, never the marked-up view. The notes of a two-part answer are not part of the comparison.
- Very long texts (about 2,000 words or more on each side) are shown as one deleted block and one inserted block, because the exact comparison would be too slow.
