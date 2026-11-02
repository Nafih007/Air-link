package server

import (
	"path/filepath"
	"testing"
)

func TestSanitizeUploadName(t *testing.T) {
	cases := map[string]string{
		"photo.jpg":        "photo.jpg",
		"../../etc/passwd": "passwd",
		`C:\Windows\x.png`: "x.png",
		"":                 "upload.bin",
		".":                "upload.bin",
		"..":               "upload.bin",
		"  hi.png  ":       "hi.png",
	}
	for in, want := range cases {
		if got := SanitizeUploadName(in); got != want {
			t.Errorf("SanitizeUploadName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestUniquePath(t *testing.T) {
	taken := map[string]bool{}
	exists := func(p string) bool { return taken[p] }

	p1 := UniquePath("/out", "a.jpg", exists)
	if filepath.Base(p1) != "a.jpg" {
		t.Fatalf("%s", p1)
	}
	taken[p1] = true

	p2 := UniquePath("/out", "a.jpg", exists)
	if filepath.Base(p2) != "a (1).jpg" {
		t.Fatalf("%s", p2)
	}
	taken[p2] = true

	p3 := UniquePath("/out", "a.jpg", exists)
	if filepath.Base(p3) != "a (2).jpg" {
		t.Fatalf("%s", p3)
	}
}
