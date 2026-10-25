# How Replace works

Replace is built on the one thing every editable field understands: Ctrl+V. That keeps it working in almost every app, and it also sets its limits.

## The steps

1. **Check.** The source window must still exist and its app must not be on the deny list.
2. **Adapt the text.** The answer is trimmed, then the leading and trailing whitespace of the original selection is put back. All line breaks get the style of the original: `\r\n` if it contains `\r\n`, otherwise `\n`, and the OS default if it has no line break. This step is OS-neutral and unit-tested.
3. **Back up the clipboard.** All formats that live in global memory are copied. Bitmaps and metafiles as GDI handles, and formats Windows derives itself (such as plain text from Unicode text), are skipped. Very large items are skipped and logged.
4. **Write the answer** to the clipboard as Unicode text. For `output: markdown` actions the answer is also converted to HTML (goldmark, GitHub-flavoured, raw HTML in the answer dropped) and added as `CF_HTML` ("HTML Format"). Apps that understand rich text, such as Outlook and Word, paste the formatting; Notepad and code editors take the plain text, which is the Markdown source. `plain` actions never get `CF_HTML`. It is marked to stay out of the Windows clipboard history (Win+V) and the cloud clipboard.
5. **Hide the result window** and wait until you have released Ctrl, Shift, Alt and Win. Otherwise a key still held from the Ctrl+Enter shortcut could turn the paste into Ctrl+Shift+V.
6. **Activate the source window** and wait until it is in front. If Windows refuses, Spellcast borrows the input queue of the current foreground window for a moment. If the window does not come forward within half a second, nothing is pasted and you get a message.
7. **Send Ctrl+V.**
8. **Restore the clipboard** half a second later. Some apps read the clipboard lazily, so restoring right away could paste the old content. If you copied something else in the meantime, the clipboard is left alone.

If a step before the paste fails, the old clipboard is restored at once.

## What Spellcast cannot know

After Ctrl+V, Spellcast has no way to see whether the app accepted the text. A read-only view ignores the paste silently. That is why the window hides before the paste, and why **Show last answer** exists: the answer stays in memory until the next trigger, so a paste that went nowhere costs one click, not a new request.

Windows allows input only into windows of the same or a lower privilege level (UIPI). Replacing in an app that runs as administrator needs Spellcast to run as administrator too.

## Staying on top of the source window only

While the result window is open, a foreground watcher (`SetWinEventHook`) follows which window is in front. The result window is always-on-top while the source window or the result window itself is in front. When another app comes to the front, the window drops directly behind it, and it returns to the top when you go back to the source window. If the source window is closed, the result window becomes a normal window and Replace is disabled.

The window is not made a child of the source window: that would couple the input queues of both processes, and a hanging source app could then hang Spellcast.

Both z-order changes use `SWP_NOACTIVATE`. Without it, returning to the source window would give the focus to the result window, as Wails' own always-on-top call does.
