# Icon specification

Requirements for the spellcast app and tray icon, and a ready-to-use prompt for an image/icon generator. To change the icon, see [Regenerate the app icon](../how-to/regenerate-app-icon.md).

## Chosen design
Free-standing violet wand (`#6D28D9`) with a white band, and two gold (`#FBBF24`) four-pointed sparkles with a violet outline. Bold on purpose: thick wand (170 px), thick outline (56 px), and the motif fills about 90 % of the canvas (more than the 80 % below), because the tray shows it at only 16-32 px. The outline keeps the sparkles visible on light and dark taskbars. Source: [build/appicon.svg](../../build/appicon.svg). To change it, see [Regenerate the app icon](../how-to/regenerate-app-icon.md).

## Requirements
| Item | Requirement |
|---|---|
| Motif | "Spell casting" applied to text: a magic wand with a sparkle, or a wand touching a letter/text line. Simple, one clear idea |
| Style | Flat, bold, geometric. Thick strokes, no thin lines, no gradients that matter, no tiny details, no text/letters in the image (unreadable at 16 px) |
| Colors | Max. 2-3 colors. One strong accent (suggestion: violet/indigo) plus white or a light tone. Must be recognizable on both a light and a dark Windows taskbar |
| Master file | 1024×1024 px PNG, transparent background, motif fills about 80 % of the canvas, centered. SVG preferred if the tool can export it |
| Shape | Free-standing symbol or a rounded square (squircle) badge. No drop shadow, no 3D |
| Small sizes | Must stay legible at 16×16, 24×24, 32×32 px. Test by scaling the master down |
| Monochrome variant | Optional, for a future macOS menu bar: solid black on transparent, same silhouette |
