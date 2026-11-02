package content

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

// MaxDirectQRChars is the PRD limit for encoding content directly in a QR
// (no local HTTP server).
const MaxDirectQRChars = 300

// FitsDirectQR reports whether s can go straight into a QR code.
func FitsDirectQR(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return utf8.RuneCountInString(s) <= MaxDirectQRChars
}

// LooksLikeURL reports whether s is an absolute http(s) URL.
func LooksLikeURL(s string) bool {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

// NormalizeClipboardText trims and rejects empty.
func NormalizeClipboardText(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errEmpty
	}
	return s, nil
}

type emptyErr struct{}

func (emptyErr) Error() string { return "clipboard text is empty" }

var errEmpty = emptyErr{}
