# Changelog

## 1.0.1 — 2026-11-03

- Rename the Go module and CLI package from `sendtophone` to `airlink` (`cmd/airlink`). The Windows binary is still `airlink.exe`.
- Internal temp-file prefixes now use `airlink-`. Auto-register still rejects leftover `sendtophone.exe` paths.

## 1.0.0 — 2026-11-03

First public release of **Air-Link**.

- Right-click files or folders on Windows and choose **Send with Air-Link**, or drop files on the desktop window
- Scan a QR with the phone camera; the file opens in Safari or Chrome — no phone app, no account, no cloud
- Same QR session is two-way: the phone can send photos, files, and text back to the PC (approval on the PC unless you turn on auto-allow)
- Large files stream with HTTP range requests; folders zip on the fly with no temp zip on disk
- Optional PIN, pairing for later sends from the phone (`/home`, `/api/upload`), tray icon, clipboard, screenshot, and Wi-Fi QR from the command line
- Binds only your LAN IP; session URLs include a random token; logs live in `%LOCALAPPDATA%\Air-Link\logs` (tokens stripped)
- Received files land in `%USERPROFILE%\Downloads\Air-Link` and never overwrite an existing name

Windows SmartScreen will warn on the unsigned `.exe`. Use **More info** → **Run anyway**.
