package server

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// jpgName replaces the extension with .jpg.
func jpgName(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(SanitizeUploadName(name), ext)
	if base == "" {
		base = "photo"
	}
	return base + ".jpg"
}

// WantsJPGConvert is true for still images that are not already JPEG.
func WantsJPGConvert(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".gif", ".bmp", ".tif", ".tiff", ".webp", ".heic", ".heif":
		return true
	default:
		return false
	}
}

// ConvertToJPEG writes a JPEG of src to dst. HEIC uses a Windows helper.
func ConvertToJPEG(src, dst string) error {
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".heic" || ext == ".heif" {
		return convertHEICToJPEG(src, dst)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	img, _, err := image.Decode(in)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 90}); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("encode jpeg: %w", err)
	}
	return out.Close()
}
