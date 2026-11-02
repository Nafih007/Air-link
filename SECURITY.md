# Security policy

## Supported versions

Only the latest release on [GitHub Releases](https://github.com/Nafih007/Air-link/releases) is supported. Development builds (`dev`) receive fixes on `main` but are not a supported product line.

## How to report a vulnerability

**Do not open a public GitHub issue** for security reports. Do not paste session URLs, join codes (`/j/…`), device tokens (`/t/…`), or PINs.

Use GitHub **private vulnerability reporting**:

1. Open [github.com/Nafih007/Air-link/security](https://github.com/Nafih007/Air-link/security).
2. Choose **Report a vulnerability** (Advisories).
3. Describe the issue, impact, and steps to reproduce. Include OS version and Air-Link version (`airlink.exe -version`).

The repository owner will acknowledge the report and work on a fix before any public disclosure.

If private reporting is not enabled on the repo yet, enable it under **Settings → Code security → Private vulnerability reporting**. Until then, contact the maintainer through GitHub without posting tokens or full URLs.

## In scope

- The local LAN HTTP server (bind address, token checks, range requests, zip streaming)
- Join codes, session/device tokens, and PIN gating
- Phone → PC uploads (approval, path traversal, overwrite, size limits)
- Right-click / SendTo registration (HKCU verbs, shortcut targets)
- Pairing store (`%AppData%\Air-Link\devices.json` hashes)

## Out of scope

- Attacks that require already being on the same LAN and possessing a live token (the token *is* the capability)
- Unsigned SmartScreen warnings
- iOS not offering Web Share over `http://` LAN (documented limitation)
- Third-party tools (Shortcuts, HTTP Shortcuts) misconfigured by the user

## Design notes (not bugs by themselves)

Air-Link binds one LAN IPv4, never `0.0.0.0` by default. Every session URL carries a random token; requests without it get 404. Uploads need PC approval unless auto-accept is on for that session. Uploaded names never overwrite. There is no cloud backend and no telemetry.
