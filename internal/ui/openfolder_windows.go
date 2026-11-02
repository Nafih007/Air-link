//go:build windows

package ui

import (
	"fmt"
	"os/exec"
)

// OpenFolder reveals dir in Explorer.
func OpenFolder(dir string) error {
	if dir == "" {
		return fmt.Errorf("empty folder path")
	}
	cmd := exec.Command("explorer.exe", dir)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open folder: %w", err)
	}
	return nil
}
