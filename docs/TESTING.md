# Manual testing checklist

Hand-test on a real PC + phone. Tick when verified. Criteria come from the PRD (Core MVP user flow, C1–C11, and non-functional security/performance notes relevant to M0).

## Checkpoint CP1 — QR + single-file download (M0.1–M0.4)

PC and phone on the same Wi-Fi. Allow inbound TCP if Windows Firewall prompts.

```powershell
cd <repo-root>
go run ./cmd/airlink -- "$env:WINDIR\win.ini"
```

Optional: `go run ./cmd/airlink -list-ifaces` then `go run ./cmd/airlink -bind <ip> -- "$env:WINDIR\win.ini"`

- [ ] PC prints `Interface: Wi-Fi` (not Cloudflare WARP), `Listening on 192.168.x.x:<port>`, opens a PNG QR window, and prints `Or open in Safari`
- [ ] Prefer scanning the **PNG window** (not a photo of the terminal ASCII QR)
- [ ] If Camera shows a **blank white box**, paste the `Or open` URL into Safari instead — iOS Camera previews often hang on LAN `http://` links
- [ ] Phone shows a landing page with **file name, size, and Download** (not an immediate raw download)
- [ ] Tap Download → file arrives
- [ ] Opening the URL with the token removed (e.g. `http://IP:PORT/`) returns **404**
- [ ] A second run uses a **different port** (and different token in the URL)

## Setup

- [ ] PC and phone on the **same Wi-Fi** (not guest/isolated unless testing diagnostics)
- [ ] Fresh portable binary (or `go run`) with no prior session running
- [ ] Test files ready: small text (~1 KB), medium image (~20 MB), large video (~200 MB+), a folder with several files

## Core user flow (M0)

- [ ] Right-click one file → **Send with Air-Link** (or CLI with that path) opens a small window with QR, file name, size and countdown
- [ ] Phone camera scans QR → browser opens the download page without installing an app
- [ ] PC window shows phone connected / progress while downloading
- [ ] After transfer, PC shows **Done** and the server shuts down (unless keep-alive)
- [ ] Time from right-click (or launch) to download start feels ≤ ~10 s on a quiet LAN

## Checkpoint CP5 — window progress + classic right-click (M0.11–M0.12)

**Window: connected / progress / Done**

```powershell
cd <repo-root>
go run ./cmd/airlink -- "$env:WINDIR\win.ini"
```

- [ ] Window shows QR, file name, size, countdown
- [ ] After phone opens the page: status **Phone connected**
- [ ] During download: progress line updates
- [ ] After download: **Done**, then window closes

**Keep-alive**

```powershell
go run ./cmd/airlink -keep-alive -- "$env:WINDIR\win.ini"
```

- [ ] First phone downloads; window stays open (`Sent ×1 — keep-alive…`)
- [ ] Second phone (or same phone again) can download without restarting
- [ ] Close the window to stop

**Classic right-click menu**

```powershell
cd <repo-root>
go build -o airlink.exe ./cmd/airlink
.\airlink.exe -install-shell
```

- [ ] Right-click a file → **Show more options** (Win11) / context menu (Win10) → **Send with Air-Link**
- [ ] Win11 compact menu: **Send with Air-Link** without Show more options (restart Explorer; Developer Mode if `-install-shell` warned)
- [ ] Session window + QR appear; phone can download
- [ ] Multi-select several files → one session (zip), not many windows

Uninstall later: `.\airlink.exe -uninstall-shell`

## Checkpoint CP6 — end of M0 (portable + security)

```powershell
cd <repo-root>
.\scripts\build-windows.ps1
Copy-Item .\airlink.exe "$env:TEMP\stp-portable\"
& "$env:TEMP\stp-portable\airlink.exe" -- "$env:WINDIR\win.ini"
```

- [ ] Portable `.exe` alone runs a transfer (WebView2 + QR)
- [ ] See `docs/WIN11_MENU.md`: compact menu after `-install-shell` (Developer Mode if needed); classic + Show more options works without it
- [ ] Security smoke: URL without token → 404; Listening IP is Wi-Fi not WARP; no token printed in normal window mode

