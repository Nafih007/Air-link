# Tasks — Milestone M0 (Core MVP)

Work one task at a time. Tick the box when done and note which PRD IDs it covers.
Order: each task leaves something you can run and verify before the next begins.

## M0 — right-click → local server → QR (C1–C11)

- [x] **M0.1 — CLI stub and session args**  
  Accept one or more file/folder paths as CLI args (and `--help` / version). Print chosen paths and exit 0; no server yet.  
  **Test:** `go run ./cmd/airlink -- path\to\file.txt` prints the path; `--help` works.  
  **PRD:** C9 (portable CLI entry point) — done

- [x] **M0.2 — Tokenised single-file HTTP serve**  
  Bind to an explicit `--bind` IP (not `0.0.0.0`), random port, 128-bit URL token; stream one file; wrong/missing token → 404. Never log the token.  
  **Test:** From phone on same Wi-Fi, open the printed URL (copy IP:port by hand) and download the file; a URL without the token returns 404.  
  **PRD:** C4 — done

- [x] **M0.3 — Auto-select LAN interface**  
  Detect a usable LAN IPv4; skip VPN, Docker, Hyper-V and WSL adapters; default bind uses that IP. Add a flag to list candidates.  
  **Test:** Run with auto-bind; phone opens `http://<shown-ip>:…` and downloads. Confirm a VPN-only machine still offers a sensible fallback or a clear error.  
  **PRD:** C3 — done

- [x] **M0.4 — QR for the session URL**  
  Generate a QR encoding the session URL (PNG and/or terminal). Print the bare host/port for humans; do not print the full tokenised URL in logs.  
  **Test:** Scan the QR with the phone camera; Safari/Chrome opens the download and the file saves.  
  **PRD:** C4 — done

- [x] **M0.5 — Embedded phone download page**  
  Plain HTML/CSS/JS in `web/`, embedded with `go:embed`. Token-gated page shows file name, size and a Download button.  
  **Test:** Scan QR → page loads on phone → tap Download → file arrives.  
  **PRD:** C4, C9 — done

- [x] **M0.6 — Desktop session window**  
  Small PC window: QR, file names, total size, idle/expiry countdown. Closing the window stops the server.  
  **Test:** Launch via CLI with a file; window appears; phone still downloads; closing the window ends the session.  
  **PRD:** C7 (countdown visible; shutdown on close) — done (WebView2)

- [x] **M0.7 — Multi-select and folder zip streaming**  
  Multiple files and/or folders stream as a zip built on the fly (no temp zip on disk).  
  **Test:** Pass a folder plus two files; phone downloads a `.zip` that extracts correctly.  
  **PRD:** C5 — done

- [x] **M0.8 — HTTP range requests**  
  Support `Range` so interrupted downloads resume and media can seek.  
  **Test:** Start a large download, cancel mid-way, resume; on phone, scrub a video in the browser player.  
  **PRD:** C6 — done (single-file; zip rejects Range)

- [x] **M0.9 — Auto-shutdown after transfer or idle**  
  Shut down after successful transfer, or after configurable idle (default 10 minutes). Show “Done” briefly before exit.  
  **Test:** One successful download → window shows Done and exits; with a short `--idle` flag and no phone, process exits when idle elapses.  
  **PRD:** C7 — done

- [x] **M0.10 — Network diagnostics + hotspot hint**  
  If no phone connects within ~20 s, show likely causes (firewall, guest isolation, wrong Wi-Fi) and the hotspot fallback tip.  
  **Test:** Start a session with phone on a different network (or firewall blocking inbound); within ~20 s the PC shows diagnostics including the hotspot hint.  
  **PRD:** C10, C11 — done

- [x] **M0.11 — Keep-alive mode**  
  Optional keep-alive so several phones can use the same session until the user stops it (no auto-shutdown after first transfer).  
  **Test:** Start with keep-alive; download from phone A then phone B without restarting; stop manually from the window.  
  **PRD:** C8 — done

