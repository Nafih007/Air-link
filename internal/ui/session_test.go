package ui

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSessionHTML(t *testing.T) {
	html, err := buildSessionHTML(Session{
		FileName:   "hello world.txt",
		FileSize:   2048,
		ListenAddr: "192.168.1.3:54321",
		QRPNG:      []byte{0x89, 0x50, 0x4e, 0x47}, // minimal stand-in
		IdleFor:    2 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"hello world.txt",
		"192.168.1.3:54321",
		"2.0 KB",
		"data:image/png;base64,",
		`"idleSeconds":120`,
		"Air-Link",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in html", want)
		}
	}
	// Must not embed a tokenised path accidentally in listen line.
	if strings.Contains(html, "/t/") {
		t.Fatal("HTML should not contain /t/ token path")
	}
}

func TestBuildSessionHTMLIdleOff(t *testing.T) {
	html, err := buildSessionHTML(Session{
		FileName:   "x",
		ListenAddr: "192.168.1.3:1",
		QRPNG:      []byte{1},
		IdleFor:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `"idleSeconds":0`) {
		t.Fatalf("%s", html)
	}
}

func TestBuildSessionHTMLRequiresQR(t *testing.T) {
	if _, err := buildSessionHTML(Session{FileName: "x"}); err == nil {
		t.Fatal("expected error")
	}
}