## C1 / C2 — Shell integration

- [ ] **Windows 10:** context-menu “Send with Air-Link” appears and launches a session
- [ ] **Windows 11:** after `-install-shell`, **Send with Air-Link** in the compact menu (restart Explorer; Developer Mode if the package did not register)
- [ ] **Windows 11 classic path:** entry under “Show more options” works via registry verb
- [ ] **macOS / Linux (when implemented):** Finder Quick Action or Nautilus/Dolphin script starts a session

## C3 — LAN interface selection

- [ ] Default bind uses a real LAN IPv4 the phone can reach
- [ ] VPN / Docker / Hyper-V / WSL adapters are not chosen when a normal LAN iface exists
- [ ] Listing interfaces (debug/flag) matches what the user expects

## C4 — Port, token, binding

- [ ] Each session uses a **random port** and a **new URL token**
- [ ] Request **without** the token returns **404**
- [ ] Server binds to the **chosen LAN IP**, not `0.0.0.0` by default
- [ ] Logs never contain the token or full tokenised URLs

## Checkpoint CP3 — zip + resume (M0.7–M0.8)

```powershell
cd <repo-root>
# Zip: folder + file (create a temp folder with a couple of files first)
New-Item -ItemType Directory -Force -Path "$env:TEMP\stp-cp3" | Out-Null
Set-Content "$env:TEMP\stp-cp3\one.txt" "one"
Set-Content "$env:TEMP\stp-cp3\two.txt" "two"
go run ./cmd/airlink -- "$env:TEMP\stp-cp3" "$env:WINDIR\win.ini"
```

- [ ] Session window opens; phone page shows items / zip name
- [ ] Download is a `.zip` that extracts with expected files
- [ ] No huge temp `.zip` left under the project or `%TEMP%` from Air-Link

Range / resume (single large file, e.g. a video):

```powershell
go run ./cmd/airlink -- "C:\path\to\large-video.mp4"
```

- [ ] Start download, cancel mid-way, resume (or re-open) and it continues
- [ ] On phone, open/scrub the video in the browser player before the full file finishes

## C5 — Folders and multi-select

- [ ] Multiple files arrive as a **zip**
- [ ] A folder arrives as a **zip** with expected structure
- [ ] Zip is streamed (no large temp zip left on disk after the transfer)

## C6 — Range requests

- [ ] Interrupted download can **resume** (or re-fetch remaining bytes via Range)
- [ ] Video on the phone can **seek/scrub** before the full file finishes

## Checkpoint CP4 — idle shutdown + diagnostics (M0.9–M0.10)

**Done after transfer**

```powershell
cd <repo-root>
go run ./cmd/airlink -- "$env:WINDIR\win.ini"
```

- [ ] Phone downloads the file
- [ ] PC window shows **Done**, then closes (~2 s)
- [ ] Terminal prints `Done` then `Stopping.`

**Idle timeout (no phone)**

```powershell
go run ./cmd/airlink -idle 15s -- "$env:WINDIR\win.ini"
```

- [ ] Do not open the QR on the phone
- [ ] After ~15 s, session shuts down (`Idle timeout`)

**Diagnostics (no phone for 20s)**

```powershell
go run ./cmd/airlink -idle 2m -diagnose-after 20s -- "$env:WINDIR\win.ini"
```

- [ ] Leave phone on cellular / different Wi-Fi (or just don’t scan)
- [ ] Within ~20 s, PC window and terminal show hints: Firewall, same Wi-Fi, wrong adapter, **phone hotspot** fallback

## C7 — Auto-shutdown

- [ ] After a successful transfer (non–keep-alive), session ends within a short Done period
- [ ] With no phone activity, session ends after **idle timeout** (default 10 minutes; use a short flag for a quick check)
- [ ] Closing the PC window stops the server

## C8 — Keep-alive

