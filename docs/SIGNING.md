# Code signing (M4.13)

Do **not** buy a certificate until you are ready to ship publicly. This is a cost note only.

## Why sign

Windows Defender SmartScreen warns on unsigned `.exe` downloads. Users click “More info → Run anyway”, or they leave. Signing with a certificate known to Microsoft reduces (EV) or eventually clears that warning.

## Options (2026 ballpark)

| Option | Typical cost | SmartScreen | Notes |
| --- | --- | --- | --- |
| **Don’t sign** | $0 | Warning on download | Fine for personal use / `go run`. |
| **OV Authenticode** (standard org/individual) | ~$70–400 / year (DigiCert, Sectigo, SSL.com) | Reputation builds slowly | USB or cloud HSM often required now (no exportable PFX). |
| **EV Authenticode** | ~$300–600 / year | Immediate better reputation | Hardware token; stricter identity check. Best for “download from GitHub”. |
| **Microsoft Store / MSIX** | Store fees + signing | Store handles trust | Different distribution; sparse package for Win11 menu also wants this identity. |
| **Sigstore / open-source self-serve** | $0 | Does **not** silence SmartScreen | Useful for GitHub attestations, not Explorer trust. |

CI: GitHub Actions can sign if the cert lives in a cloud HSM (Azure Key Vault, SSL.com eSigner). Secrets in GitHub for a software PFX are increasingly refused by CAs.

## Recommendation for v1

Ship unsigned portable `airlink.exe` on GitHub Releases. Document SmartScreen. Buy **EV** only if downloads are public and the warning is blocking users. Win11 compact menu uses a self-signed sparse package (Developer Mode); a purchased cert still helps SmartScreen and store-like install.
