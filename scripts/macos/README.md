# macOS Finder Quick Action (stub)

1. Build: `GOOS=darwin GOARCH=arm64 go build -o airlink ./cmd/sendtophone`
2. Open **Automator** → Quick Action.
3. Workflow receives: **files or folders** in **Finder**.
4. Add **Run Shell Script**, Pass input: **as arguments**:

```bash
/usr/local/bin/airlink -- "$@"
```

5. Save as **Air-Link**.
6. Finder → right-click → Quick Actions → Air-Link.

Desktop UI is Windows/WebView2-only today; on macOS use `-no-window` or extend `internal/ui`.
