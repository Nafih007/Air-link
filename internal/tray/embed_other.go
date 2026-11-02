//go:build !windows

package tray

// Embedded is unused off Windows.
type Embedded struct {
	OnShow       func()
	OnOpenFolder func()
	OnQuit       func()
}

// StartEmbedded is a no-op off Windows.
func StartEmbedded(Embedded) {}

// StopEmbedded is a no-op off Windows.
func StopEmbedded() {}
