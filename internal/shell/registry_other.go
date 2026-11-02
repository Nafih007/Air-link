//go:build !windows

package shell

import "fmt"

// InstallClassicVerb is Windows-only.
func InstallClassicVerb(string) error {
	return fmt.Errorf("classic context-menu install is only supported on Windows")
}

// UninstallClassicVerb is Windows-only.
func UninstallClassicVerb() error {
	return fmt.Errorf("classic context-menu uninstall is only supported on Windows")
}

// InstallContextMenu is Windows-only.
func InstallContextMenu(string) error {
	return fmt.Errorf("context-menu install is only supported on Windows")
}

// UninstallContextMenu is Windows-only.
func UninstallContextMenu() error {
	return fmt.Errorf("context-menu uninstall is only supported on Windows")
}

// EnsureContextMenu is a no-op off Windows.
func EnsureContextMenu(string) error { return nil }

// ContextMenuStatus is empty off Windows.
func ContextMenuStatus() MenuStatus {
	st := MenuStatus{Win11Note: win11SettingsNote(), Detail: "Windows only"}
	st.Label = menuLabel(st)
	return st
}
