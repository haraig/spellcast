# Regenerate the app icon

Goal: change `build/appicon.svg` and get the tray icon, the `.exe` icon and the macOS icon updated. No external tools are needed, only Go.

1. Edit `build/appicon.svg`.
   - Use only `<path>`, `<rect>` and `<circle>` with plain coordinates. The renderer does not support `<line>` or `transform` on groups.
2. Render the 1024×1024 PNG (transparent background):
   ```bash
   cd tools/render-icon
   go run . ../../build/appicon.svg ../../build/appicon.png 1024
   ```
3. Open `build/appicon.png` and check it.
4. Build. `task build` derives `build/windows/icon.ico` and `build/darwin/icons.icns` from the PNG:
   ```bash
   GOOS=windows task build
   ```
5. Check the tray icon at 100 % and 150 % scaling, on a light and a dark taskbar.

Design and requirements: [Icon specification](../reference/icon-specs.md).
