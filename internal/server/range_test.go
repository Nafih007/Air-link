package server

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestSingleFileRangeRequest(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	payload := []byte("0123456789abcdefghij")
	if err := os.WriteFile(p, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := ResolvePaths([]string{p})
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

	url := "http://" + ln.Addr().String() + "/t/" + dev + "/download"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-4")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusPartialContent {
		t.Fatalf("status=%d want 206", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "01234" {
		t.Fatalf("body=%q", body)
	}
	if got := res.Header.Get("Content-Range"); got == "" {
		t.Fatal("missing Content-Range")
	}

	// Resume from middle.
	req2, _ := http.NewRequest(http.MethodGet, url, nil)
	req2.Header.Set("Range", "bytes=10-")
	res2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusPartialContent {
		t.Fatalf("status=%d", res2.StatusCode)
	}
	body2, _ := io.ReadAll(res2.Body)
	if string(body2) != "abcdefghij" {
		t.Fatalf("body=%q", body2)
	}
}

func TestZipRejectsRange(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(a, []byte("a"), 0o644)
	_ = os.WriteFile(b, []byte("b"), 0o644)
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

	req, _ := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/t/"+dev+"/download", nil)
	req.Header.Set("Range", "bytes=0-1")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("status=%d want 416", res.StatusCode)
	}
}
