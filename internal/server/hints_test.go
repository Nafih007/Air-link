package server

import (
	"strings"
	"testing"
)

func TestConnectionHints(t *testing.T) {
	h := ConnectionHints()
	if len(h) < 3 {
		t.Fatalf("too few hints: %d", len(h))
	}
	joined := strings.Join(h, "\n")
	for _, want := range []string{"Firewall", "Wi-Fi", "hotspot"} {
		if !strings.Contains(strings.ToLower(joined), strings.ToLower(want)) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "/t/") {
		t.Fatal("hints must not contain token paths")
	}
}