- [x] **M0.12 — Windows classic registry context menu**  
  Register a classic shell verb “Send with Air-Link” (Works on Windows 10 and under “Show more options” on Windows 11). Launch the binary with selected paths.  
  **Test:** Right-click a file → Send with Air-Link → session window + QR appear; phone downloads.  
  **PRD:** C1 (classic registry path first) — done

- [x] **M0.13 — Portable binary + Win11 modern menu / non-Windows stubs**  
  Ship one portable Windows binary (no runtime). Add or spike the Windows 11 modern-menu entry (`IExplorerCommand`); stub or document macOS Finder Quick Action and Linux Nautilus/Dolphin scripts. Optional installer only adds menu entries.  
  **Test:** Copy the `.exe` alone to another folder and run a transfer; confirm modern-menu entry if implemented, or document the remaining C1/C2 gap.  
  **PRD:** C1, C2, C9 — done (classic + Win11 sparse package in M4.7)

## Gate — iOS Web Share spike (before M1)

- [x] **SPIKE.1 — Confirm / refresh LAN HTTP Web Share**  
  LAN-IP HTTP page with JPEG + MP4, Share/Save (`navigator.share`), Download, long-press note; show `isSecureContext`, `canShare`, on-page errors; terminal URL + QR.  
  **Test:** Run spike; scan on iPhone; record Share vs Download vs long-press results. **STOP** — wait for iPhone results, then update PRD Feature 1 + Risks before M1.  
  **PRD:** Feature 1 risk (HTTPS / secure context) — done: HTTP → no Web Share; Photos via fallback worked; PRD updated 2026-10-29

## M1 — iPhone-first saving (F1.x)

- [x] **M1.1 — OS detection on phone page**  
  Detect iOS vs Android (and desktop) in `web/`; adapt primary button labels.  
  **Test:** Open session on iPhone Safari and Android Chrome; buttons match platform.  
  **PRD:** F1.1 — done

- [x] **M1.2 — Web Share “Save” for images**  
  On iOS, Save uses `navigator.share({ files })` for a single image; show errors on page.  
  **Test:** Send ~20 MB JPEG; ≤ 3 taps to Photos (or clear failure).  
  **PRD:** F1.2 — done (button only when secure context + share exist)

- [x] **M1.3 — Share fallback**  
  If share fails or `canShare` is false within ~2 s, show download + Files > Downloads guide (never blank).  
  **Test:** Force failure (huge file or insecure context); fallback appears quickly.  
  **PRD:** F1.3 — done (HTTP shows fallback immediately)

- [x] **M1.4 — Video share + stream preview**  
  Same Save/share path for video; in-page player uses range requests so preview starts early.  
  **Test:** ~200 MB MP4 — scrub before full download; try Save to Photos.  
  **PRD:** F1.2, F1.5 — done

- [x] **M1.5 — Keep original formats**  
  Serve HEIC/HEVC/etc. unchanged; never re-encode.  
  **Test:** Send a HEIC (or similarly exotic) file; phone receives identical bytes/extension.  
  **PRD:** F1.6 — done (mime by extension only)

- [x] **M1.6 — Multi-image gallery**  
  Thumbnail gallery for multiple images; select some or Save all.  
  **Test:** Send 4 images; select 2 and save; Save all works.  
  **PRD:** F1.4 — done

- [x] **M1.7 — Android parity**  
  Same page; primary action is standard download; share sheet optional.  
  **Test:** Android Chrome downloads to Downloads; optional share works if available.  
  **PRD:** F1.7 — done

- [x] **M1.8 — (Optional P2) Add to Home Screen stub**  
  Document or minimal PWA hint; remember recent sessions only if cheap. Skip if time-boxed.  
  **Test:** Manual note in TESTING.md; or smoke “Add to Home Screen” prompt.  
  **PRD:** F1.8 — skipped (P2); noted in TESTING.md

