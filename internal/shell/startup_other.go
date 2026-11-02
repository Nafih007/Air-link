//go:build !windows

package shell

import "fmt"

// InstallStartup is Windows-only.
func InstallStartup(commandExe string) error {
	return fmt.Errorf("start with Windows is only supported on Windows")
}

// UninstallStartup is Windows-only.
func UninstallStartup() error {
	return fmt.Errorf("start with Windows is only supported on Windows")
}

// StartupEnabled is Windows-only.
func StartupEnabled() (bool, error) {
	return false, nil
}
