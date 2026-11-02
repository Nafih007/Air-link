package qr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skip2/go-qrcode"
)

// PNGBytes returns a PNG-encoded QR of the given pixel size.
func PNGBytes(content string, sizePx int) ([]byte, error) {
	if content == "" {
		return nil, fmt.Errorf("empty QR content")
	}
	if sizePx < 128 {
		sizePx = 256
	}
	png, err := qrcode.Encode(content, qrcode.Medium, sizePx)
	if err != nil {
		return nil, fmt.Errorf("encode QR PNG: %w", err)
	}
	return png, nil
}

// Terminal returns a scannable ASCII QR for content (typically the session URL).
func Terminal(content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("empty QR content")
	}
	code, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("encode QR: %w", err)
	}
	return code.ToSmallString(false), nil
}

// WritePNG writes a high-contrast QR PNG to path (good for phone cameras).
func WritePNG(content, path string, sizePx int) error {
	if content == "" {
		return fmt.Errorf("empty QR content")
	}
	if sizePx < 128 {
		sizePx = 256
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("qr dir: %w", err)
	}
	if err := qrcode.WriteFile(content, qrcode.Medium, sizePx, path); err != nil {
		return fmt.Errorf("write QR PNG: %w", err)
	}
	return nil
}
