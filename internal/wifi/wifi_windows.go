//go:build windows

package wifi

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Credentials holds Wi-Fi SSID + password for a WIFI: QR payload.
type Credentials struct {
	SSID     string
	Password string
	Auth     string // WPA, WEP, or nopass
}

// CurrentProfile reads the connected Wi-Fi profile name (SSID).
func CurrentProfile() (string, error) {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("netsh interfaces: %w", err)
	}
	re := regexp.MustCompile(`(?m)^\s*SSID\s*:\s*(.+)\s*$`)
	m := re.FindSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("no Wi-Fi SSID found (are you connected?)")
	}
	ssid := strings.TrimSpace(string(m[1]))
	if ssid == "" || strings.EqualFold(ssid, "SSID") {
		return "", fmt.Errorf("empty SSID")
	}
	return ssid, nil
}

// ReadPassword reads the key for a WLAN profile. Caller must have user consent.
func ReadPassword(profile string) (string, error) {
	out, err := exec.Command("netsh", "wlan", "show", "profile", "name="+profile, "key=clear").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("netsh profile: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	re := regexp.MustCompile(`(?m)^\s*Key Content\s*:\s*(.*)\s*$`)
	m := re.FindSubmatch(out)
	if m == nil {
		return "", nil // open network
	}
	return strings.TrimSpace(string(m[1])), nil
}

// QRPayload builds a WIFI: QR string (never logged by callers).
func QRPayload(c Credentials) string {
	auth := c.Auth
	if auth == "" {
		if c.Password == "" {
			auth = "nopass"
		} else {
			auth = "WPA"
		}
	}
	return fmt.Sprintf("WIFI:T:%s;S:%s;P:%s;;",
		escapeWIFI(auth), escapeWIFI(c.SSID), escapeWIFI(c.Password))
}

func escapeWIFI(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, `"`, `\"`, `:`, `\:`)
	return r.Replace(s)
}
