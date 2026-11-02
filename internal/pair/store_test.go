package pair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashAndValid(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 64 {
		t.Fatalf("len %d", len(tok))
	}
	rec, err := s.Add("phone", tok)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Hash == tok || strings.Contains(string(mustRead(t, s.path)), tok) {
		t.Fatal("raw token must not be stored")
	}
	if !s.Valid(tok) {
		t.Fatal("valid")
	}
	if s.Valid(tok + "ff") {
		t.Fatal("wrong token")
	}
	if err := s.Remove(rec.ID); err != nil {
		t.Fatal(err)
	}
	if s.Valid(tok) {
		t.Fatal("removed")
	}
}

func TestRename(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "d.json"))
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := NewToken()
	rec, err := s.Add("phone", tok)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Rename(rec.ID, "Kitchen iPhone"); err != nil {
		t.Fatal(err)
	}
	if s.List()[0].Name != "Kitchen iPhone" {
		t.Fatalf("%q", s.List()[0].Name)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
