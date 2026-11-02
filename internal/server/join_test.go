package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClaimJoinOnce(t *testing.T) {
	s := &Server{access: newAccess()}
	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	s.SetAutoAllow(true)
	s.access.mu.Lock()
	code := s.access.join.plain
	s.access.mu.Unlock()

	tok, st, err := s.claimJoin(code, "10.0.0.2", "Safari")
	if err != nil || st != "ok" || len(tok) != 64 {
		t.Fatalf("tok=%q st=%s err=%v", tok, st, err)
	}
	_, st2, err := s.claimJoin(code, "10.0.0.3", "Chrome")
	if err != nil || st2 != "used" {
		t.Fatalf("second st=%s err=%v", st2, err)
	}
}

func TestClaimJoinExpired(t *testing.T) {
	s := &Server{access: newAccess()}
	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	s.access.mu.Lock()
	code := s.access.join.plain
	s.access.join.expires = time.Now().Add(-time.Second)
	s.access.mu.Unlock()
	_, st, err := s.claimJoin(code, "10.0.0.2", "Safari")
	if err != nil || st != "expired" {
		t.Fatalf("st=%s err=%v", st, err)
	}
}

func TestClaimJoinRaceFirstWins(t *testing.T) {
	s := &Server{access: newAccess()}
	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	s.SetAutoAllow(true)
	s.access.mu.Lock()
	code := s.access.join.plain
	s.access.mu.Unlock()

	var mu sync.Mutex
	counts := map[string]int{}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, st, err := s.claimJoin(code, "10.0.0.2", "Safari")
			if err != nil {
				t.Errorf("%v", err)
				return
			}
			mu.Lock()
			counts[st]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if counts["ok"] != 1 {
		t.Fatalf("ok=%d all=%v", counts["ok"], counts)
	}
	if counts["used"] != 7 {
		t.Fatalf("used=%d all=%v", counts["used"], counts)
	}
}

func TestJoinPendingAllowDeny(t *testing.T) {
	s := &Server{access: newAccess()}
	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	s.access.mu.Lock()
	code := s.access.join.plain
	s.access.mu.Unlock()
	tok, st, err := s.claimJoin(code, "10.0.0.2", "Safari")
	if err != nil || st != "pending" {
		t.Fatalf("st=%s err=%v", st, err)
	}
	m := s.lookupMember(tok)
	if m == nil || m.Status != devPending {
		t.Fatalf("%+v", m)
	}
	s.AllowDevice(m.ID)
	if s.lookupMember(tok).Status != devAllowed {
		t.Fatal("allow")
	}

	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	s.access.mu.Lock()
	code2 := s.access.join.plain
	s.access.mu.Unlock()
	tok2, st, err := s.claimJoin(code2, "10.0.0.3", "Chrome")
	if err != nil || st != "pending" {
		t.Fatalf("st=%s", st)
	}
	s.DenyDevice(s.lookupMember(tok2).ID)
	if s.lookupMember(tok2).Status != devDenied {
		t.Fatal("deny")
	}
	if s.lookupMember(tok).Status != devAllowed {
		t.Fatal("first device must stay allowed")
	}
}

func TestRefreshJoinKeepsDevices(t *testing.T) {
	s := &Server{access: newAccess()}
	tok, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	if s.lookupMember(tok).Status != devAllowed {
		t.Fatal("refresh must not drop device")
	}
}

func TestKickOneDeviceOnly(t *testing.T) {
	s := &Server{access: newAccess()}
	a, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	s.KickDevice(s.lookupMember(a).ID)
	if s.lookupMember(a).Status != devKicked {
		t.Fatal("kicked")
	}
	if s.lookupMember(b).Status != devAllowed {
		t.Fatal("other device")
	}
}

func TestHTTPJoinSwapAndSecondUsed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(p, []byte("hi"), 0o644)
	pl, err := ResolvePaths([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := NewToken()
	s := &Server{cfg: Config{BindIP: "127.0.0.1", Token: tok, Payload: pl}, access: newAccess(), rate: newRateLimiter()}
	if err := s.RotateJoin(); err != nil {
		t.Fatal(err)
	}
	s.SetAutoAllow(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	s.listener = ln
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()
	runJoinHTTPTest(t, s, "http://"+ln.Addr().String())
}

func runJoinHTTPTest(t *testing.T, s *Server, base string) {
	t.Helper()
	join := s.JoinURL()
	res, err := http.Post(join, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var first struct {
		Status      string `json:"status"`
		DeviceToken string `json:"deviceToken"`
		Page        string `json:"page"`
	}
	if json.Unmarshal(body, &first) != nil || first.Status != "ok" || len(first.DeviceToken) != 64 {
		t.Fatalf("%s", body)
	}
	res2, err := http.Post(join, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	if res2.StatusCode != http.StatusGone {
		t.Fatalf("second=%d %s", res2.StatusCode, body2)
	}
	if !jsonContains(body2, "already used") {
		t.Fatalf("msg=%s", body2)
	}
	meta, err := http.Get(base + first.Page + "meta")
	if err != nil {
		t.Fatal(err)
	}
	meta.Body.Close()
	if meta.StatusCode != 200 {
		t.Fatalf("meta %d", meta.StatusCode)
	}
}

func jsonContains(b []byte, sub string) bool {
	return strings.Contains(string(b), sub)
}
