//go:build windows

package shell

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed win11/ExplorerCommand.cs win11/install.ps1
var win11Files embed.FS

// InstallWin11Menu compiles the IExplorerCommand DLL, registers COM + sparse package.
func InstallWin11Menu(commandExe string) error {
	exe, err := filepath.Abs(commandExe)
	if err != nil {
		return err
	}
	script, err := extractWin11Scripts()
	if err != nil {
		return err
	}
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", script, "-Exe", exe)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		fmt.Print(string(out))
	}
	if err != nil {
		return fmt.Errorf("win11 menu: %w", err)
	}
	return nil
}

// UninstallWin11Menu removes the sparse package and COM class.
func UninstallWin11Menu() error {
	script, err := extractWin11Scripts()
	if err != nil {
		return err
	}
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", script, "-Uninstall")
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		fmt.Print(string(out))
	}
	if err != nil {
		return fmt.Errorf("win11 menu uninstall: %w", err)
	}
	return nil
}

func extractWin11Scripts() (scriptPath string, err error) {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Air-Link", "win11menu-src")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, name := range []string{"ExplorerCommand.cs", "install.ps1"} {
		data, err := win11Files.ReadFile("win11/" + name)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return "", err
		}
	}
	return filepath.Join(dir, "install.ps1"), nil
}