- [ ] Keep-alive mode stays up after the first download
- [ ] A second phone can download from the same QR/session
- [ ] User can stop the session manually

## C9 — Packaging

- [ ] Single portable binary runs with no extra runtime installed
- [ ] Optional installer (when present) only needed to add menu entries
- [ ] Install/artifact size target ≤ 15 MB when measuring releases

## C10 / C11 — Diagnostics and hotspot

- [ ] If no phone connects within ~**20 s**, PC shows likely causes (firewall, guest network isolation, wrong Wi-Fi) and a fix
- [ ] Hotspot fallback hint appears: turn on the phone’s hotspot and connect the PC

## Security / privacy (M0-relevant)

- [ ] Only user-selected paths are served; `..` / absolute / symlink escape attempts fail safely
- [ ] No outbound internet calls during a normal transfer session

## Performance smoke (M0-relevant)

- [ ] Cold start to QR on screen feels ≤ ~1 s for a small file
- [ ] Large file transfer does not balloon PC memory (streamed; rough check via Task Manager on a multi-GB file)

## Cross-device matrix (smoke)

| Case | iPhone Safari | Android Chrome |
| --- | --- | --- |
| Small file download | [ ] | [ ] |
| Folder zip download | [ ] | [ ] |
| Large video + seek | [ ] | [ ] |
| QR scan → page loads | [ ] | [ ] |

## Checkpoint — end of M1 (iPhone-first page)

```powershell
cd <repo-root>
go run ./cmd/airlink -- "C:\path\to\photo.jpg"
go run ./cmd/airlink -- "C:\path\to\video.mp4"
go run ./cmd/airlink -- "C:\path\to\a.jpg" "C:\path\to\b.jpg" "C:\path\to\c.jpg" "C:\path\to\d.jpg"
```

**iPhone (HTTP — expected after spike)**

- [ ] Badge shows **iPhone**
- [ ] Red/amber **Save tip** appears immediately (no secure context / no Web Share)
- [ ] JPEG: preview image; **Download**; long-press → Save to Photos
- [ ] MP4: in-page player scrubs before full download; Download / long-press guidance
- [ ] No **Save to Photos** share-sheet button on plain HTTP (only when HTTPS later)
- [ ] Four JPEGs: thumbnail gallery with checkboxes; **Download selected (zip)** uses ticks; **Download all (zip)** gets everything

**Android**

- [ ] Badge shows **Android**; primary **Download**; optional Share only if secure context allows

**Formats**

- [ ] HEIC (if you have one): downloads with `.heic` name; bytes not re-encoded

**P2 skipped:** Add to Home Screen / PWA not implemented (F1.8).

### Feature 1 (acceptance tracking)

- [ ] HTTP fallback reaches Photos (long-press / Download + guide) — never blank page
- [ ] Share sheet ≤ 3 taps — **blocked until LAN HTTPS** (see PRD spike 2026-10-29)

### Feature 2 (beyond files)

- [ ] Copied short URL opens on the phone within ~5 s of `-clipboard`, with **no** network server when it fits in the QR
- [ ] Long text: page with Copy works; session follows idle rules
- [ ] Clipboard image: Save path like a normal PNG
- [ ] Screenshot: phone gets fullscreen PNG
- [ ] Wi-Fi: confirm before password read; WIFI: QR; password not stored on disk

## Checkpoint — end of M2 (beyond files)

```powershell
cd <repo-root>
# Short URL → direct QR (no server)
go run ./cmd/airlink -clipboard
# Long text → tokenised page + Copy
go run ./cmd/airlink -text "paste a long paragraph here..."
# Clipboard image (copy an image first)
go run ./cmd/airlink -clipboard-image
# Fullscreen screenshot
go run ./cmd/airlink -screenshot
# Wi-Fi QR (type y at the confirm prompt)
go run ./cmd/airlink -wifi
```

**Phone checks**

- [ ] Short URL: camera/scan opens the link; PC shows “direct QR — no server”
- [ ] Long text: page loads; Copy button puts text on phone clipboard
- [ ] Clipboard image / screenshot: Download / long-press → Photos like Feature 1
- [ ] Wi-Fi: after `y`, QR encodes WIFI:; phone offers to join (or shows network name); cancel with non-`y` leaves no password read

