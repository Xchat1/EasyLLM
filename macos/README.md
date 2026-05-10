# EasyLLM macOS App

This directory contains the native macOS wrapper for EasyLLM.

The app keeps the existing Go backend and Vue web project unchanged:

- `scripts/build-macos-app.sh` builds `web/dist`.
- The Go backend is compiled into `EasyLLM.app/Contents/Resources/easyllm`.
- `web/dist` is copied into `EasyLLM.app/Contents/Resources/web/dist`.
- `macos/MakeAppIcon.swift` generates `EasyLLM.icns` from `web/src/assets/brand/easyllm-app-icon.png` during packaging.
- The Swift launcher starts the bundled backend and opens it in a native `WKWebView`.

Build:

```bash
./scripts/build-macos-app.sh
```

Output:

```bash
build/macos/EasyLLM.app
```

Runtime data is stored outside the app bundle:

```text
~/Library/Application Support/EasyLLM/
```