## M2 — Beyond files (F2.x)

- [x] **M2.1 — Short clipboard / URL in QR**  
  CLI `-clipboard` / `-text`: if ≤ 300 chars, encode in QR with no server; URLs open in browser.  
  **Test:** Copy a short URL; `-clipboard`; phone opens it without a download server.  
  **PRD:** F2.1

- [x] **M2.2 — Long text page**  
  Longer clipboard text served on a tokenised page with Copy button.  
  **Test:** `-text` with a long paragraph; scan; Copy on phone works; session expires with idle rules.  
  **PRD:** F2.2

- [x] **M2.3 — Clipboard image**  
  Clipboard image treated as a file; reuses Feature 1 saving.  
  **Test:** Copy an image; `-clipboard-image`; iPhone Save path works like a normal image file.  
  **PRD:** F2.3

- [x] **M2.4 — Screenshot hotkey**  
  CLI `-screenshot` captures fullscreen then sends as PNG (global hotkey deferred — needs a dependency; ask before adding).  
  **Test:** `-screenshot`; QR appears; phone gets the screenshot.  
  **PRD:** F2.4

- [x] **M2.5 — Share Wi-Fi**  
  CLI `-wifi`; confirm before reading password; WIFI: QR; never store password.  
  **Test:** Confirm prompt; phone joins (or shows WIFI QR contents) without password left on disk.  
  **PRD:** F2.5

- [ ] **M2.6 — (Optional P2) Browser extension stub**  
  Deferred for v1 — use `-text` / `-clipboard` instead.  
  **Test:** Doc-only OK for v1.  
  **PRD:** F2.6

## M3 — Two-way transfer (F3.x)

- [x] **M3.1 — Send-to-PC drop zone on session page**  
  Every download page also has pick photos/files + paste text; POST `/upload` (pending unless `-auto-accept`).  
  **Test:** Scan once; Send to PC one photo with `-auto-accept` (approval UI is M3.3).  
  **PRD:** F3.1

- [x] **M3.2 — Receive-from-phone session**  
  CLI `-receive` upload-only session with QR (tray deferred with hotkey deps).  
  **Test:** `-receive -auto-accept`; phone uploads with no download payload.  
  **PRD:** F3.2

- [x] **M3.3 — PC approval prompt**  
  Incoming batch needs Accept/Decline in the session window (or CLI with `-no-window`) unless `-auto-accept`.  
  **Test:** Phone offers 2 photos; PC prompts once; Decline blocks; Accept saves.  
  **PRD:** F3.3

- [x] **M3.4 — Save folder + no overwrite**  
  Configurable `-upload-dir` (default Downloads\Air-Link); numeric suffix on clashes (done with M3.1 unique names).  
  **Test:** Upload same name twice; get `file (1).ext`; originals intact.  
  **PRD:** F3.4

- [x] **M3.5 — Open folder + toast**  
  In-window toast when upload finishes; Open folder button opens Explorer.  
  **Test:** After Accept, toast + Open folder reveals files.  
  **PRD:** F3.7

- [x] **M3.6 — Chunked resumable uploads**  
  1 MiB chunks via upload-init/chunk/finish; resume from sessionStorage; progress on phone + PC.  
  **Test:** Upload a large file; progress updates; interrupt and Send again to resume; completes.  
  **PRD:** F3.5

- [x] **M3.7 — Text to PC clipboard**  
  Phone-pasted text lands on PC clipboard with notification (not written as a file).  
  **Test:** Paste text on phone; PC clipboard matches; toast shown.  
  **PRD:** F3.6

- [x] **M3.8 — Keep HEIC + optional JPG convert**  
  Default keeps original bytes/extension; `-convert-jpg` writes JPEG (HEIC via WIA; fallback to original if convert fails).  
  **Test:** Upload HEIC; file on disk is HEIC unless `-convert-jpg` and a HEIF codec is available.  
  **PRD:** F3.8

## M4 — Ship v1

