//go:build windows

package server

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// convertHEICToJPEG uses Windows Image Acquisition (WIA) when a HEIF codec is installed.
func convertHEICToJPEG(src, dst string) error {
	srcEsc := strings.ReplaceAll(src, "'", "''")
	dstEsc := strings.ReplaceAll(dst, "'", "''")
	ps := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$img = New-Object -ComObject WIA.ImageFile
$img.LoadFile('%s')
$proc = New-Object -ComObject WIA.ImageProcess
$proc.Filters.Add($proc.FilterInfos.Item('Convert').FilterID)
$proc.Filters.Item(1).Properties.Item('FormatID').Value = '{B96B3CAE-0728-11D3-9D7B-0000F81EF32E}'
try { $proc.Filters.Item(1).Properties.Item('Quality').Value = 90 } catch {}
$out = $proc.Apply($img)
if (Test-Path -LiteralPath '%s') { Remove-Item -LiteralPath '%s' -Force }
$out.SaveFile('%s')
`, srcEsc, dstEsc, dstEsc, dstEsc)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("heic convert: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	fi, err := os.Stat(dst)
	if err != nil {
		return fmt.Errorf("heic convert: no output: %w", err)
	}
	if fi.Size() == 0 {
		_ = os.Remove(dst)
		return fmt.Errorf("heic convert: empty jpeg")
	}
	return nil
}
