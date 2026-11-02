//go:build windows

package capture

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FullscreenPNG captures the virtual screen to a PNG file path.
func FullscreenPNG() (path string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "sendtophone-shot-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	path = filepath.Join(dir, "screenshot.png")
	escaped := strings.ReplaceAll(path, "'", "''")
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$bounds = [System.Windows.Forms.SystemInformation]::VirtualScreen
$bmp = New-Object System.Drawing.Bitmap $bounds.Width, $bounds.Height
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
$bmp.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose(); $bmp.Dispose()
`, escaped)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("screenshot: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return path, cleanup, nil
}