Work one task at a time. Do not start the next until the current test passes. Historical spike notes: `docs/SPIKES.md`.

- [x] **M4.0 — M0–M3 regression pass**  
  `go test ./...` plus a code check against `docs/TESTING.md` (bind, token 404, unique names). Phone boxes stay yours.  
  **Test:** Unit tests green; no default `0.0.0.0` bind; window HTML has no `/t/` token path.  
  **PRD:** C3–C7, security (token, bind)

### Security

- [x] **M4.1 — Optional 4-digit PIN**  
  Session flag (off by default) shows a PIN on the PC; phone page requires it before content. Lock the session after 5 wrong attempts.  
  **Test:** Correct PIN loads the page; 5 wrong PINs → locked for that session; a new session gets a new PIN.  
  **PRD:** Security (optional PIN)

- [x] **M4.2 — Connected-device list + Kick**  
  Desktop window lists each connected device (IP + browser/UA). Kick revokes that client so later requests fail.  
  **Test:** Phone appears after it opens the page; Kick → refresh/download fails; another device can still use the session.  
  **PRD:** Security (connected devices)

- [x] **M4.3 — Rate-limit token and PIN guesses**  
  Per-IP limit on bad tokens and wrong PINs (20/min → 429); PIN guesses share that counter.  
  **Test:** A burst of wrong-token URLs is throttled; a normal first request still succeeds.  
  **PRD:** Security (rate-limit)

- [ ] **M4.4 — Security checkpoint (STOP)**  
  Hand-test on your LAN: good PIN, wrong PIN, lockout, Kick.  
  **Test:** PIN works; wrong PIN rejected; Kick drops that phone.  
  **PRD:** Security section  
  **STOP**

### LAN HTTPS (spike only)

- [x] **M4.5 — LAN HTTPS spike (STOP)**  
  HTTPS spike only — no changes to the main app. Compare (a) self-signed cert vs (b) a real cert on a DNS name that resolves to the LAN IP. For each: setup steps, what iPhone Safari shows, whether `navigator.share` with files works, whether it needs internet. End with a recommendation: adopt one, or drop HTTPS and polish HTTP fallback.  
  **Test:** Spike serves on the LAN; you try both options on iPhone Safari.  
  **PRD:** Feature 1 HTTPS risk, F1.2 — v1 decision recorded: no app HTTPS (`docs/SPIKES.md`)  
  **STOP** — wait for your phone spike if you still want to try Safari; app HTTPS stays off

### Windows 11 menu (C1)

- [x] **M4.6 — Modern menu plan (STOP)**  
  Write what a top-level Win11 “Send with Air-Link” needs (sparse package / `IExplorerCommand`, signing) and how Win10 still uses the classic HKCU verb. No install yet.  
  **Test:** You read the plan and say go.  
  **PRD:** C1  
  **STOP**

- [x] **M4.7 — Win11 top-level “Send with Air-Link”**  
  `IExplorerCommand` DLL + sparse package from `-install-shell`; classic verb kept for Win10 / “Show more options”.  
  **Test:** On Windows 11, right-click a file (top-level menu, not “Show more options”) shows **Send with Air-Link** and starts a session.  
  **PRD:** C1 — done (needs Developer Mode / self-signed sparse package; see `docs/WIN11_MENU.md`)

### Tray + hotkey (F2.4)

- [x] **M4.8 — Propose tray/hotkey dependency (STOP)**  
  Name the library, why stdlib is not enough, approximate binary size, and the fallback if you say no. Do not add the module until you approve.  
  **Test:** Proposal in chat (or a short doc); no `go get` yet.  
  **PRD:** F2.4  
  **STOP**

- [x] **M4.9 — Tray icon + screenshot hotkey**  
  `-tray` / `-hotkey` / `-hotkey-combo` (default Ctrl+Shift+P, off unless `-hotkey`); conflict shows a message box.  
  **Test:** Enable hotkey; capture → QR; conflict path shows a clear message.  
  **PRD:** F2.4

