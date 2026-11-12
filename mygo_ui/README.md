# FilmPrint Lab — MyGo native UI

A **MyGo** rebuild of the SpektraLab Windows/Linux editor surface.

The earlier Android Compose MVP (`android/`) was a throwaway screen: one
column of Material defaults with no editor structure. This package starts
over using [MyGo](https://github.com/joei4cm/mygo) (fork of egoist/mygo):

- **Native UI in Go** — `view = f(state)`, GPU-drawn, no webview / no React
- **Desktop editor layout** — top bar · left film/paper rail · canvas
  (same three-region idea as `desktop/src/windows/EditorWindow.tsx`)
- **Real host wire** — talks to `spektralab-host` over
  [`desktop/HOST-PROTOCOL.md`](../desktop/HOST-PROTOCOL.md) (stdio frames),
  so the C++/Vulkan engine stays out-of-process and **no cgo** is needed
  (MyGo’s hard rule)

Unofficial working name **FilmPrint Lab** (not SpektraLab branding).

## Layout

```
┌─────────────────────────────────────────────────────────┐
│ FilmPrint Lab · MyGo          [Open…] [Render] [Export] │
├──────────────┬──────────────────────────────────────────┤
│ Film         │                                          │
│  Portra 400  │              canvas / preview            │
│  …           │                                          │
│ Paper        │                                          │
│  Endura      │                                          │
├──────────────┴──────────────────────────────────────────┤
│ status                                       host info  │
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

- **Go 1.27.3+** (`go version`)
- Either:
  - a built `spektralab-host` + engine resources (Vulkan), or
  - **Node ≥ 22** for the TypeScript mock host (`desktop/mock-host`)

`go.mod` pulls MyGo from `github.com/joei4cm/mygo` via a `replace` (module
path remains `github.com/egoist/mygo`).

## Run (mock host — no GPU)

From the SpektraLab repo root:

```bash
cd mygo_ui
SPEKTRALAB_MOCK_HOST=1 go run .
```

Opens a native window; Open any path (the mock invents a picture per path),
pick film/paper, Render, Export PNG.

## Run (real Vulkan host)

```bash
# once: build the host (see engine/build-linux.sh)
engine/build-linux.sh

cd mygo_ui
SPEKTRALAB_HOST=../build/host-linux-x64/spektralab-host \
SPEKTRALAB_ENGINE_RESOURCES=../build/host-linux-x64/engine \
  go run .
```

Or put `spektralab-host` on `PATH` and point
`SPEKTRALAB_ENGINE_RESOURCES` at a staged `engine/` tree (baked resources +
`vulkan/*.spv`).

## Build

```bash
cd mygo_ui
go test ./...
go build -o filmprint-lab .
```

## What’s in / out of this MVP

| In | Out (follow-ups) |
|---|---|
| Open JPEG/PNG/RAW via host | Full right-rail adjustments / curves |
| Film + paper picker (curated subset) | Filmstrip, batch export, pair frames |
| Preview render (`render_view` → `render`) | Geometry / crop / compare |
| Export PNG | Sidecars, menus, i18n parity with Tauri |

The Tauri (`desktop/`) and Qt (`windows_UI/`) frontends stay; this is the
MyGo-native line for the same host protocol.
