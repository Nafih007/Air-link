//go:build !windows

package clipboard

import "fmt"

// ReadText is Windows-only in this build.
func ReadText() (string, error) {
	return "", fmt.Errorf("clipboard text is only supported on Windows in this build")
}

// HasImage is Windows-only in this build.
func HasImage() bool { return false }

// ReadImagePNG is Windows-only in this build.
func ReadImagePNG() ([]byte, error) {
	return nil, fmt.Errorf("clipboard image is only supported on Windows in this build")
}

// WriteText is Windows-only in this build.
func WriteText(text string) error {
	return fmt.Errorf("clipboard write is only supported on Windows in this build")
}
