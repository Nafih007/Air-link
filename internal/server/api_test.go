package server

import (
	"bytes"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"airlink/internal/pair"
)

func TestAPIUploadRequiresPair(t *testing.T) {
	out := t.TempDir()
	st, err := pair.Open(filepath.Join(t.TempDir(), "d.json"))
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := pair.NewToken()
	if _, err := st.Add("p", tok); err != nil {
		t.Fatal(err)
	}
	sess, _ := NewToken()
	s := &Server{cfg: Config{
		BindIP: "127.0.0.1", Token: sess, Payload: ReceivePayload(),
		Pairs: st, UploadDir: out, AutoAccept: true,
	}, rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", s.handleAPIUpload)
	mux.HandleFunc("/", s.handle)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	hs := &http.Server{Handler: mux}
	go hs.Serve(ln)
	defer hs.Close()
	base := "http://" + ln.Addr().String()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("files", "shot.jpg")
	_, _ = part.Write([]byte("img"))
	w.Close()

	req, _ := http.NewRequest(http.MethodPost, base+"/api/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no bearer %d", res.StatusCode)
	}

	body.Reset()
	w = multipart.NewWriter(&body)
	part, _ = w.CreateFormFile("files", "shot.jpg")
	_, _ = part.Write([]byte("img"))
	w.Close()
	req, _ = http.NewRequest(http.MethodPost, base+"/api/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("paired %d", res.StatusCode)
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 1 {
		t.Fatalf("saved %d", len(entries))
	}
}

func TestAPIUploadText(t *testing.T) {
	var clip string
	st, _ := pair.Open(filepath.Join(t.TempDir(), "d.json"))
	tok, _ := pair.NewToken()
	_, _ = st.Add("p", tok)
	sess, _ := NewToken()
	s := &Server{cfg: Config{
		BindIP: "127.0.0.1", Token: sess, Payload: ReceivePayload(),
		Pairs: st, AutoAccept: true,
		Hooks: Hooks{OnClipboardText: func(t string) { clip = t }},
	}, rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", s.handleAPIUpload)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	hs := &http.Server{Handler: mux}
	go hs.Serve(ln)
	defer hs.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("text", "hello from phone")
	w.Close()
	req, _ := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/api/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%d", res.StatusCode)
	}
	if clip != "hello from phone" {
		t.Fatalf("clip %q", clip)
	}
}

func TestAPIUploadTooLarge(t *testing.T) {
	st, _ := pair.Open(filepath.Join(t.TempDir(), "d.json"))
	tok, _ := pair.NewToken()
	_, _ = st.Add("p", tok)
	sess, _ := NewToken()
	s := &Server{cfg: Config{
		BindIP: "127.0.0.1", Token: sess, Payload: ReceivePayload(),
		Pairs: st, UploadDir: t.TempDir(), AutoAccept: true, MaxUpload: 4,
	}, rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", s.handleAPIUpload)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	hs := &http.Server{Handler: mux}
	go hs.Serve(ln)
	defer hs.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("files", "big.bin")
	_, _ = part.Write([]byte("12345"))
	w.Close()
	req, _ := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/api/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge && res.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 413/400 got %d", res.StatusCode)
	}
}

func TestHomePage(t *testing.T) {
	st, _ := pair.Open(filepath.Join(t.TempDir(), "d.json"))
	sess, _ := NewToken()
	s := &Server{cfg: Config{Token: sess, Payload: ReceivePayload(), Pairs: st}}
	mux := http.NewServeMux()
	mux.HandleFunc("/home", s.serveHome)
	mux.HandleFunc("/manifest.json", s.serveManifest)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	hs := &http.Server{Handler: mux}
	go hs.Serve(ln)
	defer hs.Close()
	base := "http://" + ln.Addr().String()
	res, err := http.Get(base + "/home")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("home %d", res.StatusCode)
	}
	res, err = http.Get(base + "/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("manifest %d", res.StatusCode)
	}
	s2 := &Server{cfg: Config{Token: sess, Payload: ReceivePayload()}}
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/home", s2.serveHome)
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln2.Close()
	hs2 := &http.Server{Handler: mux2}
	go hs2.Serve(ln2)
	defer hs2.Close()
	res, err = http.Get("http://" + ln2.Addr().String() + "/home")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("no pairs home %d", res.StatusCode)
	}
}
