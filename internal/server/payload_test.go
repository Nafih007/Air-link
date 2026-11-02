package server

import (
	"archive/zip"
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoomPayload(t *testing.T) {
	p := RoomPayload()
	if !p.Room || p.HasDownload() || p.ReceiveOnly {
		t.Fatalf("%+v", p)
	}
	if p.DisplayName == "" {
		t.Fatal("display name")
	}
}

func TestResolvePathsSingleFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(p, []byte("hi"), 0o644)
	pl, err := ResolvePaths([]string{p})
	if err != nil || pl.Single == nil || pl.DownloadName != "a.txt" {
		t.Fatalf("pl=%+v err=%v", pl, err)
	}
}

func TestResolvePathsFolderZip(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "photos")
	_ = os.MkdirAll(root, 0o755)
	_ = os.WriteFile(filepath.Join(root, "one.txt"), []byte("1"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "two.txt"), []byte("22"), 0o644)

	pl, err := ResolvePaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if pl.Single != nil || len(pl.ZipEntries) != 2 {
		t.Fatalf("entries=%d single=%v", len(pl.ZipEntries), pl.Single != nil)
	}
	if pl.DownloadName != "photos.zip" {
		t.Fatalf("name=%q", pl.DownloadName)
	}
	for _, e := range pl.ZipEntries {
		if !strings.HasPrefix(e.NameInZip, "photos/") {
			t.Fatalf("zip name=%q", e.NameInZip)
		}
		if strings.Contains(e.NameInZip, "..") {
			t.Fatal("parent in zip name")
		}
	}
}

func TestResolvePathsMultiFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(a, []byte("a"), 0o644)
	_ = os.WriteFile(b, []byte("b"), 0o644)
	pl, err := ResolvePaths([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if pl.DownloadName != "airlink.zip" || len(pl.ZipEntries) != 2 {
		t.Fatalf("%+v", pl)
	}
}

func TestUniqueZipName(t *testing.T) {
	used := map[string]int{}
	n1 := uniqueZipName("a.txt", used)
	n2 := uniqueZipName("a.txt", used)
	if n1 != "a.txt" || n2 != "a (1).txt" {
		t.Fatalf("%q %q", n1, n2)
	}
}

func TestSafeZipRel(t *testing.T) {
	if safeZipRel("..") || safeZipRel("../x") || safeZipRel("") {
		t.Fatal("expected false")
	}
	if !safeZipRel("ok/file.txt") {
		t.Fatal("expected true")
	}
}

func TestServeZipDownload(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(a, []byte("aaa"), 0o644)
	_ = os.WriteFile(b, []byte("bbb"), 0o644)
	pl, err := ResolvePaths([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Config{BindIP: "127.0.0.1", Token: "sess", Payload: pl}, access: newAccess(), rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Get("http://" + ln.Addr().String() + "/t/" + dev + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status=%d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("zip files=%d", len(zr.File))
	}

	res2, err := http.Get("http://" + ln.Addr().String() + "/t/" + dev + "/download?files=1")
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	body2, _ := io.ReadAll(res2.Body)
	zr2, err := zip.NewReader(bytes.NewReader(body2), int64(len(body2)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr2.File) != 1 {
		t.Fatalf("filtered zip files=%d", len(zr2.File))
	}
}
