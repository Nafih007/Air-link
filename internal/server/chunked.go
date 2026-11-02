package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultChunkSize is 1 MiB — small enough to resume on flaky Wi‑Fi.
const DefaultChunkSize = 1 << 20

type chunkInitRequest struct {
	Text  string             `json:"text"`
	Files []chunkFileMetaReq `json:"files"`
}

type chunkFileMetaReq struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type chunkInitResponse struct {
	ID        string `json:"id"`
	ChunkSize int64  `json:"chunkSize"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

type chunkFileState struct {
	Name        string
	Size        int64
	ChunkSize   int64
	TotalChunks int
	Received    map[int]bool
	BytesGot    int64
}

type chunkSession struct {
	ID    string
	Dir   string
	Text  string
	Files []chunkFileState
	Bytes int64 // total expected
	Got   int64 // total received
}

type chunkProgressResponse struct {
	Status  string              `json:"status"` // receiving | pending | saved | …
	ID      string              `json:"id"`
	Got     int64               `json:"got"`
	Total   int64               `json:"total"`
	Files   []chunkFileProgress `json:"files,omitempty"`
	Saved   []string            `json:"saved,omitempty"`
	Message string              `json:"message,omitempty"`
}

type chunkFileProgress struct {
	Index       int   `json:"index"`
	Received    int   `json:"received"`
	TotalChunks int   `json:"totalChunks"`
	BytesGot    int64 `json:"bytesGot"`
	Size        int64 `json:"size"`
}

func (s *Server) serveUploadInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.noteHit()
	var req chunkInitRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" && len(req.Files) == 0 {
		http.Error(w, "no files or text", http.StatusBadRequest)
		return
	}

	dir, err := os.MkdirTemp("", "sendtophone-chunk-*")
	if err != nil {
		http.Error(w, "temp error", http.StatusInternalServerError)
		return
	}
	id, err := newUploadID()
	if err != nil {
		_ = os.RemoveAll(dir)
		http.Error(w, "id error", http.StatusInternalServerError)
		return
	}

	sess := &chunkSession{ID: id, Dir: dir, Text: req.Text}
	for _, f := range req.Files {
		if f.Size < 0 || f.Size > s.uploadLimit() {
			_ = os.RemoveAll(dir)
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		name := SanitizeUploadName(f.Name)
		total := int((f.Size + DefaultChunkSize - 1) / DefaultChunkSize)
		if f.Size == 0 {
			total = 1
		}
		sess.Files = append(sess.Files, chunkFileState{
			Name:        name,
			Size:        f.Size,
			ChunkSize:   DefaultChunkSize,
			TotalChunks: total,
			Received:    make(map[int]bool),
		})
		sess.Bytes += f.Size
	}
	if req.Text != "" {
		sess.Bytes += int64(len(req.Text))
		sess.Got += int64(len(req.Text))
	}

	st := s.uploads()
	st.mu.Lock()
	if st.chunks == nil {
		st.chunks = make(map[string]*chunkSession)
	}
	st.chunks[id] = sess
	st.mu.Unlock()

	writeJSON(w, http.StatusOK, chunkInitResponse{
		ID:        id,
		ChunkSize: DefaultChunkSize,
		Status:    "receiving",
		Message:   "Send chunks then finish",
	})
}

func (s *Server) serveUploadChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.noteHit()

	id := r.Header.Get("X-Upload-Id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	fileIdx, _ := strconv.Atoi(firstNonEmpty(r.Header.Get("X-File-Index"), r.URL.Query().Get("file")))
	chunkIdx, _ := strconv.Atoi(firstNonEmpty(r.Header.Get("X-Chunk-Index"), r.URL.Query().Get("chunk")))
	if id == "" || fileIdx < 0 || chunkIdx < 0 {
		http.Error(w, "missing id/file/chunk", http.StatusBadRequest)
		return
	}

	st := s.uploads()
	st.mu.Lock()
	sess := st.chunks[id]
	st.mu.Unlock()
	if sess == nil {
		http.Error(w, "unknown upload", http.StatusNotFound)
		return
	}
	if fileIdx >= len(sess.Files) {
		http.Error(w, "bad file index", http.StatusBadRequest)
		return
	}
	cf := &sess.Files[fileIdx]
	if chunkIdx >= cf.TotalChunks {
		http.Error(w, "bad chunk index", http.StatusBadRequest)
		return
	}

	// Already have this chunk — idempotent resume.
	if cf.Received[chunkIdx] {
		writeJSON(w, http.StatusOK, s.chunkProgressLocked(sess))
		return
	}

	path := chunkPath(sess.Dir, fileIdx, chunkIdx)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}
	n, err := io.Copy(f, io.LimitReader(r.Body, DefaultChunkSize+1))
	_ = f.Close()
	if err != nil || n > DefaultChunkSize {
		_ = os.Remove(path)
		http.Error(w, "read chunk", http.StatusBadRequest)
		return
	}

	// Validate size for non-final chunks.
	expect := int64(DefaultChunkSize)
	if chunkIdx == cf.TotalChunks-1 {
		rem := cf.Size - int64(chunkIdx)*DefaultChunkSize
		if rem < 0 {
			rem = 0
		}
		expect = rem
		if cf.Size == 0 {
			expect = 0
		}
	}
	if n != expect {
		_ = os.Remove(path)
		http.Error(w, fmt.Sprintf("chunk size %d want %d", n, expect), http.StatusBadRequest)
		return
	}

	st.mu.Lock()
	if !cf.Received[chunkIdx] {
		cf.Received[chunkIdx] = true
		cf.BytesGot += n
		sess.Got += n
	}
	prog := s.chunkProgressLocked(sess)
	st.mu.Unlock()

	if s.cfg.Hooks.OnUploadProgress != nil {
		s.cfg.Hooks.OnUploadProgress(prog.Got, prog.Total)
	}
	writeJSON(w, http.StatusOK, prog)
}

func (s *Server) serveUploadFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.noteHit()
	var body struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	if body.ID == "" {
		body.ID = r.URL.Query().Get("id")
	}
	if body.ID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	st := s.uploads()
	st.mu.Lock()
	sess := st.chunks[body.ID]
	if sess != nil {
		delete(st.chunks, body.ID)
	}
	st.mu.Unlock()
	if sess == nil {
		http.Error(w, "unknown upload", http.StatusNotFound)
		return
	}

	for i, cf := range sess.Files {
		if len(cf.Received) != cf.TotalChunks {
			_ = os.RemoveAll(sess.Dir)
			http.Error(w, fmt.Sprintf("file %d incomplete", i), http.StatusBadRequest)
			return
		}
	}

	batch := &pendingBatch{ID: body.ID, Text: sess.Text, TotalSize: sess.Bytes}
	for i, cf := range sess.Files {
		assembled := filepath.Join(sess.Dir, fmt.Sprintf("file-%d-final", i))
		if err := assembleChunks(sess.Dir, i, cf.TotalChunks, assembled); err != nil {
			_ = os.RemoveAll(sess.Dir)
			http.Error(w, "assemble failed", http.StatusInternalServerError)
			return
		}
		batch.TempFiles = append(batch.TempFiles, pendingFile{TempPath: assembled, Name: cf.Name, Size: cf.Size})
	}

	if s.cfg.AutoAccept {
		saved, err := s.commitBatch(batch)
		_ = os.RemoveAll(sess.Dir)
		if err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		msg := "Saved on PC"
		if batch.Text != "" && len(saved) == 0 {
			msg = "Copied to PC clipboard"
		}
		s.settle(body.ID, UploadResult{Status: "saved", Saved: saved, Message: msg})
		writeJSON(w, http.StatusOK, UploadResult{Status: "saved", ID: body.ID, Saved: saved, Message: msg})
		s.notifyCommit(saved, batch)
		return
	}

	// Keep assembled files; drop individual chunk parts.
	for i, cf := range sess.Files {
		for c := 0; c < cf.TotalChunks; c++ {
			_ = os.Remove(chunkPath(sess.Dir, i, c))
		}
	}
	st.mu.Lock()
	st.pending[body.ID] = batch
	st.mu.Unlock()

	if s.cfg.Hooks.OnUploadPending != nil {
		s.cfg.Hooks.OnUploadPending(body.ID, len(batch.TempFiles), batch.TotalSize, sess.Text != "")
	}
	writeJSON(w, http.StatusAccepted, UploadResult{
		Status:  "pending",
		ID:      body.ID,
		Message: "Waiting for PC approval",
	})
}

func (s *Server) chunkProgressLocked(sess *chunkSession) chunkProgressResponse {
	out := chunkProgressResponse{
		Status: "receiving",
		ID:     sess.ID,
		Got:    sess.Got,
		Total:  sess.Bytes,
	}
	for i, cf := range sess.Files {
		out.Files = append(out.Files, chunkFileProgress{
			Index:       i,
			Received:    len(cf.Received),
			TotalChunks: cf.TotalChunks,
			BytesGot:    cf.BytesGot,
			Size:        cf.Size,
		})
	}
	return out
}

// enhance serveUploadStatus to report chunk receiving progress
func (s *Server) serveUploadStatusChunkAware(w http.ResponseWriter, r *http.Request, id string) bool {
	st := s.uploads()
	st.mu.Lock()
	sess := st.chunks[id]
	var prog chunkProgressResponse
	if sess != nil {
		prog = s.chunkProgressLocked(sess)
	}
	st.mu.Unlock()
	if sess != nil {
		writeJSON(w, http.StatusOK, prog)
		return true
	}
	return false
}

func chunkPath(dir string, fileIdx, chunkIdx int) string {
	return filepath.Join(dir, fmt.Sprintf("%d.%06d", fileIdx, chunkIdx))
}

func assembleChunks(dir string, fileIdx, total int, dest string) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	for c := 0; c < total; c++ {
		in, err := os.Open(chunkPath(dir, fileIdx, c))
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		in.Close()
		if err != nil {
			return err
		}
	}
	return out.Close()
}

func newUploadID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "c" + hex.EncodeToString(b[:]), nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
