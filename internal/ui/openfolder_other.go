//go:build !windows

package ui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenFolder opens dir in the platform file manager.
func OpenFolder(dir string) error {
	if dir == "" {
		return fmt.Errorf("empty folder path")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open folder: %w", err)
	}
	return nil
}
