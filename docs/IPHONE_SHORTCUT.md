# Send to PC from the phone (Feature 4)

Pair **once** (open the receive QR → **Pair this phone**). The page shows your URL and device token. The PC stores only a hash of the token.

Receive mode must stay on (`airlink.exe -receive`). It listens on port **17612** by default (no idle timeout unless you pass `-idle`). Bind is the LAN IP only. Close the window or turn Receive off in the tray to stop.

The PC answers mDNS for `<hostname>.local` on the LAN. If DHCP changes the IP, `.local` should still resolve; an old IP in a Shortcut will fail until you edit it or use `.local`. If `.local` fails, use the IP shown on the pairing page.

**Add to Home Screen:** after pairing, open `http://PC_IP:17612/home` (also printed on the PC and on the pairing page) → Share → Add to Home Screen. That page stores the device token and can send files without a new QR.

## iPhone Shortcut (share sheet)

1. Open **Shortcuts** → **+** → name it **Send to PC**.
2. Add **Receive** → enable **Images**, **Files**, **Text** in the share sheet.
3. Add **Get Contents of URL**:
   - URL: `http://PC_IP:PORT/api/upload` (from the pairing page)
   - Method: **POST**
   - Headers: `Authorization` = `Bearer DEVICE_TOKEN` (from the pairing page)
   - Request Body: **Form**
     - `files` = **Shortcut Input** (when it’s a file/photo)
     - `text` = **Shortcut Input** (when it’s text)
4. Add to the share sheet. Share a photo → **Send to PC**.

Do not paste the token into chat logs. If you leak it, **Remove** the device on the PC and pair again.

**Add to Home Screen:** while the pairing page is open, Share → Add to Home Screen. The page remembers the token in the browser (same origin). Re-open the icon to send files from the on-page picker without scanning.

## Android — HTTP Shortcuts

Install [HTTP Shortcuts](https://http-shortcuts.rmy.ch/). New shortcut:

- Method **POST**, URL `http://PC_IP:PORT/api/upload`
- Header `Authorization: Bearer DEVICE_TOKEN`
- Body: multipart form field `files` (file picker) and/or `text`

## curl check

```powershell
curl.exe -H "Authorization: Bearer DEVICE_TOKEN" -F "files=@C:\path\to\photo.jpg" http://PC_IP:PORT/api/upload
```

Wrong/missing Bearer → 401. Unpaired tokens never upload.
