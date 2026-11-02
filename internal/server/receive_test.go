package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
)

func TestReceiveOnlySession(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Config{
		BindIP:  "127.0.0.1",
		Token:   tok,
		Payload: ReceivePayload(),
	}, access: newAccess(), rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	httpSrv := &http.Server{Handler: mux}
	go httpSrv.Serve(ln)
	defer httpSrv.Close()

	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String() + "/t/" + dev

	res, err := http.Get(base + "/meta")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var meta SessionMeta
	if err := json.NewDecoder(res.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if meta.Mode != "receive" {
		t.Fatalf("mode=%q", meta.Mode)
	}

	res, err = http.Get(base + "/download")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("download status %d", res.StatusCode)
	}

	res, err = http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("page %d", res.StatusCode)
	}
	if len(body) == 0 {
		t.Fatal("empty page")
	}
}

func TestRoomSessionWaiting(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Config{
		BindIP:  "127.0.0.1",
		Token:   tok,
		Payload: RoomPayload(),
	}, access: newAccess(), rate: newRateLimiter()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	httpSrv := &http.Server{Handler: mux}
	go httpSrv.Serve(ln)
	defer httpSrv.Close()

	dev, err := s.grantForTest()
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String() + "/t/" + dev
	res, err := http.Get(base + "/meta")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var meta SessionMeta
	if err := json.NewDecoder(res.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if meta.Mode != "room" {
		t.Fatalf("mode=%q", meta.Mode)
	}
	res, err = http.Get(base + "/download")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("download %d", res.StatusCode)
	}
}