**Tray / hotkey (M4.9)**

```powershell
.\airlink.exe -tray
.\airlink.exe -tray -hotkey
```

- [ ] Tray icon: Receive from phone → QR upload session; click again to stop
- [ ] Tray Screenshot → QR of a PNG
- [ ] `-hotkey`: Ctrl+Shift+P captures; if the combo is taken, a message box appears and the tray still works
- [ ] M2.6 browser extension remains P2 / deferred

### Feature 3 (two-way)

- [ ] One scan: download a file **and** upload ~10 photos (≥ 50 MB total) back
- [ ] PC shows a single approval prompt for the upload batch (unless auto-accept)

## Checkpoint — M3.1 (Send to PC on session page)

```powershell
cd <repo-root>
go run ./cmd/airlink -auto-accept -- "C:\path\to\small.txt"
```

**Phone**

- [ ] Page shows **Send to PC** under the download actions
- [ ] Pick one photo → **Send to PC** → status **Saved on PC**
- [ ] File appears in `Downloads\Air-Link` (or `-upload-dir`)
- [ ] Without `-auto-accept`: status **Waiting for PC approval** (M3.3 will Accept/Decline)

## Checkpoint — M3.2 (Receive from phone)

```powershell
cd <repo-root>
go run ./cmd/airlink -receive -auto-accept
```

**Phone**

- [ ] Title **Receive from phone**; no file download actions
- [ ] **Send to PC** works (photo/file lands in `Downloads\Air-Link`)
- [ ] Session stays open for another upload until idle/close

## Checkpoint — M3.3 / M3.4 (approval + no overwrite)

```powershell
cd <repo-root>
# No -auto-accept — PC must Accept/Decline
go run ./cmd/airlink -receive
```

**Phone + PC**

- [ ] Phone picks **2 photos** → Send to PC → “Waiting for PC approval”
- [ ] PC window shows **Phone wants to send** with Accept / Decline
- [ ] **Decline** → phone shows Declined; nothing new in `Downloads\Air-Link`
- [ ] Send again → **Accept** → files saved; phone shows Saved
- [ ] Upload the same filename twice (Accept both) → second file is `name (1).ext`

## Checkpoint — M3.5 (toast + Open folder)

```powershell
cd <repo-root>
go run ./cmd/airlink -receive
```

- [ ] After **Accept**, PC window shows a green toast with **Open folder**
- [ ] Clicking **Open folder** opens `Downloads\Air-Link` (or `-upload-dir`) with the new files
- [ ] Terminal also prints `Folder: …`

## Checkpoint — M3.6 (chunked resume)

```powershell
cd <repo-root>
go run ./cmd/airlink -receive -auto-accept
```

- [ ] Phone uploads a large file (≥ 5 MB, ideally ≥ 50 MB); status shows percent / chunk progress
- [ ] PC window progress shows **Receiving x / y**
- [ ] Optional: toggle airplane mode mid-upload, turn Wi‑Fi back on, tap **Send to PC** again — resumes without starting at 0%
- [ ] File in `Downloads\Air-Link` matches size

## Checkpoint — M3.7 (text to clipboard)

```powershell
cd <repo-root>
go run ./cmd/airlink -receive -auto-accept
```

- [ ] Phone: paste a unique sentence in **Send to PC** (no files) → Send
- [ ] Phone status: **Copied to PC clipboard**
- [ ] PC window toast: **Copied text from phone to clipboard**
- [ ] Paste on the PC (Notepad) — text matches; no extra `from-phone.txt` in `Downloads\Air-Link`

## Checkpoint — M3.8 (HEIC keep / optional JPEG)

```powershell
cd <repo-root>
go run ./cmd/airlink -receive -auto-accept
# then, optional convert:
go run ./cmd/airlink -receive -auto-accept -convert-jpg
```

