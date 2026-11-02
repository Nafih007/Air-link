package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFileOK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello world.txt")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	rf, err := ResolveFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if rf.Name != "hello world.txt" {
		t.Fatalf("name=%q", rf.Name)
	}
	if rf.Size != 2 {
		t.Fatalf("size=%d", rf.Size)
	}
}

func TestResolveFileZeroByte(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	rf, err := ResolveFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if rf.Size != 0 {
		t.Fatalf("size=%d", rf.Size)
	}
}

func TestResolveFileRejectsDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolveFile(dir); err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestResolveFileRejectsMissing(t *testing.T) {
	if _, err := ResolveFile(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveFileRejectsEmpty(t *testing.T) {
	if _, err := ResolveFile("  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveFileUnicodeName(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "照片😀.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rf, err := ResolveFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if rf.Name != "照片😀.txt" {
		t.Fatalf("name=%q", rf.Name)
	}
}
