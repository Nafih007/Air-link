package qr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminal(t *testing.T) {
	s, err := Terminal("http://192.168.1.10:12345/t/0123456789abcdef0123456789abcdef/")
	if err != nil {
		t.Fatal(err)
	}
	if len(s) < 10 {
		t.Fatalf("unexpected QR output: %q", s)
	}
}

func TestTerminalEmpty(t *testing.T) {
	if _, err := Terminal(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestWritePNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "q.png")
	err := WritePNG("http://192.168.1.3:9/t/0123456789abcdef0123456789abcdef/", path, 256)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < 100 {
		t.Fatalf("png missing or tiny: %v size=%v", err, fi)
	}
}

func TestPNGBytes(t *testing.T) {
	b, err := PNGBytes("http://192.168.1.3:9/t/0123456789abcdef0123456789abcdef/", 256)
	if err != nil || len(b) < 100 {
		t.Fatalf("err=%v len=%d", err, len(b))
	}
}

func TestWritePNGEmpty(t *testing.T) {
	if err := WritePNG("", filepath.Join(t.TempDir(), "x.png"), 256); err == nil {
		t.Fatal("expected error")
	}
	_ = strings.TrimSpace
}