### Release

- [x] **M4.10 — Versioned release `.exe`**  
  `-version` from build-time ldflags; documented `go build -ldflags "-s -w -X …"` for a smaller Windows binary.  
  **Test:** Built exe `-version` prints the embedded version; file is smaller than an unstripped debug build.  
  **PRD:** C9

- [x] **M4.11 — README**  
  Install, usage, troubleshooting (same Wi-Fi, Private network, firewall, VPN), slot for a demo GIF.  
  **Test:** Someone following README can send one file.  
  **PRD:** C9, first-run ≤ 60 s

- [x] **M4.12 — GitHub Actions release**  
  Workflow: on version tag, build the Windows `.exe` and attach it to a GitHub Release.  
  **Test:** Tag (or workflow_dispatch) produces a Release with the exe attached.  
  **PRD:** C9

- [x] **M4.13 — Code-signing notes**  
  Options and costs (Authenticode / EV, SmartScreen). Do not buy anything.  
  **Test:** Notes exist in docs; no purchase.  
  **PRD:** Risks (SmartScreen)

## M5 — Phone starts the transfer (F4.x)

Work one task at a time. Reuse Feature 3 upload code. Ask before adding a dependency (mDNS / tray if M4.9 is not done). Never log device tokens or `Authorization` headers.

- [x] **M5.1 — Background receive mode**  
  `-receive` stays up (idle off unless `-idle`); port **17612**; tray Receive on/off; `-install-startup` / tray **Start with Windows**. Bind LAN only.  
  **Test:** Turn on; paired phone can upload; turn off → uploads fail.  
  **PRD:** F4.1, F4.8 (bind, one-click off)

- [x] **M5.2 — Pair a phone**  
  Pair QR → 256-bit token; hashes on disk; PC window **Rename / Remove**; `/api/upload` only with a valid Bearer. Pairing requires PIN when `-pin` is on.  
  **Test:** Pair; upload succeeds; unpaired → 401; Remove → that token fails; hashes on disk.  
  **PRD:** F4.2, F4.8 (paired only, never log tokens)

- [x] **M5.3 — Stable address (mDNS)**  
  Stdlib multicast responder for `<hostname>.local` A record; IP fallback documented.  
  **Test:** Phone opens `http://<hostname>.local:17612/home` and the IP URL.  
  **PRD:** F4.3

- [x] **M5.4 — `POST /api/upload`**  
  Multipart `files` and `text`; Bearer; rate-limit; `-max-upload` (default 512MiB).  
  **Test:** curl with a valid token saves a file; missing/wrong Bearer rejected; oversized body rejected.  
  **PRD:** F4.4, F4.8 (rate limit, max size)

- [x] **M5.5 — iPhone Shortcut docs + pairing page**  
  Write `docs/IPHONE_SHORTCUT.md`: share sheet → Receive images/files/text → Get Contents of URL (POST, form, header). After pairing, the phone page shows those steps with **this user’s URL and token** filled in.  
  **Test:** You build the Shortcut from the page; share one photo → it lands on the PC.  
  **PRD:** F4.4

- [x] **M5.6 — Add to Home Screen**  
  Stable `/home` on port 17612; `manifest.json`; localStorage token; on-page send uses Bearer `/api/upload`.  
  **Test:** Pair → open `/home` → Add to Home Screen → send a file without scanning again.  
  **PRD:** F4.5

- [x] **M5.7 — Android HTTP Shortcuts**  
  Document the same endpoint in `docs/IPHONE_SHORTCUT.md` (or a short Android section): HTTP Shortcuts app, POST, multipart, Bearer header.  
  **Test:** Doc is followable; optional smoke on an Android phone.  
  **PRD:** F4.6

- [x] **M5.8 — Notification + Open folder**  
  In-window toast + **Open folder**, plus a Windows toast when possible.  
  **Test:** Shortcut upload → toast; Open folder shows the new file.  
  **PRD:** F4.7

