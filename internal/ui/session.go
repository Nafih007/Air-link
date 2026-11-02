package ui

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"time"
)

//go:embed session.html
var sessionHTML string

//go:embed logomark-transparent.png
var logomarkPNG []byte

// Session describes what the desktop window shows.
// Do not put the raw session token / full tokenised URL in Title or ListenAddr.
type Session struct {
	FileName   string
	FileSize   int64
	ListenAddr string // host:port only
	QRPNG      []byte
	IdleFor    time.Duration
	PIN        string // optional 4-digit; shown on PC only
	OnExpired  func() // optional; called on UI thread when countdown hits 0
}

type sessionPayload struct {
	FileName    string `json:"fileName"`
	SizeLabel   string `json:"sizeLabel"`
	ListenAddr  string `json:"listenAddr"`
	QRBase64    string `json:"qrBase64"`
	IdleSeconds int    `json:"idleSeconds"`
	PIN         string `json:"pin,omitempty"`
}

func buildSessionHTML(s Session) (string, error) {
	if len(s.QRPNG) == 0 {
		return "", fmt.Errorf("QR PNG is required")
	}
	idleSec := 0
	if s.IdleFor > 0 {
		idleSec = int(s.IdleFor.Seconds())
	}
	payload := sessionPayload{
		FileName:    s.FileName,
		SizeLabel:   FormatSize(s.FileSize),
		ListenAddr:  s.ListenAddr,
		QRBase64:    base64.StdEncoding.EncodeToString(s.QRPNG),
		IdleSeconds: idleSec,
		PIN:         s.PIN,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("session payload: %w", err)
	}
	tmpl, err := template.New("session").Parse(sessionHTML)
	if err != nil {
		return "", fmt.Errorf("parse session template: %w", err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		PayloadJSON template.JS
		Logomark    string
	}{
		PayloadJSON: template.JS(raw),
		Logomark:    base64.StdEncoding.EncodeToString(logomarkPNG),
	})
	if err != nil {
		return "", fmt.Errorf("render session: %w", err)
	}
	return buf.String(), nil
}
