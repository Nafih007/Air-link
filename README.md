# Air-Link

<p align="center">
  <img src="assets/brand/logomark-transparent.png" alt="Air-Link" width="96">
</p>

Right-click a file on your PC, scan a QR code, download on your phone. No phone app, no account, no cloud.

<!-- demo GIF here -->

## Features

- Right-click files or folders → **Send with Air-Link** (classic Explorer menu / Send to)
- Phone camera scans a QR; Safari or Chrome downloads in the browser
- Same session is two-way: phone can upload photos and files back to the PC
- Text, clipboard, screenshot, and Wi-Fi QR from the CLI
- HTTP range requests so large videos can stream and resume
- Folders zip on the fly (no temp zip on disk)
- LAN-only bind, random session token in the URL, no telemetry

## Tech stack

| Layer | Stack |
| --- | --- |
| Language | Go 1.22+ (module `airlink`; product binary `airlink.exe`) |
| Desktop | [Wails v2](https://wails.io/) + WebView2 |
| Phone page | Plain HTML, CSS, vanilla JS in `web/` (`go:embed`) |
| PC window | Vanilla HTML/CSS/JS in `frontend/dist` |
| Build | `wails build` (portable `airlink.exe`) |

This is a **desktop app with a local HTTP server**, not a hosted website. GitHub Pages cannot run it. Releases are Windows binaries.

## Prerequisites

- Windows 10 or 11
- Phone on the **same Wi-Fi** as the PC (not a guest/isolated network)
- Microsoft Edge WebView2 Runtime (usually already installed)
- To build from source: [Go](https://go.dev/dl/) 1.22+ and [Wails CLI v2.17+](https://wails.io/docs/gettingstarted/installation)

## Installation

### From GitHub Releases

1. Open [Releases](https://github.com/Nafih007/Air-link/releases) and download `airlink.exe` (or the NSIS installer, if that asset is attached).
2. Put `airlink.exe` in any folder and double-click it. First launch registers **Send with Air-Link** on the right-click menu for your user account.

Windows SmartScreen will warn because this build is **not code-signed**. Choose **More info** → **Run anyway**. That is expected. There is no silent, warning-free install until a signing certificate is used.

Repair the Explorer menu later:

```powershell
& ".\airlink.exe" --install-menu
```

### From source

```powershell
git clone https://github.com/Nafih007/Air-link.git
cd Air-link
wails build
```

Output: `build\bin\airlink.exe`. Optional installer: `wails build -nsis` (needs [NSIS](https://nsis.sourceforge.io/) / `makensis` on `PATH`).

## Environment variables

Air-Link does **not** use API keys or a `.env` file at runtime. Session tokens are generated with `crypto/rand`.

Optional build-time variable is listed in [`.env.example`](.env.example). Copy it only if you need it:

```powershell
Copy-Item .env.example .env
```

## Run locally

Desktop (hot reload):

```powershell
wails dev
```

Production binary:

```powershell
wails build
& ".\build\bin\airlink.exe"
```

The same flags work on `airlink.exe` (the desktop build can attach to a parent console when you pass flags):

```powershell
.\build\bin\airlink.exe
.\build\bin\airlink.exe -- "C:\path\to\file.pdf"
.\build\bin\airlink.exe -pin -- "C:\path\to\file.pdf"
.\build\bin\airlink.exe -receive
.\build\bin\airlink.exe -tray
.\build\bin\airlink.exe -clipboard
.\build\bin\airlink.exe -version
```

Developers can also run the CLI package without Wails: `go run ./cmd/airlink -h`.

## Build

| Command | Result |
| --- | --- |
| `go test ./...` | Unit tests |
| `go vet ./...` | Static checks |
| `gofmt -l .` | Format check |
| `wails build` | `build\bin\airlink.exe` |
| `make build` | Same as `wails build` (if `make` is installed) |

## Deploy

Ship the Windows binary (and optionally the NSIS installer) as a **GitHub Release**.

1. Push to `main`. CI runs tests and a Wails build on every push/PR (`.github/workflows/ci.yml`).
2. Tag a version: `git tag v1.0.1 && git push origin v1.0.1`
3. `.github/workflows/release.yml` builds `airlink.exe` and attaches it to the release.

This app **cannot** be deployed to GitHub Pages, Vercel, or Netlify: it must run on the PC so it can bind a LAN port and read local files.

## Project structure

```
cmd/airlink/       CLI entry (product binary is airlink.exe)
internal/server/   HTTP server, tokens, range, zip, uploads
internal/netinfo/  LAN interface detection
internal/qr/       QR generation
internal/ui/       CLI session window (WebView2)
internal/shell/    Explorer context menu / SendTo
internal/tray/     System tray
web/               Phone page (embedded)
frontend/dist/     Desktop window UI
assets/brand/      Source logos
build/             Wails icons, NSIS, output bin/
docs/              PRD, tasks, testing notes, spike decisions
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Please follow the [Code of Conduct](CODE_OF_CONDUCT.md). Security reports: [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © 2026 Muhammad Sinan

## Author

**Muhammad Sinan**

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| Phone never loads the page | Same Wi-Fi, not guest isolation. PC network profile **Private**. Allow the firewall prompt. |
| QR works but IP is wrong | `-list-ifaces` then `-bind 192.168.x.x` (skip VPN / WARP). |
| iPhone Camera shows a blank QR | Paste the URL into Safari. |
| SmartScreen warning | Unsigned binary — **More info** → **Run anyway**. See `docs/SIGNING.md`. |