- [ ] Default: upload a HEIC (or PNG); disk file keeps original extension and bytes
- [ ] `-convert-jpg`: PNG becomes `.jpg`; HEIC becomes `.jpg` if Windows HEIF codec/WIA works, otherwise stays `.heic` (original kept)

## Checkpoint — end of M3 (two-way, one QR)

PRD Feature 3 acceptance: one scan, download a file **and** upload photos back.

```powershell
cd <repo-root>
go run ./cmd/airlink -keep-alive -- "C:\path\to\small.txt"
```

- [ ] Scan once; download the file from the phone
- [ ] Same page: **Send to PC** with ~10 photos (≥ 50 MB total if you have them)
- [ ] PC shows **one** Accept/Decline for the batch (not per file)
- [ ] Accept → files in `Downloads\Air-Link`, toast + **Open folder**
- [ ] Optional receive-only: `go run ./cmd/airlink -receive`

## Checkpoint — M4.1–M4.4 (PIN, Kick, rate-limit)

```powershell
cd <repo-root>
go run ./cmd/airlink -pin -- "$env:WINDIR\win.ini"
```

- [ ] PC window shows a **4-digit PIN**
- [ ] Phone: PIN prompt; wrong PIN rejected; correct PIN loads the page; Download works
- [ ] Five wrong PINs → session locked (new run for a new PIN)
- [ ] After a successful open, PC lists the phone (IP · Safari/Chrome) with **Kick**
- [ ] Kick → phone refresh/download fails; start a new session to send again
- [ ] Optional: hammer a wrong-token URL; after ~20 tries you get HTTP 429

## Checkpoint — M6.1 (empty start)

```powershell
cd <repo-root>
go run ./cmd/airlink
```

- [ ] PC window: QR + **Waiting for phone…** (no file names)
- [ ] Scan on iPhone → page loads (Connected / waiting); Send to PC still works
- [ ] Close the window → process exits
- [ ] `go run ./cmd/airlink -- "$env:USERPROFILE\Desktop\test-files\note.txt"` still does a one-shot send

## Checkpoint — M5 pairing + Shortcut API

```powershell
cd <repo-root>
go run ./cmd/airlink -receive
```

- [ ] Listens on port **17612**; idle does not expire until you close the window
- [ ] Phone: **Pair this phone** → page shows `POST` URL, Bearer token, and `/home`
- [ ] PC window **Paired phones**: Rename / Remove; Remove → that Bearer fails
- [ ] Follow `docs/IPHONE_SHORTCUT.md` (or curl below) without scanning again
- [ ] File lands in `Downloads\Air-Link`; PC toast + Open folder
- [ ] Open `http://PC_IP:17612/home` (or hostname.local); Add to Home Screen; send a file
- [ ] Request without Bearer / wrong token → 401
- [ ] Close the receive window → uploads fail (one-click off)

```powershell
curl.exe -H "Authorization: Bearer PASTE_TOKEN" -F "files=@C:\path\to\photo.jpg" http://PC_IP:17612/api/upload
```

## Checkpoint CP-A — Wails desktop window (M7.1–M7.2)

PC and iPhone on the same Wi-Fi. Build from the repo root (where `wails.json` lives):

```powershell
cd <repo-root>
wails build
```

- [ ] `build\bin\airlink.exe` exists
- [ ] Double-click the `.exe` → window titled **Air-Link** with a large QR, URL + Copy, status **Scan with your phone camera**
- [ ] If WebView2 is missing: a message with the Evergreen download link, not a blank crash
- [ ] iPhone Camera or Safari opens the room after scanning (or paste URL if Camera hangs on LAN `http://`)
- [ ] PC status becomes **Connected** (and the device list / Kick if a phone is listed)
- [ ] CLI still works: `go run ./cmd/airlink -- "$env:WINDIR\win.ini"`

## Checkpoint — single-use join QR (M7.9)

Same Wi-Fi. Two phones, or iPhone + a laptop browser. Auto-allow **off**.

