# Spike notes (historical)

Throwaway experiments used to live under `spikes/`. They were never imported by the app. Results that still matter for v1:

## Web Share over LAN HTTP

A LAN-IP `http://` page with JPEG + MP4 was opened in iPhone Safari.

| Check | Result |
| --- | --- |
| `window.isSecureContext` | false |
| `navigator.share` / `canShare({files})` | absent / false |
| Save to Photos | Worked via Download / long-press (non–Web Share) |

**Decision:** Feature 1.2 (share sheet) does not work over plain HTTP on a LAN IP. The phone page must treat HTTP fallbacks as the default. HTTPS is a prerequisite for share-sheet saves, not optional polish.

## LAN HTTPS

Compared (a) an ephemeral self-signed cert with the LAN IP in the SAN vs (b) a real cert on a DNS name that resolves to the LAN IP.

| Option | Safari | `navigator.share` with files | Internet |
| --- | --- | --- | --- |
| Self-signed | “Not Private”; Continue does not install trust | Typically still missing until the cert is in the trust store | Not required |
| Real cert + DNS-01 | Normal lock if the name matches | Likely works | Needed to issue/renew; transfers stay on LAN |

Installing a custom CA on every iPhone is extra setup, not “scan and go”. DHCP IP changes break a DNS A record until it is updated.

**v1 decision (M4.5):** do not bake HTTPS into the app. Keep HTTP Download / long-press as the iPhone save path. Revisit only with a real hostname + DNS-01 (or a trusted local CA you are willing to install on each phone).
