//go:build !windows

package tray

import (
	"fmt"
)

// Run is Windows-only.
func Run(opt Options) error {
	return fmt.Errorf("tray is only supported on Windows")
}
