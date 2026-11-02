# Tray icon + screenshot hotkey (M4.9)

**Dependency:** `github.com/getlantern/systray` (Windows tray only; screenshot capture stays in `internal/capture`).

```powershell
.\build\bin\airlink.exe -tray
.\build\bin\airlink.exe -tray -hotkey
```

`-hotkey` implies `-tray`. Default combo is **Ctrl+Shift+P** (`-hotkey-combo`, off unless `-hotkey`). **Start with Windows** is a tray checkbox (`-install-startup` / `-uninstall-startup`). Receive from the tray uses the stable port **17612** and does not idle out.

## Menu

- **Receive from phone** — starts `-receive` as a child process (QR window). Click again to stop it.
- **Screenshot** — starts `-screenshot` as a child (avoids nesting WebView2 in the tray process).
- **Exit** — kills a running receive child and quits the tray.

## Hotkey conflict

If Ctrl+Shift+P is already registered, a message box explains that the hotkey is off; the tray **Screenshot** item still works.

## Fallback without tray

`airlink.exe -screenshot` and `airlink.exe -receive` still work from the CLI or the classic right-click menu.
