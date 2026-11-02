# PRD: Air-Link via QR

Oct 27, 2026 · @Muhammad Sinan

## Overview

We are building a tiny desktop utility that sends anything from a computer to any phone in under 10 seconds: right-click, scan a QR code, done. No app on the phone, no account, no cloud. It runs a temporary local web server on the PC and shows a QR code that opens a page in the phone's browser.

**Problem.** Moving a file, a link or a snippet of text from PC to phone is still clumsy. Cables need drivers, cloud apps need logins and upload time, and Bluetooth fails often. Platform tools are split by ecosystem: Google Quick Share serves Android only, AirDrop serves Apple only. Existing QR tools prove the idea works, but they are terminal-only, send-only, or treat iPhones as an afterthought.

**Our bet.** Win on four add-ons the competition lacks:

1. **iPhone-first saving and media:** files land in Photos or Files properly, and videos stream before they finish downloading.
2. **Send more than files:** text, links, clipboard, screenshots and Wi-Fi credentials.
3. **Two-way with one QR:** the same page both downloads from the PC and uploads back to it.
4. **Phone starts the transfer:** after pairing once, the phone sends files or text to the PC without scanning a QR each time (Shortcuts / share sheet).
5. **Connected room:** open the app with nothing selected, scan once, and either side can send any number of times until the session ends.

## Goals, non-goals and success metrics

**Goals**

- First successful transfer within 60 seconds of installing, with zero configuration.
- Any phone with a camera and a browser works: iOS 16+ and Android 10+.
- Every transfer stays on the local network; nothing touches a server we run.

**Non-goals (v1)**

- Transfers over the internet or across networks (no relay or TURN server).
- Continuous folder sync (Syncthing territory).
- A native phone app.
- PC-to-PC transfers.

**Success metrics** (targets are proposals; confirm before launch)

| Metric | Target | How measured |
| --- | --- | --- |
| Time from right-click to download start | ≤ 10 s median | Local timing in debug builds and user tests |
| First-try success rate | ≥ 90% of sessions | Opt-in anonymous telemetry, or user testing if no telemetry |
| iPhone photo/video saved to Photos | ≥ 95% of attempts on iOS 16+ | Manual test matrix |
| Install size | ≤ 15 MB, single binary | Release artifact |
| GitHub stars in 6 months | 1,000 | GitHub |

## Target users and use cases

The primary user is a Windows PC owner with an iPhone or mixed-ecosystem household, who moves something to their phone several times a week and won't install heavy sync software.

| Persona | Typical moment | What they send |
| --- | --- | --- |
| Everyday PC + iPhone user | Downloaded a PDF ticket, needs it at the venue | Files, PDFs |
| Content creator | Exported a video edit, wants to post from phone | Large videos, photos |
| Developer / tester | Built an APK or test page, needs it on a device | Builds, URLs |
| Office worker | Reading a long article, wants to finish on commute | Links, selected text |
| Host with guests | Guest asks for the Wi-Fi password | Wi-Fi QR |
| Student | Photographed whiteboard on phone, needs it in notes | Phone → PC upload |
| Repeat sender | Shares photos from the phone several times a day | Paired phone → PC, no new QR |

## Competitive landscape and positioning

The core mechanic is already proven by several tools, so we compete on the three add-ons, not on the idea.

