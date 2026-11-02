# Contributing to Air-Link

Thanks for helping. Please read the [Code of Conduct](CODE_OF_CONDUCT.md) first.

Security issues are **not** reported as public GitHub issues. See [SECURITY.md](SECURITY.md).

## Report a bug

Use the **Bug report** issue template. Include:

- OS (Windows 10/11) and Air-Link version (`airlink.exe -version`)
- What you did, what you expected, what happened
- Whether the PC and phone were on the same Wi-Fi
- Logs from `%LOCALAPPDATA%\Air-Link\logs\` with tokens/URLs redacted

Never paste a session URL, join code, or device token.

## Suggest a feature

Use the **Feature request** template. Requirements live in `docs/PRD.md` (IDs like C4, F5.8). Say which requirement you are extending, or that it is new.

## Development setup

Prerequisites:

- Go 1.22+ (this repo uses Go 1.25 in `go.mod`)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation) (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`)
- WebView2 Runtime (usually already installed on Windows 10/11)
- Windows for the desktop app, tray, and Explorer menu

```powershell
git clone https://github.com/Nafih007/Air-link.git
cd Air-link
go test ./...
wails build
```

There is no `.env` to fill in. Optional build version: see `.env.example`.

Useful commands (from the repo root):

| Task | Command |
| --- | --- |
| Format check | `gofmt -l .` (write: `gofmt -w .`) |
| Vet | `go vet ./...` |
| Tests | `go test ./...` |
| Desktop (dev) | `wails dev` or `make start` |
| Desktop (prod) | `wails build` or `make build` |
| CLI package | `go run ./cmd/airlink -h` (product binary is `airlink.exe`) |

The Go module path is `airlink`.

## Branch and commit conventions

- Branch from `main`: `fix/short-name` or `feat/short-name`
- One focused change per PR
- Commits: short imperative summary (e.g. `Fix range request for empty files`)
- Do not change app behavior unless the PR is explicitly a bug fix or a requested feature
- Do not log session tokens or full tokenised URLs

## Pull requests

1. Run `gofmt -w .`, `go vet ./...`, and `go test ./...`
2. Open a PR with the pull request template
3. Describe how you tested on a real phone if the change touches transfers, QR, or the phone page

Maintainers may ask you to keep files small and stick to the standard library unless a dependency is already approved.
