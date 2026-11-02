package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestChunkedUploadResumeAndSave(t *testing.T) {
	out := t.TempDir()
	dir := t.TempDir()
	src := filepath.Join(dir, "seed.txt")
	_ = os.WriteFile(src, []byte("seed"), 0o644)
	pl, _ := ResolvePaths([]string{src})
	s := &Server{cfg: Config{
		BindIP:     "127.0.0.1",
		Token:      "sess",
		Payload:    pl,
		UploadDir:  out,
		AutoAccept: true,
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
	payload := []byte("abcdefghijklmnopqrstuvwxyz0123456789") // 36 bytes
	initBody, _ := json.Marshal(chunkInitRequest{
		Files: []chunkFileMetaReq{{Name: "big.bin", Size: int64(len(payload))}},
	})
	res, err := http.Post(base+"/upload-init", "application/json", bytes.NewReader(initBody))
	if err != nil {
		t.Fatal(err)
	}
	var init chunkInitResponse
	_ = json.NewDecoder(res.Body).Decode(&init)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || init.ID == "" {
		t.Fatalf("init %d %+v", res.StatusCode, init)
	}
	if init.ChunkSize != DefaultChunkSize {
		t.Fatalf("chunkSize %d", init.ChunkSize)
	}

	// Send first chunk only (entire file fits in one chunk when small).
	req, _ := http.NewRequest(http.MethodPut, base+"/upload-chunk", bytes.NewReader(payload))
	req.Header.Set("X-Upload-Id", init.ID)
	req.Header.Set("X-File-Index", "0")
	req.Header.Set("X-Chunk-Index", "0")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("chunk %d %s", res.StatusCode, body)
	}

	// Idempotent re-send of same chunk (resume).
	req, _ = http.NewRequest(http.MethodPut, base+"/upload-chunk", bytes.NewReader(payload))
	req.Header.Set("X-Upload-Id", init.ID)
	req.Header.Set("X-File-Index", "0")
	req.Header.Set("X-Chunk-Index", "0")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("resume chunk %d", res.StatusCode)
	}

	fin, _ := json.Marshal(map[string]string{"id": init.ID})
	res, err = http.Post(base+"/upload-finish", "application/json", bytes.NewReader(fin))
	if err != nil {
		t.Fatal(err)
	}
	var done UploadResult
	_ = json.NewDecoder(res.Body).Decode(&done)
	res.Body.Close()
	if done.Status != "saved" {
		t.Fatalf("%+v", done)
	}
	got, err := os.ReadFile(filepath.Join(out, "big.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("saved bytes mismatch")
	}
}
