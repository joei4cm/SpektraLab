# mygo_ui/ — UI development notes

Working log for whoever picks up the MyGo editor next. Newest first.
Read with `git log --oneline -20 -- mygo_ui`.

Merged to `Windows_Linux` as PR #4 (2026-11-12).

---

## Direction (2026-11-12)

**UI development for this fork’s Windows/Linux line goes through MyGo
native UI (`mygo_ui/`), not the Android Compose MVP and not new Tauri/Qt
screens unless asked.**

- **Stack:** [joei4cm/mygo](https://github.com/joei4cm/mygo) (module path
  `github.com/egoist/mygo`). Go 1.27.3+. View = function of state; GPU-drawn;
  **no cgo**, **no webview** for the editor chrome.
- **Engine boundary:** `spektralab-host` over
  [`desktop/HOST-PROTOCOL.md`](../desktop/HOST-PROTOCOL.md) (stdio frames).
  Do not link the C++ engine into the UI process.
- **Layout parity:** three regions like
  `desktop/src/windows/EditorWindow.tsx` — left Film/Print · top bar +
  canvas · (right Parameters still TODO).
- **Name:** unofficial **FilmPrint Lab** in the window title; keep SpektraLab
  trademark/icon out of this surface (same rule as the Android APK).
- **Android (`android/`):** installable debug APK + release CI stay as-is;
  do not grow that Compose throwaway as the product UI. New UI work lands
  here first.
- **Tauri (`desktop/`) / Qt (`windows_UI/`):** leave them; they remain the
  existing host clients. MyGo is an additional native line on the same wire.

### How to run while iterating

```bash
cd mygo_ui
SPEKTRALAB_MOCK_HOST=1 go run .          # mock host, no Vulkan
# or real host after engine/build-linux.sh — see README.md
```

### Next UI chunks (suggested order)

1. Right rail: exposure / print exposure / grain toggles via `set_params`
2. Live vs preview tier; cancel in-flight render
3. Filmstrip + multi-file open (`open` per path, session map)
4. Export dialog parity (`export_image` formats, long edge)
5. Geometry / crop when the host view path is wired the same as Tauri

Measure against the mock host first (`SPEKTRALAB_MOCK_HOST=1`); a broken
wire shows up as a picture that does not change when a control moves
(same honesty rule as `desktop/mock-host`).
