# Replace the selected text with the answer

Goal: put Claude's answer in place of the text you selected, without copying and pasting by hand.

Prerequisites: an [API key](set-the-api-key.md) and Spellcast running with the [built-in hotkeys](use-the-built-in-hotkeys.md).

## Steps

1. Select text in an editable field, for example in a mail you are writing, Notepad or a code editor.
2. Press CapsLock + a digit. The answer streams into the result window.
3. When the answer is complete, click **Replace** or press Ctrl+Enter.
4. The result window closes, the source window comes back to the front, and the answer takes the place of the selection.

Your clipboard is restored about half a second later.

## What Replace does and does not do

- It pastes into the **current** selection or cursor position of the source window. If you edited the text or moved the cursor in the meantime, the answer goes there.
- It is only enabled when the answer is complete.
- Spaces, tabs and line breaks at the start and end of your selection are kept, and the answer gets the line break style of the original (for example a trailing line break of a full-line selection in VS Code).
- For actions with `output: markdown`, apps that support rich text (Outlook, Word) get the formatted answer: headings, lists, bold. Other apps get the Markdown source as plain text.
- **Copy** and Ctrl+C always copy the raw answer. They are not affected by this.

## When Replace is disabled

Hover over the button to see the reason.

| Reason | What to do |
|---|---|
| Replace is disabled in `<app>` | The app is on the deny list, such as a terminal. Use **Copy**. See [Configuration](../reference/configuration.md). |
| The source window was closed | Use **Copy**. |
| The source window is unknown | Spellcast could not tell which window the text came from. Use **Copy**. |

## Read-only text

In a read-only view, such as the VS Code Markdown preview or a web page, Replace changes nothing and shows no error. Spellcast cannot tell whether a paste worked.

The answer is not lost: click the tray icon and choose **Show last answer**. The window reopens near the mouse pointer with the last answer. Use **Copy**, or put the cursor in an editable field first and then press **Replace**.

## If Replace fails

| Message | Meaning |
|---|---|
| Replace failed: could not switch back to the source window | Windows did not let Spellcast activate the window. Click the window yourself and try **Replace** again from **Show last answer**. |
| Replace failed: Windows blocked the paste | The source app runs as administrator and Spellcast does not. Run Spellcast as administrator too. |
| Replace failed: cannot write the clipboard | Another program holds the clipboard. Try again. |

How it works inside: [How Replace works](../explanation/how-replace-works.md).
