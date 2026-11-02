//go:build !windows

package ui

import (
	"fmt"
	"os"
	"os/signal"
)

// RunSessionWindow falls back to waiting for Ctrl+C outside Windows.
func RunSessionWindow(s Session, ctrl *Control) error {
	if ctrl == nil {
		ctrl = NewControl()
	}
	ctrl.Attach(func(string) {}, func() {})
	fmt.Fprintln(os.Stderr, "airlink: desktop window is Windows-only; waiting for Ctrl+C")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	return nil
}
