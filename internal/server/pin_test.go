package server

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func pinTestServer(t *testing.T, pin string) (*Server, string, string) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(src, []byte("x"), 0o644)
	pl, err := ResolvePaths([]string{src})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Config{BindIP: "127.0.0.1", Token: "sess", Payload: pl, PIN: pin}, rate: newRateLimiter(), access: newAccess()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	httpSrv := &http.Server{Handler: mux}
	go httpSrv.Serve(ln)
	t.Cleanup(func() { httpSrv.Close() })
	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	return s, dev, "http://" + ln.Addr().String() + "/t/" + dev
}

func TestPINBlocksThenUnlocks(t *testing.T) {
	_, _, base := pinTestServer(t, "4242")
	res, err := http.Get(base + "/meta")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("meta without pin: %d", res.StatusCode)
	}

	body, _ := json.Marshal(map[string]string{"pin": "4242"})
	res, err = http.Post(base+"/pin", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("pin %d", res.StatusCode)
	}
	cookies := res.Cookies()
	res.Body.Close()

	req, _ := http.NewRequest(http.MethodGet, base+"/meta", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("meta with pin cookie %d", res.StatusCode)
	}
}

func TestPINLockout(t *testing.T) {
	_, _, base := pinTestServer(t, "9999")
	wrong, _ := json.Marshal(map[string]string{"pin": "0000"})
	for i := 0; i < 5; i++ {
		res, err := http.Post(base+"/pin", "application/json", bytes.NewReader(wrong))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}
	res, err := http.Post(base+"/pin", "application/json", bytes.NewReader(wrong))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("locked want 403 got %d", res.StatusCode)
	}
}

func TestKickDevice(t *testing.T) {
	s, _, base := pinTestServer(t, "")
	req, _ := http.NewRequest(http.MethodGet, base+"/", nil)
	req.Header.Set("User-Agent", "KickTest/1.0")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	list := s.DeviceList()
	if len(list) != 1 {
		t.Fatalf("devices %d", len(list))
	}
	s.KickDevice(list[0].ID)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("kicked %d", res.StatusCode)
	}
}

func TestRateLimitBadToken(t *testing.T) {
	_, tok, base := pinTestServer(t, "")
	bad := base[:len(base)-len(tok)] + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var last int
	for i := 0; i < 25; i++ {
		res, err := http.Get(bad + "/")
		if err != nil {
			t.Fatal(err)
		}
		last = res.StatusCode
		res.Body.Close()
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after burst, got %d", last)
	}
}

func TestWrongPINCountsTowardRateLimit(t *testing.T) {
	_, _, base := pinTestServer(t, "4242")
	wrong, _ := json.Marshal(map[string]string{"pin": "0000"})
	for i := 0; i < 5; i++ {
		res, err := http.Post(base+"/pin", "application/json", bytes.NewReader(wrong))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}
	bad := base[:len(base)-32] + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var last int
	for i := 0; i < 16; i++ {
		res, err := http.Get(bad + "/")
		if err != nil {
			t.Fatal(err)
		}
		last = res.StatusCode
		res.Body.Close()
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after PIN+token burst, got %d", last)
	}
}

func TestNewPINFormat(t *testing.T) {
	p, err := NewPIN()
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 4 {
		t.Fatalf("%q", p)
	}
	for _, c := range p {
		if c < '0' || c > '9' {
			t.Fatalf("%q", p)
		}
	}
}