```powershell
cd <repo-root>
wails build
& ".\build\bin\airlink.exe"
```

- [ ] QR has a countdown (~2 min); **Refresh QR** issues a new code without dropping anyone already allowed
- [ ] Phone A scans → PC **New device: Safari (or Chrome). Allow / Deny** → Allow → phone page works; reload stays connected
- [ ] Phone B (or laptop) scans the *old* QR → “This code was already used. Scan the new one on the PC”
- [ ] Phone B scans the *new* QR on the PC → Allow → both stay in the same room; Kick A does not drop B
- [ ] Deny a third scan → that browser cannot load files
- [ ] Unused QR left for 2 minutes refreshes by itself
- [ ] `%LOCALAPPDATA%\Air-Link\logs\` has no `/j/<hex>` or `/t/<hex>`

## Checkpoint CP-B — Send / receive from the window (M7.3)

Same Wi-Fi. One session, no extra QR.

```powershell
cd <repo-root>
wails build
& ".\build\bin\airlink.exe"
```

- [ ] Window: **Send files…**, drop a file, timeline shows the outbound item
- [ ] iPhone page (already open) lists the file after a couple of seconds — Save/Download works
- [ ] Phone **Send to PC** a photo → PC timeline **Accept** → file in `Downloads\Air-Link`
- [ ] **Open received folder** shows that file
- [ ] **End session** → phone refresh 404s; **New QR** starts a new room

## Checkpoint CP-C — Desktop app polish (M7.4–M7.8)

- [ ] Double-click `build\bin\airlink.exe`: no console window; taskbar icon is the Air-Link logo (not generic Go)
- [ ] After a session, `%LOCALAPPDATA%\Air-Link\logs\YYYY-MM-DD.log` exists and has no `/t/<32 hex>/`
- [ ] App running; launch the exe again → one window, focused
- [ ] Right-click a file → **Show more options** → **Send with Air-Link** (after first launch of the Wails exe) adds it to the open room
- [ ] Close → **Keep in tray** → tray icon; uploads still work; tray **Exit** quits
- [ ] Close → **Quit** ends the session
- [ ] PowerShell (no console on double-click still):

```powershell
cd <repo-root>
.\build\bin\airlink.exe --help
.\build\bin\airlink.exe -list-ifaces
go run ./cmd/airlink -- "$env:WINDIR\win.ini"
```

## Checkpoint — right-click menu (M7.10, C1, F5.8)

Use the Wails exe (not `go run`). First launch writes HKCU + a SendTo shortcut.

```powershell
cd <repo-root>
wails build
```

**CP1 — menu appears**

```powershell
& ".\build\bin\airlink.exe"
```

- [ ] Desktop / taskbar / window title bar show the Air-Link logo (not a generic icon)
- [ ] Window header: logomark + **Air-Link**, dark `#242529` background, QR black on white
- [ ] Phone page: Air-Link header + favicon (scan the QR)
- [ ] Open Settings in the window: **Right-click menu: Installed ✓**
- [ ] Shift+right-click `C:\Windows\win.ini` (or any `.txt`) → **Show more options** if needed → **Send with Air-Link**
- [ ] Right-click the same file → **Send to** → **Send with Air-Link**
- [ ] Windows 11 compact menu (no Shift) does **not** need to show it

**CP2 — app already open**

Keep the window from CP1, Allow the iPhone. Shift+right-click a photo → Send with Air-Link.

- [ ] Photo appears on the iPhone page without a new scan (timeline **→** on the PC)

**CP3 — app closed**

Quit the app (not tray). Shift+right-click a file → Send with Air-Link.

- [ ] One window opens with QR and the file queued on the timeline

**CP4 — 20 files**

Select 20 files in Explorer → Shift+right-click → Send with Air-Link.

- [ ] One window (not 20)
- [ ] Timeline lists 20 items (or one zip of 20 — both OK as long as it is a single session)

Repair / remove:

```powershell
& ".\build\bin\airlink.exe" --install-menu
& ".\build\bin\airlink.exe" --uninstall-menu
```
