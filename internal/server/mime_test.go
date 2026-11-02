package server

import "testing"

func TestMIMEByFilename(t *testing.T) {
	cases := map[string]string{
		"a.JPG":     "image/jpeg",
		"x.heic":    "image/heic",
		"clip.mp4":  "video/mp4",
		"clip.MOV":  "video/quicktime",
		"notes.txt": "application/octet-stream",
	}
	for name, want := range cases {
		if got := MIMEByFilename(name); got != want {
			t.Errorf("%s: got %q want %q", name, got, want)
		}
	}
}

func TestFileKind(t *testing.T) {
	if FileKind("a.png") != "image" || FileKind("a.mp4") != "video" || FileKind("a.bin") != "other" {
		t.Fatal("kind mismatch")
	}
}
