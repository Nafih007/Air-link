//go:build !windows

package notify

// Toast is a no-op off Windows.
func Toast(title, body string) {}