| Product | Phone app needed | PC interface | Two-way | Beyond files | iPhone experience |
| --- | --- | --- | --- | --- | --- |
| [PhoneDrop](https://github.com/willmanstoolbox/phonedrop) | No | GUI, Windows right-click, Linux Open With | Send only | No | Plain browser download |
| [qrcp](https://github.com/claudiodangelis/qrcp) | No | Terminal | Yes, separate send and receive modes | No | Plain browser download |
| qrSend (older) | Yes, plus a scanner app | Right-click | No | No | Android only |
| [LocalSend](https://localsend.org) | Yes | App | Yes | Text | Native app |
| [PairDrop](https://github.com/schlagmichdoch/pairdrop) | No (web app) | Browser tab | Yes | Text | Browser |
| [Google Quick Share](https://android.com/windows/better-together/quick-share-app) | Built into Android | App, Google sign-in | Yes | No | Not supported |

**Positioning statement:** for people who move things between a PC and a phone, our tool is the fastest no-app way to do it, and the only one that saves properly to an iPhone's Photos, sends text and links as well as files, keeps a two-way room after one scan, and lets a paired phone send again without scanning.

## Core MVP: right-click → local server → QR

The baseline every competitor has; it must be flawless before the add-ons matter.

**User flow**

1. User right-clicks one or more files or a folder and picks **Send with Air-Link**.
2. A small window shows a QR code, the file names, total size and a countdown to expiry.
3. User scans the QR with the phone's camera; the browser opens a download page.
4. The PC window shows "Phone connected" and live progress.
5. When the transfer ends, the window shows "Done" and the server shuts down.

**Requirements**

| ID | Requirement | Priority |
| --- | --- | --- |
| C1 | Context-menu entry in the Windows 11 modern menu (not only under "Show more options"), plus Windows 10 | P0 |
| C2 | macOS Finder Quick Action; Linux Nautilus/Dolphin script | P1 |
| C3 | Auto-select the correct LAN interface; skip VPN, Docker, Hyper-V and WSL adapters | P0 |
| C4 | Random port per session. The QR encodes a one-time join code, not a long-lived session token | P0 |
| C5 | Folders and multi-select stream as a zip, created on the fly (no temp file) | P0 |
| C6 | HTTP range requests so interrupted downloads resume | P0 |
| C7 | Auto-shutdown after transfer or after 10 minutes idle (configurable) | P0 |
| C8 | Keep-alive mode to serve several phones | P1 |
| C9 | Single portable binary; optional installer only adds menu entries | P0 |
| C10 | Network diagnostics: if no phone connects in 20 s, show likely cause (firewall, guest network isolation, wrong Wi-Fi) and a fix | P0 |
| C11 | Hotspot fallback hint: "Turn on your phone's hotspot and connect this PC" | P1 |

## Feature 1: iPhone-first saving and media experience

A photo or video sent from the PC lands in the iPhone's Photos app in two taps, instead of getting lost in Files > Downloads. This is our headline differentiator.

**Why it matters.** Plain browser downloads on iOS Safari go to the Files app, and videos often fail to save to Photos. Quick Share doesn't serve iPhones at all, so this gap has no good free answer today.

**User story.** As an iPhone user, when I send a video from my PC, I want to tap "Save to Photos" and have it appear in my camera roll, so I can post or edit it right away.

**Requirements**

| ID | Requirement | Priority |
| --- | --- | --- |
| F1.1 | Landing page detects iOS vs Android and adapts its buttons | P0 |
| F1.2 | On iOS, a "Save" button uses the Web Share API (navigator.share with files), opening the share sheet with Save Image / Save Video | P0 |
| F1.3 | Fallback when the Web Share API fails (e.g. file too large for memory): standard download plus a one-line guide to Files > Downloads | P0 |
| F1.4 | Thumbnail gallery for multiple images; select some or Save all | P0 |
| F1.5 | Videos stream in the page player via range requests, so preview starts before the download finishes | P1 |
| F1.6 | HEIC/HEVC and other formats keep the original file; never re-encode | P0 |
| F1.7 | Android: same page, standard download goes to Downloads; share sheet optional | P0 |
| F1.8 | Optional "Add to Home Screen" so returning users open a page that remembers recent sessions | P2 |

**Acceptance criteria**

- On iOS 16+ Safari, a 20 MB JPEG and a 200 MB MP4 each reach Photos in ≤ 3 taps **when a secure context is available** (HTTPS / share sheet), **or** via documented HTTP fallbacks (long-press Save to Photos / Download + guide) when it is not.
- If the share sheet is missing or fails, the user sees the fallback within 2 s, never a blank page.

**Spike result (2026-10-29, iPhone Safari, `http://192.168.1.3:8080`):**

| Check | Result |
| --- | --- |
| `window.isSecureContext` | **false** |
| `navigator.share` | **false** (API absent) |
| `navigator.canShare` | **false** |
| Protocol / host | `http:` / LAN IP |
| Save to Photos | User reports download/save to Photos **worked** via non–Web Share path (HTTP fallback) |

**Conclusion:** F1.2 (Web Share → share sheet) **does not work over plain HTTP on a LAN IP** on this device. M1 must treat **F1.3 fallbacks as the default on HTTP**, and treat HTTPS (or equivalent secure context) as a prerequisite for the share-sheet path—not an optional polish.

**Open questions:** (1) Web Share file-size ceiling once HTTPS is adopted later. **v1 HTTPS decision (M4.5):** do **not** bake HTTPS into the app; keep HTTP fallbacks (see `docs/SPIKES.md`).

## Feature 2: Send more than files

Text, links, the clipboard, a screenshot or Wi-Fi details go to the phone with the same right-click-and-scan gesture. Most competitors are file-only, and these small items are what people send most often.

**User story.** As someone reading an article on my PC, I want to right-click the link and open it on my phone in one scan, without emailing it to myself.

**Requirements**

| ID | Content type | Entry point | Phone result | Priority |
| --- | --- | --- | --- | --- |
| F2.1 | Short text or URL (≤ 300 chars, so the QR stays easy to scan) | Tray menu "Send clipboard", hotkey | Encoded directly in the QR, no server needed; URLs open in the browser | P0 |
| F2.2 | Long text or clipboard (> 300 chars) | Tray menu, hotkey | Page with the text and a Copy button | P0 |
| F2.3 | Clipboard image | Tray menu, hotkey | Treated as a file; uses Feature 1 saving | P0 |
| F2.4 | Screenshot (full or region) | Global hotkey, e.g. Ctrl+Shift+P | Captured, then sent as an image | P1 |
| F2.5 | Wi-Fi credentials | Tray menu "Share Wi-Fi" | Standard WIFI: QR so the phone joins in one tap | P1 |
| F2.6 | Browser link or selection | Optional browser extension | Same as F2.1 / F2.2 | P2 |

**Rules**

- Hotkeys are configurable and off by default for screenshots.
- Wi-Fi sharing reads the password only after the user confirms, and never stores it.
- Text pages are served once and expire with the session.

**Acceptance criteria:** a copied URL opens on the phone within 5 s of pressing the hotkey, with no network server involved when it fits in the QR.

## Feature 3: Two-way transfer with one QR

The page a phone opens to download from the PC also has a drop zone to send files and text back. qrcp needs two separate modes for this; we make it one session and one scan.

**User stories**

- As a student, I want to scan once and send whiteboard photos from my phone to my PC.
- As a user who just received a file, I want to send a reply photo back without starting a new session.
- As a user with nothing selected, I want a tray option "Receive from phone" that shows a QR for upload only.

**Requirements**

| ID | Requirement | Priority |
| --- | --- | --- |
| F3.1 | Every session page shows a "Send to PC" area: pick photos/files, or paste text | P0 |
| F3.2 | Tray action "Receive from phone" opens an upload-only session | P0 |
| F3.3 | Incoming items need approval on the PC ("iPhone wants to send 4 photos, 38 MB: Accept / Decline"), unless the user turns on auto-accept for that session | P0 |
| F3.4 | Files save to a configurable folder (default Downloads\\Air-Link); name clashes get a suffix, never overwrite | P0 |
| F3.5 | Uploads are chunked and resumable, with progress on both screens | P1 |
| F3.6 | Text sent from the phone goes to the PC clipboard, with a notification | P1 |
| F3.7 | "Open folder" button and toast notification when an upload finishes | P0 |
| F3.8 | Keep original photo formats (HEIC stays HEIC); optional "Convert to JPG" setting | P1 |

**Acceptance criteria:** from one scan, a user downloads a file and uploads 10 photos (≥ 50 MB total) back, with the approval prompt shown once.

## Feature 4: Phone starts the transfer

After pairing once, the phone can send files or text to the PC without scanning a QR each time. No native phone app: iPhone Shortcuts (share sheet) and a home-screen page; Android uses the same HTTP API.

**User stories**

- As a student, I want to share a whiteboard photo from the iOS share sheet straight to my PC.
- As a repeat user, I do not want to walk to the PC and scan a QR for every upload.
- As a host of a long-running receiver, I want to turn receive mode off in one click.

**Requirements**

| ID | Requirement | Priority |
| --- | --- | --- |
| F4.1 | Tray “Receive mode: on/off”; optional start with Windows; uses the same upload pipeline as Feature 3 | P0 |
| F4.2 | “Pair a phone” shows a QR once; phone gets a 256-bit device token; only paired devices can upload; tray list with rename and Remove; PC stores token hashes only | P0 |
| F4.3 | Advertise the PC as `<hostname>.local` (mDNS); keep the LAN IP as a fallback; document what happens when the IP changes | P0 |
| F4.4 | `POST /api/upload` accepts multipart files and plain text with `Authorization: Bearer <device token>`; `docs/IPHONE_SHORTCUT.md` steps to build a share-sheet Shortcut; after pairing, the phone page shows those steps with this user’s URL and token filled in | P0 |
| F4.5 | The upload page works when saved with Add to Home Screen and remembers the device token | P0 |
| F4.6 | Document Android HTTP Shortcuts setup for the same endpoint | P1 |
| F4.7 | PC notification on every received item with Open folder | P0 |
| F4.8 | Rate limiting; configurable max upload size; never log tokens or Bearer values; bind receive mode to the LAN interface only; receive mode off in one click | P0 |

**Acceptance criteria:** after pairing once, share a photo from the iPhone share sheet to “Send to PC” with no new QR; the file appears in the PC receive folder; an unpaired request is rejected.

## Feature 5: Connected room — scan once, send both ways

Open the app with no file selected. A QR is already on screen. The phone scans once, then either side can send files or text, any number of times, in any order, until someone ends the session or idle expires. This is the live room on top of Features 1, 3 and 4 — not a second server.

**User stories**

- As someone at a desk, I want to open the app, scan once, and keep sending photos and files both ways without walking back to the PC.
- As a phone user, I want new items from the PC to appear on the page I already have open, with the same Save / Download behaviour as Feature 1.
- As a PC user, I want drop, a Send files button, and right-click “Send with Air-Link” to feed the *same* room if the app is already running.
- As someone whose phone locked, I want the page to reconnect and show anything I missed.

**Requirements**

| ID | Requirement | Priority |
| --- | --- | --- |
| F5.1 | App start with no arguments opens the desktop window with a single-use join QR and a “Waiting for phone…” state (no file required). Each QR is one device; after a scan (or 2 minutes unused) a new QR appears | P0 |
| F5.2 | Phone page opens a WebSocket on that device’s token (PIN if enabled). Both screens show “Connected” with the device name; the PC lists connected phones and Allow / Deny for new scans | P0 |
| F5.3 | PC → phone: drag-and-drop onto the window, a “Send files” button, and right-click → Send with Air-Link all add items to the room. The phone page shows each new item instantly with Save/Download using Feature 1 | P0 |
| F5.4 | Phone → PC: “Send to PC” on the same page for files and text, using Feature 3 upload code, visible in the shared timeline | P0 |
| F5.5 | One shared timeline on PC and phone: who sent what, size, time, progress, done or failed | P0 |
| F5.6 | If the phone locks or backgrounds, the page reconnects when reopened (device token in the URL / localStorage) and shows missed items. Paired devices (Feature 4) can rejoin without scanning | P0 |
| F5.7 | “End session” on both sides; auto-end after configurable idle (default 30 minutes). After the session ends, join QRs and device tokens stop working. “Refresh QR” only replaces the unused join code and does not drop connected devices | P0 |
| F5.8 | If the app is already running, right-click “Send with Air-Link” adds the files to the open room instead of starting a new session (single instance) | P0 |
| F5.9 | WebSocket requires that device’s token (not a shared session token), checks `Origin` and `Host`, binds only to the LAN interface, and limits message size. Never log join codes, device tokens, or full URLs. Reuse serving, uploads, PIN, device list and pairing — do not duplicate them | P0 |
| F5.10 | Join QR is single-use: scan swaps the join code for a per-device token; the code is invalid immediately; unused codes expire in 2 minutes; two phones scanning the same code: first wins, second sees “already used”; PC Allow / Deny (Auto-allow off by default); Kick/Remove is per device; rate-limit join attempts | P0 |

**Rules**

- Room mode is the no-args default. Passing file paths still starts (or, if a room is open, joins) a transfer as today.
- Idle default for a room is 30 minutes (existing file-only sessions may keep their shorter default).
- WebSocket is `ws://` on the same LAN HTTP server as the QR (v1 stays HTTP). Ask before adding a WebSocket library; the Go standard library has no WebSocket.
- Transfers stay streamed; the timeline is metadata, not a second copy of the file in memory.

**Acceptance criteria:** open the app with no args → scan a join QR → PC Allow → PC sends a file into the room and the phone can save it (Feature 1) → a second phone needs a new QR → phone sends a photo back in the same session (Feature 3) → lock the phone, reopen, missed items are there → End session (or idle) makes the old QR 404.

## Non-functional requirements, security and architecture

One small native binary runs a short-lived HTTP server bound to the LAN, and the phone needs nothing but its browser.

**Security and privacy**

- The QR encodes a 128-bit one-time join code (`/j/{code}`), never a long-lived session token. Scanning swaps it for a 256-bit per-device token (`/t/{deviceToken}/`). The join code is invalid immediately; unused codes expire after 2 minutes.
- Two phones scanning the same code: only the first claim wins; the second is told the code was already used. Invalid/expired/used join attempts are rate-limited (same budget as other auth failures).
- New scans wait for **Allow / Deny** on the PC unless **Auto-allow** is on (off by default). Kick or Remove applies only to that device’s token.
- Optional 4-digit PIN shown on the PC, required on the phone page after join.
- The server binds only to the chosen LAN interface, never 0.0.0.0 by default.
- The PC shows every connected or pending device (IP, browser) and can allow, deny, or kick it.
- Paired-device tokens are 256-bit; the PC stores hashes only and never logs join codes, device tokens, or `Authorization` headers. Paired phones may rejoin the room without scanning.
- Receive-mode uploads are accepted only from paired devices; receive mode can be turned off in one click.
- Room WebSocket: that device’s token (and PIN if enabled); validate `Origin` and `Host`; LAN bind only; limit message size.
- No telemetry by default; any analytics is opt-in, anonymous, and documented.
- No outbound internet calls except an optional update check.

**Performance**

- Transfer speed ≥ 80% of what a plain HTTP download achieves on the same Wi-Fi.
- Cold start to QR on screen ≤ 1 s.
- Memory use ≤ 50 MB during a 4 GB transfer (streamed, never loaded whole).

**Suggested stack** (open to change)

| Layer | Choice | Reason |
| --- | --- | --- |
| Core server | Go or Rust | Single static binary, easy cross-compile, strong HTTP libraries |
| Desktop UI | Tray icon + small native window (Wails or Tauri) | Small footprint |
| Phone page | Plain HTML/JS embedded in the binary, no framework | Loads fast on any phone |
| Windows menu | IExplorerCommand shell extension for Windows 11, registry verb for Windows 10 | Appears in the modern menu |
| QR | Built-in encoder | No dependencies |

**HTTPS note:** browsers treat a LAN IP over plain HTTP as an insecure context. The Web Share API needs a secure context on some browsers, which is a key risk for Feature 1 (see Risks).

## Milestones, risks and open questions

Ship the core first, then the add-ons in order of how strongly they set us apart.

&#91;embedded content: roadmap · 4 phases, 2 gates\]

The iOS save spike gates the beta because Feature 1 is the main reason to choose this tool.

**Risks**

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Web Share API needs HTTPS; a LAN IP over HTTP is **not** a secure context on iOS | Feature 1 share-sheet path unavailable on plain HTTP (**confirmed** 2026-10-29 spike) | **v1:** HTTP fallbacks only (M4.5). Revisit LAN HTTPS (DNS-01 or per-phone CA) after v1 if share-sheet ≤3 taps is required |
| Guest Wi-Fi or office networks block device-to-device traffic | Phone can't connect | Diagnostics (C10) and hotspot fallback (C11) |
| Windows Defender SmartScreen warns on an unsigned binary | Users abandon install | Code-sign releases; publish via winget and Scoop |
| Windows firewall prompt on first run | Confusing first run | Explain the prompt before it appears; request private-network access only |
| PhoneDrop or qrcp adds the same features | Weaker differentiation | Move fast on Feature 1; build a reputation on iPhone polish |
| iOS Shortcuts / HTTP Shortcuts need a stable URL | Pairing breaks after DHCP | mDNS `<hostname>.local` plus IP fallback; pairing page explains IP change |
| mDNS `.local` flaky on some Windows/guest Wi-Fi | Shortcut cannot resolve the PC | Keep LAN IP fallback; document both URLs |
| iOS Safari drops WebSockets when the phone locks or backgrounds | Room looks dead; items missed | Auto-reconnect + replay missed timeline (F5.6); HTTP download still works if WS is down |

**Open questions**

- [ ] Product name and domain?
- [ ] Open source (MIT) or freemium with a paid Pro tier?
- [ ] Go or Rust for the core?
- [ ] Is opt-in telemetry acceptable, or user testing only?
- [ ] Which Windows versions to support at launch: 10 and 11, or 11 only?
