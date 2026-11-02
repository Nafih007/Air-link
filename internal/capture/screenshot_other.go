//go:build !windows

package capture

import "fmt"

// FullscreenPNG is Windows-only in this build.
func FullscreenPNG() (string, func(), error) {
	return "", nil, fmt.Errorf("screenshot is only supported on Windows in this build")
}
