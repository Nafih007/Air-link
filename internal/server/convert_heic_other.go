//go:build !windows

package server

import "fmt"

func convertHEICToJPEG(src, dst string) error {
	return fmt.Errorf("HEIC to JPEG is only supported on Windows in this build")
}
