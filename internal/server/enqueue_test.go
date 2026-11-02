package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnqueuePathsOntoRoom(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(a, []byte("aaa"), 0o644)
	_ = os.WriteFile(b, []byte("bb"), 0o644)

	s := &Server{cfg: Config{Payload: RoomPayload()}}
	added, err := s.EnqueuePaths([]string{a})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0].Name != "a.txt" {
		t.Fatalf("%+v", added)
	}
	p := s.payload()
	if !p.Room || p.Single == nil || p.Single.Name != "a.txt" {
		t.Fatalf("%+v", p)
	}

	added2, err := s.EnqueuePaths([]string{b})
	if err != nil {
		t.Fatal(err)
	}
	if len(added2) != 1 || added2[0].Name != "b.txt" {
		t.Fatalf("%+v", added2)
	}
	p = s.payload()
	if p.Single != nil || len(p.ZipEntries) != 2 {
		t.Fatalf("%+v", p)
	}
	meta := s.buildMeta()
	if !meta.Room || meta.Mode != "file" || len(meta.Files) != 2 {
		t.Fatalf("%+v", meta)
	}
}