- [ ] **M5.9 — Feature 4 checkpoint (STOP)**  
  Pair once; send a photo from the iPhone share sheet with no new QR; unpaired request fails; Receive mode Off stops everything.  
  **Test:** PRD Feature 4 acceptance criteria.  
  **PRD:** F4.1–F4.8  
  **STOP**

## M6 — Connected room: scan once, send both ways (F5.x)

Work one task at a time. Reuse serving, uploads (F3), PIN, connected-device list, and pairing (F4). Do not duplicate that logic. Ask before adding a WebSocket library (Go stdlib has none). Never log join codes, device tokens, or full URLs. Bind only to the LAN IP.

- [x] **M6.1 — Empty start: QR, waiting**  
  No CLI args → desktop window, session QR, **Waiting for phone…** (no files required). Keep current behaviour when paths are passed and no room is open. Idle default 30 minutes for a room.  
  **Test:** `go run ./cmd/airlink` with no args opens the window + QR; closing the window exits; passing a file still starts a one-shot send if no room is running.  
  **PRD:** F5.1

- [ ] **M6.2 — Live connection (STOP)**  
  Phone page opens a WebSocket on the same token (PIN if enabled). Check `Origin` and `Host`; limit message size. Both screens show **Connected** + device name; PC lists connected phones (reuse device list).  
  **Test:** Scan the empty-start QR; both sides show Connected; wrong token / missing PIN cannot upgrade; kick still drops that phone.  
  **PRD:** F5.2, F5.9  
  **STOP** — wait for your phone check before PC→phone push

- [ ] **M6.3 — PC → phone into the room (STOP)**  
  Drag-and-drop on the window, a **Send files** button, and right-click **Send with Air-Link** enqueue items in the open room. Phone page shows each new item instantly with Save/Download (Feature 1). Stream; do not load whole files.  
  **Test:** Drop or Send files a photo + a PDF; they appear on the already-open phone page; iPhone Save/Download matches Feature 1.  
  **PRD:** F5.3  
  **STOP**

- [ ] **M6.4 — Phone → PC in the same session (STOP)**  
  Same page **Send to PC** for files and text, using Feature 3 upload (approval unless auto-accept). Items appear on both screens.  
  **Test:** Without a new QR, phone sends a photo and a text snippet; PC gets them (approval if required); room stays up so PC can send again.  
  **PRD:** F5.4  
  **STOP**

- [ ] **M6.5 — Shared timeline**  
  One timeline on PC and phone: who sent what, size, time, progress, done or failed.  
  **Test:** Mix of PC→phone and phone→PC shows in order on both sides; a failed item is marked failed, not silent.  
  **PRD:** F5.5

- [ ] **M6.6 — Reconnect + paired rejoin (STOP)**  
  Phone lock/background → reopen reconnects and shows missed items. Paired devices (M5) can rejoin the room without scanning (still LAN + token/pair rules).  
  **Test:** Send from PC while Safari is backgrounded; reopen → missed item is there. Paired phone opens the room URL (or documented rejoin) with no new QR.  
  **PRD:** F5.6  
  **STOP**

- [ ] **M6.7 — End session**  
  **End session** on PC and phone. Auto-end after idle (default **30 minutes** for a room, configurable). After end, QR and token 404.  
  **Test:** End from the phone → PC closes the room; new scan fails. Leave idle → same. Short `-idle` still works.  
  **PRD:** F5.7

- [ ] **M6.8 — Single instance**  
  If the app is already running, right-click **Send with Air-Link** adds files to the open room instead of a second process/window.  
  **Test:** Room open; right-click another file → it appears in the same room/timeline, not a second QR window.  
  **PRD:** F5.8

- [ ] **M6.9 — Feature 5 checkpoint (STOP)**  
  No-args start → one scan → PC sends → phone saves → phone sends back → reconnect shows a miss → End session kills the QR.  
  **Test:** PRD Feature 5 acceptance criteria.  
  **PRD:** F5.1–F5.9  
  **STOP**

