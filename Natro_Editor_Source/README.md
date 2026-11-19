# Natro Editor — recoverable source package

**Based on:** `Natro_Editor_Desktop_Name_Fix.exe` (v95-based build, supplied in this conversation).

This ZIP contains the maximum **recoverable/editable code** from that build, plus a
reconstructed Windows launcher that builds a functional editor application.

## What is and is not source

- **Exact extracted editor frontend**: `editor/Natro_Editor.html` contains its HTML, CSS,
  and JavaScript, including the `Import .AHK` controls, the Imported Macros dropdown,
  the editor UI, and loop tools. It was extracted byte-for-byte from the supplied EXE.
- **Reconstructed Windows launcher source**: `launcher/main_windows.go` and
  `launcher/discover.go` are equivalent source implementations of the install,
  folder-detection, redirect, Desktop-shortcut, and logging behavior. They were
  **reconstructed**, not recovered as the original `.go` files from the EXE.
- **Compiled core dependency**: `launcher/Natro_Editor_core.exe` is a byte-for-byte
  extracted, bundled native Windows app. **Its original Go source is not available**
  in the submitted binary. A compiled executable cannot simply be restored to its
  original source code. This file is retained so the reconstructed launcher can
  produce a working Windows application.
- **Yarn-ball icon**: `resources/original_app_icon.rsrc` is the native Win32 icon
  resource extracted from the latest EXE; `resources/yarn_ball_original.png` is
  the user-supplied yarn-ball image.

**This is not a complete open-source rebuild of the native Go backend.** The compiled
core remains a binary dependency. The source rebuild may not be byte-identical to
`Natro_Editor_Desktop_Name_Fix.exe`.

## Build the Windows EXE

Requires **Go 1.23+** and **Python 3** (standard libraries only).

On Windows, double-click `BUILD_WINDOWS.cmd` or run it from Command Prompt.

On Linux/macOS with Go and Python installed, run `./BUILD_LINUX.sh` to cross-compile
for Windows x64.

The output will be `build/Natro_Editor.exe` (no browser-only workaround). The launcher
embeds the original compiled editor core and the original Windows yarn-ball icon,
creates/refreshes a `Natro Editor` Desktop shortcut, and uses NatroMacro
folder detection to redirect to an installed editor.

**Note:** Windows Smart App Control/SmartScreen may still warn. This build is unsigned,
and merely building from source does not create publisher trust.

## Changing the editor UI

Edit `editor/Natro_Editor.html`. To apply *small* edits to the existing Go backend:

```text
python tools/patch_editor.py --base-core launcher/Natro_Editor_core.exe --html editor/Natro_Editor.html --output-core launcher/Natro_Editor_core_patched.exe
```

The patcher maintains the exact original embedded HTML byte length. If the new HTML
is longer, it **refuses** to patch. To rebuild arbitrary-length HTML or modify the
native backend, the original (unavailable) Go backend source would be required.

If patching succeeds, back up `launcher/Natro_Editor_core.exe`, copy the patched
file over it, and run the build script again.

## Project structure

```
README.md
BUILD_WINDOWS.cmd              Windows build script
BUILD_LINUX.sh                Cross-platform Windows build script
launcher/main_windows.go      Reconstructed launcher / native shortcut handling
launcher/discover.go          NatroMacro installation discovery
launcher/discover_test.go     Basic discovery test
launcher/go.mod               Go module declaration
launcher/Natro_Editor_core.exe Compiled original core (NOT source)
editor/Natro_Editor.html      Exact extracted frontend (editable source)
resources/yarn_ball_original.png Original yarn ball image
resources/original_app_icon.rsrc  Exact Windows icon resource from latest EXE
tools/attach_icon.py          Attaches Windows icon resources after Go build
tools/patch_editor.py         Optional fixed-length embedded frontend patcher
```

This package is intended for review and modification of the supplied application;
it does not disable Windows security or change startup/persistence policies.
