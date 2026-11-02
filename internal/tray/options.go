package tray

import "io"

// Options control the tray process (M4.9).
type Options struct {
	Hotkey bool
	Combo  string // e.g. Ctrl+Shift+P; empty = DefaultHotkey
	Stdout io.Writer
	Stderr io.Writer
}
