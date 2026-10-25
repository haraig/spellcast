# Your first 10 minutes with Spellcast

In this lesson you set up Spellcast, translate a sentence, put the result back into your text, and add an action of your own. You need Spellcast built or downloaded (see the [README](../../README.md#install)) and a Claude API key with some credit.

## 1. Start Spellcast and set the key

1. Start `spellcast.exe`. A notification says that Spellcast is running in the tray. Look for the wand icon in the taskbar's notification area (it may be under the small arrow).
2. Click the icon. The first entry is **Set API key…**. Choose it.
3. Paste your key and click **Save**. The window says "Saved. The key works." and closes.

## 2. Translate a sentence

1. Open Notepad and type: `Thank you for your quick reply, see you tomorrow.`
2. Select the sentence with the mouse or Shift+arrow keys.
3. Hold **CapsLock** and press **1**.

A small window opens next to your text cursor. The translation streams in: action 1 translates into English, and English text into German. Below it you may see notes, for example other ways to say it. The last line shows the tokens and the cost, a fraction of a cent.

## 3. Put the answer back

1. Click **Replace** (or press Ctrl+Enter).
2. The window closes, and Notepad now has the translation where your sentence was. Your clipboard is unchanged.

Try **Show changes** (Ctrl+D) on a rewrite later: it marks what the answer changed.

## 4. Add your own action

1. Click the tray icon and choose **Settings** > **Open config**. The config opens in your editor.
2. Add this below the other actions (keep the indentation):

   ```yaml
     "5":
       name: Formal
       prompt: >-
         Rewrite the text in a polite, formal tone. Keep the language and the meaning.
         Output only the rewritten text.
   ```

3. Save the file. Click the tray icon and choose **Settings** > **Reload config**.
4. Click the tray icon again: the list now shows **CapsLock+5   Formal**.
5. Select a casual sentence and press CapsLock + 5.

You now have a working Spellcast with an action of your own. Next steps:

- [Write good actions](../how-to/write-good-actions.md)
- [Keyboard shortcuts](../reference/keyboard-shortcuts.md)
- [Start Spellcast with Windows](../how-to/start-spellcast-with-windows.md)