## M7 — Desktop app (double-click window)

Do not start until **Wails v2** is approved (new module: `github.com/wailsapp/wails/v2`). Work one task at a time. The window is only a front-end: reuse `internal/server` room (M6), uploads, PIN, pairing, and the existing tray. Do not duplicate transfer logic. Never log join codes, device tokens, or full session URLs. Bind LAN only.

- [x] **M7.1 — Wails shell + room on launch**  
  Wails v2 window **Air-Link**, ~420×620. Double-click / no args starts the M6 connected room (same HTTP session). If WebView2 runtime is missing, show a clear message and the Evergreen installer link (`https://go.microsoft.com/fwlink/p/?LinkId=2124703`) instead of a blank crash. Win10 and Win11.  
  **Test:** `wails build` (or the documented script) produces an `.exe`; launch it; a window opens (or the WebView2 message).  
  **PRD:** F5.1, C9

- [x] **M7.2 — QR, URL, PIN, status (STOP CP-A)**  
  Large crisp QR (black on white, white border), **join** URL in small text + **Copy**, PIN if `-pin` / settings, status **Scan with your phone camera** → **Connected: …** and the connected-device list (reuse M4.2).  
  **Test:** Double-click the `.exe` → scannable QR; iPhone Camera/Safari opens the room; PC shows Connected.  
  **PRD:** F5.1, F5.2  
  **STOP CP-A**

- [x] **M7.3 — Send / receive from the window (STOP CP-B)**  
  Timeline of sent and received items. **Send files…** (picker), drag-and-drop onto the window, **Open received folder**, **End session / New QR**. Phone → PC uses Feature 3 upload. PC → phone uses the room (F5.3) once M6.3 exists; until then, enqueue via existing send paths if the room cannot push yet.  
  **Test:** Same session: send a file from the PC window; send a photo from the phone; both appear in the timeline; Open folder works.  
  **PRD:** F5.3–F5.5, F5.7  
  **STOP CP-B**

