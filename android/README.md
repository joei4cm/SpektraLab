# FilmPrint Lab — Android MVP (unofficial)

An **unofficial, community** Android build of the SpektraLab C++/Vulkan
film/print simulation engine. It is deliberately **not** branded as SpektraLab:
the app name is **FilmPrint Lab**, the application id is `dev.filmprint.lab`,
and the launcher icon is a plain generated film-frame mark — not the official
SpektraLab name or icon. See the licence/credits note at the bottom.

This module lives on the `Windows_Linux` branch line (Android is a
Windows/Linux/Android-style port and belongs off `main`, per the repo's
contribution rules).

## What it does

A minimal end-to-end MVP that runs the **real spectral engine**, not a LUT stub:

1. Pick a JPEG/PNG from the device.
2. Choose a **film** and a **paper** (a curated subset of the profiles already
   baked into `engine/resources` + `engine/resources_product`, colour and B&W).
3. Render through the engine's headless Vulkan compute path
   (`spk_open` → `spk_render`, RGBA16 read back on the CPU).
4. Preview the print and **Export** it as a PNG into `Pictures/FilmPrint Lab`.

The engine is reused verbatim: the same `engine/src/core`, `engine/src/pipeline`
and `engine/src/gpu/vulkan_gpu.cpp` the Linux/Windows builds compile, behind the
same `extern "C"` ABI (`engine/include/spektrafilm/spk_engine.h`). The Android
layer is only a JNI bridge plus a small Jetpack Compose UI.

## Architecture

```
Kotlin UI (Compose)                 app/src/main/java/dev/filmprint/lab/*.kt
  → FilmEngine (JNI)                 ──┐
  ↕ libfilmprint_engine.so             │  app/src/main/cpp/jni_bridge.cpp
      spk_open / spk_render            │  app/src/main/cpp/CMakeLists.txt
      SpektraLab engine (C++20)        │  → engine/src/{core,pipeline,gpu}
      Vulkan 1.0 compute (volk)        │  → engine/third_party/volk
  ← engine assets (filesDir)         ──┘  assets/engine/** (staged at build)
```

- **No display/WSI**: the engine is Vulkan *compute-only*, so it renders
  headless into a buffer the app reads back and wraps in a `Bitmap`.
- **Input**: the picked image is decoded to sRGB, downscaled so the long edge is
  ≤ 2048 px, and handed to the engine as packed RGB floats with
  `input_color_space: "sRGB"`. Output is requested in sRGB too, so the RGBA16
  result maps straight to `ARGB_8888`.
- **Assets**: `spektrafilm_constants.bin`, the profile JSONs, the print-LUT
  index and the compiled SPIR-V kernels are staged into the APK by the Gradle
  `stageEngineAssets` task and extracted to app storage on first launch.

## Prerequisites

- JDK 17+
- Android SDK: platform 34, build-tools 34, platform-tools
- Android NDK (r26; CMake `3.22.1`)
- **`glslangValidator`** on `PATH` (or `VULKAN_SDK`/`GLSLANG_VALIDATOR`): it
  compiles the engine's `.comp` kernels to SPIR-V on the host at build time.
  On Debian/Ubuntu: `sudo apt-get install glslang-tools`.

One-time SDK/NDK install with the command-line tools:

```bash
sdkmanager "platform-tools" "platforms;android-34" "build-tools;34.0.0" \
           "ndk;26.3.11579264" "cmake;3.22.1"
```

Point Gradle at the SDK (either export `ANDROID_HOME` or write
`android/local.properties`):

```properties
sdk.dir=/absolute/path/to/android-sdk
```

## Build the debug APK

```bash
cd android
./gradlew :app:assembleDebug
```

The APK is written to:

```
android/app/build/outputs/apk/debug/app-debug.apk
```

It contains native libraries for **arm64-v8a** (modern devices) and **x86_64**
(emulator), the staged engine assets, and the compiled SPIR-V kernels.

Install and run:

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

## Releases (GitHub Release with the APK attached)

A second workflow, [`.github/workflows/android-release.yml`](../.github/workflows/android-release.yml),
builds the debug APK and publishes it as a **GitHub Release asset** on this fork.

- **Tag a release:** push a tag matching `android-v*` or `v*` and the workflow
  builds and attaches `FilmPrintLab-<tag>-debug.apk`:

  ```bash
  git tag android-v0.1.0 && git push origin android-v0.1.0
  ```

- **Manual:** Actions → *Android Release* → *Run workflow*, give it a tag name
  (the tag is created at the chosen commit).

The release is marked a pre-release and the APK is debug-signed (not for
production). The same file is also kept as a workflow artifact. It needs
`contents: write`, which the workflow requests via `permissions`.

## Notes, limits and gaps

- **Vulkan required at run time.** The device/emulator needs a Vulkan 1.0
  driver (`libvulkan.so`). Modern arm64 phones have one; use an emulator image
  with GPU/host Vulkan. Without it the app installs but reports an engine
  error rather than crashing.
- **Input formats.** JPEG/PNG only — decoded with Android's own codecs. RAW is
  not wired (the host LibRaw stack is intentionally not compiled into the app).
- **Resolution.** Capped at 2048 px long edge for memory/latency on-device;
  the engine itself has no such cap.
- **Not bundled.** The Digital-Intermediate `.cube` tables (`engine/.../DI`)
  are omitted (~15 MB, unused by this MVP). Lens correction and the desktop
  memory-suspend/residency path (which needs a writable temp dir) are not used.
- **Engine validation.** The identical staged asset set was rendered headless
  on CPU Vulkan (Mesa lavapipe) through `engine/tests/render_fixture.cpp`:
  colour (Portra 400 + Endura) and B&W (Tri-X + Multigrade) both render, and
  with the stochastic stages off the two renders are bit-exact (trap 1).

## Licence and credits

The engine, its 28 measured colour profiles, the 6 product profiles and the
print LUTs come from SpektraLab / [spektrafilm](https://github.com/andreavolpato/spektrafilm)
(Andrea Volpato), with 4 B&W profiles by Hanze Qiu. The engine and app are
**GPL-3.0-or-later**; the film/paper profiles and print LUTs are **CC BY-SA 4.0**.
The attribution/licence texts (`Product-profiles-NOTICE.md`, the font licences,
`stb-LICENSE.txt`) are staged into the APK assets beside the data they cover.
FilmPrint Lab is an unofficial build and is not endorsed by the SpektraLab
authors.
