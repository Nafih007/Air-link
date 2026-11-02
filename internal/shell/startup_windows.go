//go:build windows

package shell

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	startupKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupValue = "AirLink"
)

// InstallStartup registers HKCU Run so the tray starts at logon.
func InstallStartup(commandExe string) error {
	exe, err := filepath.Abs(commandExe)
	if err != nil {
		return fmt.Errorf("startup: %w", err)
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, startupKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("startup key: %w", err)
	}
	defer k.Close()
	cmd := fmt.Sprintf("\"%s\" -tray", exe)
	if err := k.SetStringValue(startupValue, cmd); err != nil {
		return fmt.Errorf("startup value: %w", err)
	}
	_ = k.DeleteValue("SendToPhone")
	return nil
}

// UninstallStartup removes the HKCU Run entry.
func UninstallStartup() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, startupKey, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer k.Close()
	_ = k.DeleteValue("SendToPhone")
	err = k.DeleteValue(startupValue)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

// StartupEnabled reports whether the Run value is present.
func StartupEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, startupKey, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer k.Close()
	_, _, err = k.GetStringValue(startupValue)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
