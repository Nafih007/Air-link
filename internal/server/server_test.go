package server

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateBindIP(t *testing.T) {
	if err := validateBindIP("0.0.0.0"); err == nil {
		t.Fatal("0.0.0.0 must be rejected")
	}
	if err := validateBindIP("::"); err == nil {
		t.Fatal(":: must be rejected")
	}
	if err := validateBindIP("127.0.0.1"); err == nil {
		t.Fatal("loopback must be rejected")
	}
	if err := validateBindIP("192.168.1.10"); err != nil {
		t.Fatal(err)
	}
}

func TestServePageAndDownload(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello world.txt")
	if err := os.WriteFile(p, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := ResolvePaths([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Config{BindIP: "127.0.0.1", Payload: pl, Token: "sess"}, access: newAccess(), rate: newRateLimiter()}
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

	base := "http://" + ln.Addr().String()
	client := &http.Client{}
	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	gone, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("no token: %d", res.StatusCode)
	}
	goneHTML := string(gone)
	if !strings.Contains(goneHTML, "This link has expired or is invalid") {
		t.Fatalf("404 html=%q", goneHTML)
	}
	if strings.Contains(goneHTML, "secret") || strings.Contains(strings.ToLower(goneHTML), "/t/") {
		t.Fatal("404 must not echo secrets or token paths")
	}

	res, err = client.Get(base + "/t/" + dev + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("page: %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%q", ct)
	}
	body := string(page)
	if !strings.Contains(body, "app.js") || !strings.Contains(body, "Air-Link") {
		t.Fatalf("page body=%q", body)
	}

	res, err = client.Get(base + "/t/" + dev + "/meta")
	if err != nil {
		t.Fatal(err)
	}
	metaBody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("meta: %d", res.StatusCode)
	}
	if !strings.Contains(string(metaBody), "hello world.txt") {
		t.Fatalf("meta=%q", metaBody)
	}

	res, err = client.Get(base + "/t/" + dev + "/download")
	if err != nil {
		t.Fatal(err)
	}
	fileBody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("download: %d", res.StatusCode)
	}
	if string(fileBody) != "secret" {
		t.Fatalf("body=%q", fileBody)
	}
	if res.Header.Get("Content-Disposition") == "" {
		t.Fatal("missing Content-Disposition")
	}
}

func TestContentDispositionUnicode(t *testing.T) {
	cd := contentDisposition("my file #1 照片.txt")
	if !strings.Contains(cd, "filename=") || !strings.Contains(cd, "filename*=UTF-8''") {
		t.Fatalf("cd=%q", cd)
	}
}

func TestRouteAfterToken(t *testing.T) {
	tok := "0123456789abcdef0123456789abcdef"
	if kind, _ := routeAfterToken("/t/" + tok + "/"); kind != "page" {
		t.Fatal("page")
	}
	if kind, _ := routeAfterToken("/t/" + tok + "/download"); kind != "download" {
		t.Fatal("download")
	}
	if kind, _ := routeAfterToken("/t/" + tok + "/meta"); kind != "meta" {
		t.Fatal("meta")
	}
	if kind, idx := routeAfterToken("/t/" + tok + "/file/2"); kind != "file" || idx != 2 {
		t.Fatalf("file got %s %d", kind, idx)
	}
	if kind, _ := routeAfterToken("/t/" + tok + "/other"); kind != "unknown" {
		t.Fatal("unknown")
	}
}
