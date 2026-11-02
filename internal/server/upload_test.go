package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadAutoAccept(t *testing.T) {
	out := t.TempDir()
	dir := t.TempDir()
	src := filepath.Join(dir, "pic.jpg")
	if err := os.WriteFile(src, []byte("jpeg-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	pl, err := ResolvePaths([]string{src})
	if err != nil {
		t.Fatal(err)
	}
	var clip string
	s := &Server{cfg: Config{
		BindIP:     "127.0.0.1",
		Token:      "sess",
		Payload:    pl,
		UploadDir:  out,
		AutoAccept: true,
		Hooks:      Hooks{OnClipboardText: func(t string) { clip = t }},
	}, access: newAccess(), rate: newRateLimiter()}
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

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("files", "from-phone.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("phone-photo")); err != nil {
		t.Fatal(err)
	}
	_ = w.WriteField("text", "hello pc")
	w.Close()

	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	url := "http://" + ln.Addr().String() + "/t/" + dev + "/upload"
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, raw)
	}
	var got UploadResult
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "saved" || len(got.Saved) != 1 {
		t.Fatalf("%+v", got)
	}
	if clip != "hello pc" {
		t.Fatalf("clipboard text %q", clip)
	}

	entries, _ := os.ReadDir(out)
	if len(entries) != 1 {
		t.Fatalf("expected 1 saved file (text goes to clipboard), got %d", len(entries))
	}
}

func TestUploadPendingWithoutAutoAccept(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(src, []byte("x"), 0o644)
	pl, _ := ResolvePaths([]string{src})
	s := &Server{cfg: Config{BindIP: "127.0.0.1", Token: "sess", Payload: pl, UploadDir: t.TempDir()}, access: newAccess(), rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	httpSrv := &http.Server{Handler: mux}
	go httpSrv.Serve(ln)
	defer httpSrv.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("files", "up.bin")
	_, _ = part.Write([]byte("data"))
	w.Close()
	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/t/"+dev+"/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d", res.StatusCode)
	}
	var got UploadResult
	_ = json.NewDecoder(res.Body).Decode(&got)
	if got.Status != "pending" || got.ID == "" {
		t.Fatalf("%+v", got)
	}
}

func TestRouteUpload(t *testing.T) {
	if kind, _ := routeAfterToken("/t/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/upload"); kind != "upload" {
		t.Fatalf("%s", kind)
	}
	if kind, _ := routeAfterToken("/t/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/upload-status"); kind != "upload-status" {
		t.Fatalf("%s", kind)
	}
}

func TestAcceptDeclinePending(t *testing.T) {
	out := t.TempDir()
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(src, []byte("x"), 0o644)
	pl, _ := ResolvePaths([]string{src})
	s := &Server{cfg: Config{BindIP: "127.0.0.1", Token: "sess", Payload: pl, UploadDir: out}, access: newAccess(), rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	httpSrv := &http.Server{Handler: mux}
	go httpSrv.Serve(ln)
	defer httpSrv.Close()
	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}

	post := func(name string) string {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		part, _ := w.CreateFormFile("files", name)
		_, _ = part.Write([]byte("data-" + name))
		w.Close()
		req, _ := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/t/"+dev+"/upload", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var got UploadResult
		_ = json.NewDecoder(res.Body).Decode(&got)
		if got.Status != "pending" {
			t.Fatalf("%+v", got)
		}
		return got.ID
	}

	id1 := post("one.jpg")
	if _, err := s.AcceptPending(id1); err != nil {
		t.Fatal(err)
	}
	res, err := http.Get("http://" + ln.Addr().String() + "/t/" + dev + "/upload-status?id=" + id1)
	if err != nil {
		t.Fatal(err)
	}
	var st UploadResult
	_ = json.NewDecoder(res.Body).Decode(&st)
	res.Body.Close()
	if st.Status != "saved" {
		t.Fatalf("%+v", st)
	}

	id2 := post("two.jpg")
	if err := s.DeclinePending(id2); err != nil {
		t.Fatal(err)
	}
	res, err = http.Get("http://" + ln.Addr().String() + "/t/" + dev + "/upload-status?id=" + id2)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(res.Body).Decode(&st)
	res.Body.Close()
	if st.Status != "declined" {
		t.Fatalf("%+v", st)
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 1 {
		t.Fatalf("expected 1 saved file after decline, got %d", len(entries))
	}
}
