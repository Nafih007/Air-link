//go:build windows

package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ReadText returns Unicode clipboard text via PowerShell (no CGO/unsafe).
func ReadText() (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; Get-Clipboard -Raw")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read clipboard text: %w", err)
	}
	s := strings.TrimRight(string(out), "\r\n")
	if s == "" {
		return "", fmt.Errorf("clipboard has no text")
	}
	return s, nil
}

// HasImage reports whether the clipboard appears to hold an image.
func HasImage() bool {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Add-Type -AssemblyName System.Windows.Forms; if ([System.Windows.Forms.Clipboard]::ContainsImage()) { '1' } else { '0' }")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "1"
}

// ReadImagePNG returns a PNG of the clipboard image via PowerShell/.NET.
func ReadImagePNG() ([]byte, error) {
	tmp, err := os.CreateTemp("", "sendtophone-clip-*.png")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)

	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$img = [System.Windows.Forms.Clipboard]::GetImage()
if ($null -eq $img) { Write-Error 'no image'; exit 1 }
$img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
`, strings.ReplaceAll(path, "'", "''"))

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("read clipboard image: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("clipboard image was empty")
	}
	return data, nil
}

// WriteText puts Unicode text on the clipboard via PowerShell (no CGO/unsafe).
func WriteText(text string) error {
	tmp, err := os.CreateTemp("", "sendtophone-clip-*.txt")
	if err != nil {
		return err
	}
	path := tmp.Name()
	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return err
	}
	_ = tmp.Close()
	defer os.Remove(path)

	escaped := strings.ReplaceAll(path, "'", "''")
	ps := fmt.Sprintf(`$t = Get-Content -Raw -Encoding UTF8 '%s'; Set-Clipboard -Value $t`, escaped)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write clipboard text: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
