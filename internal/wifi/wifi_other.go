//go:build !windows

package wifi

import "fmt"

// Credentials is unused outside Windows.
type Credentials struct {
	SSID, Password, Auth string
}

// CurrentProfile is Windows-only.
func CurrentProfile() (string, error) {
	return "", fmt.Errorf("Wi-Fi sharing is only supported on Windows in this build")
}

// ReadPassword is Windows-only.
func ReadPassword(profile string) (string, error) {
	return "", fmt.Errorf("Wi-Fi sharing is only supported on Windows in this build")
}

// QRPayload builds a WIFI: QR string.
func QRPayload(c Credentials) string {
	return fmt.Sprintf("WIFI:T:WPA;S:%s;P:%s;;", c.SSID, c.Password)
}
