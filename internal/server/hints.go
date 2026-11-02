package server

// ConnectionHints are shown when no phone hits the session within ~20s.
// Keep them free of tokens or full session URLs.
func ConnectionHints() []string {
	return []string{
		"Windows Firewall may be blocking the app — allow it on Private networks.",
		"Phone and PC must be on the same Wi-Fi (guest/client-isolation networks block device-to-device traffic).",
		"Confirm the Listening IP is your Wi-Fi address (not a VPN/WARP adapter) — try -list-ifaces.",
		"Hotspot fallback: turn on your phone's hotspot, connect this PC to it, then run Air-Link (airlink.exe) again.",
	}
}
