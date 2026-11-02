# Packaging and shell integration

## Portable Windows binary (C9)

No installer required for core use. Build from the repo root:

```powershell
cd <repo-root>
wails build
```

Copy `build\bin\airlink.exe` alone to any folder and run:

```powershell
.\build\bin\airlink.exe -- "C:\path\to\file.pdf"
```

The product binary is `airlink.exe` from `wails build`. Developers can still run `go run ./cmd/airlink`.

Needs: Windows 10/11, WebView2 Runtime (usually preinstalled), same Wi-Fi as the phone.

The Wails desktop exe self-registers HKCU classic verbs (files + folders) and a **Send to** shortcut on first run. Re-run after moving the `.exe` (Settings → Repair, or `--install-menu`).

```powershell
& ".\build\bin\airlink.exe" --install-menu
& ".\build\bin\airlink.exe" --uninstall-menu
```

## Windows 11 modern context menu (C1)

The short Windows 11 menu needs a **signed** sparse package (`docs/WIN11_MENU.md`). Until then, **Show more options** / Shift+right-click uses the classic HKCU verb. Do not set `ExplorerCommandHandler` on that verb — it hides the classic item when the package is unsigned.

## macOS Finder Quick Action (C2 stub)

See `scripts/macos/README.md`. Uses Automator “Quick Action” → Run Shell Script calling a built `airlink` binary.

## Linux Nautilus / Dolphin (C2 stub)

See `scripts/linux/`. Nautilus script and a KDE `.desktop` service menu template.
