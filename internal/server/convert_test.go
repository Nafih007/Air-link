package server

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestJpgName(t *testing.T) {
	if got := jpgName("IMG_1234.HEIC"); got != "IMG_1234.jpg" {
		t.Fatalf("%s", got)
	}
}

func TestWantsJPGConvert(t *testing.T) {
	if !WantsJPGConvert("a.heic") || WantsJPGConvert("a.jpg") || WantsJPGConvert("a.mp4") {
		t.Fatal("wants")
	}
}

func TestConvertPNGToJPEG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "dot.png")
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	dst := filepath.Join(dir, "dot.jpg")
	if err := ConvertToJPEG(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff}) {
		t.Fatalf("not a jpeg prefix")
	}
}

func TestCommitKeepsOriginalWithoutConvert(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(t.TempDir(), "x.heic")
	payload := []byte("not-really-heic-bytes")
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Config{UploadDir: dir, ConvertToJPG: false}}
	saved, err := s.commitBatch(&pendingBatch{TempFiles: []pendingFile{
		{TempPath: tmp, Name: "x.heic", Size: int64(len(payload))},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0] != "x.heic" {
		t.Fatalf("%v", saved)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "x.heic"))
	if !bytes.Equal(got, payload) {
		t.Fatal("bytes changed")
	}
}

func TestCommitConvertPNG(t *testing.T) {
	out := t.TempDir()
	src := filepath.Join(t.TempDir(), "dot.png")
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	f, _ := os.Create(src)
	_ = png.Encode(f, img)
	f.Close()
	s := &Server{cfg: Config{UploadDir: out, ConvertToJPG: true}}
	saved, err := s.commitBatch(&pendingBatch{TempFiles: []pendingFile{
		{TempPath: src, Name: "dot.png", Size: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || saved[0] != "dot.jpg" {
		t.Fatalf("%v", saved)
	}
	if _, err := os.Stat(filepath.Join(out, "dot.png")); err == nil {
		t.Fatal("png should not remain")
	}
}
