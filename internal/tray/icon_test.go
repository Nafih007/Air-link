package tray

import "testing"

func TestBlueDiscICOHeader(t *testing.T) {
	b := appICO()
	if len(b) < 22 {
		t.Fatalf("ico too small: %d", len(b))
	}
	if b[0] != 0 || b[1] != 0 || b[2] != 1 || b[3] != 0 {
		t.Fatalf("not an ICO header: %v", b[:8])
	}
}