- [x] **M7.4 — No console + log file**  
  GUI build uses `-H=windowsgui` (Wails). Logs to `%LOCALAPPDATA%\Air-Link\logs\`; never tokens or tokenised URLs.  
  **Test:** Double-click: no console flash. After a session, a log file exists and has no `/t/<token>/`.  
  **PRD:** security (never log tokens)

- [x] **M7.5 — App icon**  
  Air-Link logo in the `.exe`; window and taskbar use it.  
  **Test:** Taskbar and window title bar show the icon (not the default Go/Wails icon).  
  **PRD:** C9

- [x] **M7.6 — Single instance**  
  Second launch focuses the existing window. Right-click **Send with Air-Link** adds files to the open room (F5.8 / M6.8).  
  **Test:** App running; launch again → one window, focused. Context-menu a file → it joins that room.  
  **PRD:** F5.8

- [x] **M7.7 — Close → tray or quit**  
  Closing the window asks **Keep receiving in the background?** Yes → minimize to tray (reuse M4.9). No → end session and quit.  
  **Test:** Close → prompt; Yes → tray icon, uploads still work; tray Exit quits.  
  **PRD:** F4.1, F5.7

- [x] **M7.8 — CLI still works (STOP CP-C)**  
  `airlink --help`, `-receive`, `-pin -- file`, etc. still work from PowerShell (attach to the parent console when args are present).  
  **Test:** From a terminal, flags work as today. Desktop shortcut: icon on Desktop + taskbar; **no console** on double-click (**CP-C**).  
  **PRD:** C9  
  **STOP CP-C**

- [x] **M7.9 — Single-use join QR (STOP)**  
  QR is a one-time join code (`/j/{code}`), not the session token. Scan swaps it for a per-device token (localStorage + URL); code dies immediately and the window shows a new QR. Unused QR expires in 2 minutes with a countdown. **Refresh QR** rotates only the unused code. New device: Allow / Deny (Auto-allow off). Kick is per device. Paired phones rejoin without scanning. Race/expired/reload handled. Never log join codes or device tokens; rate-limit join attempts.  
  **Test:** Two phones (or phone + laptop browser): first scan Allow → stays connected on reload; second scan of the *old* QR gets “already used”; new QR lets the second device in; Deny blocks; Refresh QR does not drop the first phone.  
  **PRD:** F5.1, F5.6, F5.7, F5.10, C4  
  **STOP** — two-device hand test

- [x] **M7.10 — Classic right-click + SendTo, self-register (STOP CP1–CP4)**  
  First run writes HKCU `*\shell\AirLink` and `Directory\shell\AirLink` via `golang.org/x/sys/windows/registry` (not PowerShell), strips hiding values including `ExplorerCommandHandler`, `SHChangeNotify`, SendTo `.lnk`, Settings Install/Repair/Remove, `-install-menu` / `-uninstall-menu`. Live instance batches Explorer launches (~500 ms).  
  **Test:** CP1 Shift+right-click + Send to; CP2 enqueue while connected; CP3 cold start with file; CP4 twenty files, one window.  
  **PRD:** C1, F5.8  
  **STOP** — hand test CP1–CP4

## M9 — Multi-file transfers

Do not start until you say go. Reuse `/t/{device}/file/N` streaming, chunked uploads, and the PC timeline — do not add a second HTTP server. Stream; never load whole files. Never log tokens. Update PRD **C5** when individual-file default ships (today C5 requires zip for multi-select). Folders may still zip.

- [ ] **M9.1 — Shared transfer queue**  
  One queue for both directions. Each file is its own item: name, size, direction (PC→phone / phone→PC), bytes transferred, speed, status `queued | sending | done | failed | cancelled`. Show it on the Wails window and the phone page (same metadata; poll or existing events until WS exists). Cancel an item that is queued or in flight.  
  **Test:** Send 3 files from PC and 2 from phone; both screens list 5 rows with live progress; cancel one queued item — it never starts.  
  **PRD:** F3.5, F5.5, F1.4

- [ ] **M9.2 — Parallelism (STOP)**  
  Up to **4** in-flight transfers **per device** (flag/setting, default 4). Queue the rest. Sending and receiving run at the same time in one session (no “upload waits until download finishes”). No extra zip process.  
  **Test:** Drop 6 files on the PC while the phone uploads 3 photos; Wireshark/UI shows ≤4 active per device; downloads and uploads overlap; the 5th+ stay queued until a slot frees.  
  **PRD:** F3.5, F5.3, F5.4  
  **STOP**

- [ ] **M9.3 — PC → phone: individual files by default**  
  Multi-select enqueue keeps each file at `/file/N` (Range). Do **not** wrap a multi-file send in `airlink.zip` unless the user picked a **folder** (or an explicit “Download zip” action). `EnqueuePaths` / `ResolvePaths` stay streaming.  
  **Test:** Send two JPEGs + a PDF from the window; phone lists three files, not one zip; each Download/Save gets that file; folder send still zips.  
  **PRD:** C5 (amend), F1.4, F5.3

- [ ] **M9.4 — Phone list / gallery / Save all (STOP)**  
  Phone page: list + gallery for mixed types (not images-only). **Save** per file (Web Share when secure context; else Download / long-press fallback). **Select** mode with checkboxes. **Save all** / save selected: on iPhone, `navigator.share({files})` in batches (e.g. ≤10 photos or ≤2 videos per sheet), then the next batch; if share fails or HTTP, fall back to one-by-one downloads.  
  **Test:** 12 photos + 1 video from PC; iPhone Select 12 → Save all walks batches (or fallbacks on HTTP); one Save on a single photo still works.  
  **PRD:** F1.2, F1.3, F1.4  
  **STOP**

