//go:build !windows

package shell

import "fmt"

// InstallWin11Menu is Windows-only.
func InstallWin11Menu(commandExe string) error {
	return fmt.Errorf("Win11 menu install is only supported on Windows")
}

// UninstallWin11Menu is Windows-only.
func UninstallWin11Menu() error {
	return fmt.Errorf("Win11 menu uninstall is only supported on Windows")
}
