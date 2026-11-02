package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxUploadPart = 512 << 20 // 512 MiB per part (default)

func (s *Server) uploadLimit() int64 {
	if s.cfg.MaxUpload > 0 {
		return s.cfg.MaxUpload
	}
	return maxUploadPart
}

// UploadResult is returned to the phone after POST /upload or status poll.
type UploadResult struct {
	Status  string   `json:"status"` // saved | pending | declined | unknown
	ID      string   `json:"id,omitempty"`
	Saved   []string `json:"saved,omitempty"`
	Message string   `json:"message,omitempty"`
}

type pendingBatch struct {
	ID        string
	Created   time.Time
	Text      string
	TempFiles []pendingFile
	TotalSize int64
}

type pendingFile struct {
	TempPath string
	Name     string
	Size     int64
}

type uploadState struct {
	mu      sync.Mutex
	pending map[string]*pendingBatch
	settled map[string]UploadResult
	chunks  map[string]*chunkSession
	seq     int
}

func (s *Server) uploads() *uploadState {
	s.uploadOnce.Do(func() {
		s.upload = &uploadState{
			pending: make(map[string]*pendingBatch),
			settled: make(map[string]UploadResult),
			chunks:  make(map[string]*chunkSession),
		}
	})
	return s.upload
}

func (s *Server) settle(id string, result UploadResult) {
	st := s.uploads()
	st.mu.Lock()
	defer st.mu.Unlock()
	result.ID = id
	st.settled[id] = result
}

func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.noteHit()

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad multipart", http.StatusBadRequest)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	text := strings.TrimSpace(r.FormValue("text"))
	files := r.MultipartForm.File["files"]
	if text == "" && len(files) == 0 {
		http.Error(w, "no files or text", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "airlink-up-*")
	if err != nil {
		http.Error(w, "temp error", http.StatusInternalServerError)
		return
	}

	batch := &pendingBatch{
		Created: time.Now(),
		Text:    text,
	}
	cleanupTemps := func() {
		for _, f := range batch.TempFiles {
			_ = os.Remove(f.TempPath)
		}
		_ = os.RemoveAll(tmpDir)
	}

	for _, fh := range files {
		limit := s.uploadLimit()
		if fh.Size > limit {
			cleanupTemps()
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		src, err := fh.Open()
		if err != nil {
			cleanupTemps()
			http.Error(w, "open part", http.StatusBadRequest)
			return
		}
		name := SanitizeUploadName(fh.Filename)
		dstPath := filepath.Join(tmpDir, fmt.Sprintf("%d-%s", len(batch.TempFiles), name))
		dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
		if err != nil {
			src.Close()
			cleanupTemps()
			http.Error(w, "temp write", http.StatusInternalServerError)
			return
		}
		n, err := io.Copy(dst, io.LimitReader(src, limit+1))
		src.Close()
		dst.Close()
		if err != nil || n > limit {
			cleanupTemps()
			http.Error(w, "read part", http.StatusBadRequest)
			return
		}
		batch.TempFiles = append(batch.TempFiles, pendingFile{TempPath: dstPath, Name: name, Size: n})
		batch.TotalSize += n
	}
	if text != "" {
		batch.TotalSize += int64(len(text))
	}

	auto := s.cfg.AutoAccept || strings.HasPrefix(r.URL.Path, "/api/")
	if auto {
		saved, err := s.commitBatch(batch)
		cleanupTemps()
		if err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		msg := "Saved on PC"
		if batch.Text != "" && len(saved) == 0 {
			msg = "Copied to PC clipboard"
		}
		writeJSON(w, http.StatusOK, UploadResult{
			Status:  "saved",
			Saved:   saved,
			Message: msg,
		})
		s.notifyCommit(saved, batch)
		return
	}

	st := s.uploads()
	st.mu.Lock()
	st.seq++
	batch.ID = fmt.Sprintf("u%d", st.seq)
	st.pending[batch.ID] = batch
	st.mu.Unlock()

	if s.cfg.Hooks.OnUploadPending != nil {
		s.cfg.Hooks.OnUploadPending(batch.ID, len(batch.TempFiles), batch.TotalSize, text != "")
	}

	writeJSON(w, http.StatusAccepted, UploadResult{
		Status:  "pending",
		ID:      batch.ID,
		Message: "Waiting for PC approval",
	})
}

func (s *Server) serveUploadStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.noteHit()
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	if s.serveUploadStatusChunkAware(w, r, id) {
		return
	}
	st := s.uploads()
	st.mu.Lock()
	_, pending := st.pending[id]
	settled, ok := st.settled[id]
	st.mu.Unlock()
	if pending {
		writeJSON(w, http.StatusOK, UploadResult{Status: "pending", ID: id, Message: "Waiting for PC approval"})
		return
	}
	if ok {
		writeJSON(w, http.StatusOK, settled)
		return
	}
	writeJSON(w, http.StatusOK, UploadResult{Status: "unknown", ID: id, Message: "No longer pending (expired)"})
}

// DefaultUploadDir is ~/Downloads/Air-Link.
func DefaultUploadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads", "Air-Link"), nil
}

// UploadFolder returns the configured or default save directory.
func (s *Server) UploadFolder() string {
	if s.cfg.UploadDir != "" {
		return s.cfg.UploadDir
	}
	dir, err := DefaultUploadDir()
	if err != nil {
		return ""
	}
	return dir
}

func (s *Server) commitBatch(batch *pendingBatch) ([]string, error) {
	dir := s.UploadFolder()
	if dir == "" {
		return nil, fmt.Errorf("upload dir unavailable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir upload dir: %w", err)
	}
	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}
	var saved []string
	for _, f := range batch.TempFiles {
		if s.cfg.ConvertToJPG && WantsJPGConvert(f.Name) {
			dest := UniquePath(dir, jpgName(f.Name), exists)
			if err := ConvertToJPEG(f.TempPath, dest); err == nil {
				saved = append(saved, filepath.Base(dest))
				continue
			}
			// Keep the original bytes if conversion is unavailable (typical for HEIC without a codec).
		}
		dest := UniquePath(dir, f.Name, exists)
		if err := copyFile(f.TempPath, dest); err != nil {
			return saved, err
		}
		saved = append(saved, filepath.Base(dest))
	}
	// Text is delivered to the PC clipboard (F3.6), not written as a file.
	return saved, nil
}

func (s *Server) notifyCommit(saved []string, batch *pendingBatch) {
	if batch.Text != "" && s.cfg.Hooks.OnClipboardText != nil {
		s.cfg.Hooks.OnClipboardText(batch.Text)
	}
	if len(batch.TempFiles) > 0 && s.cfg.Hooks.OnUploadSaved != nil {
		s.cfg.Hooks.OnUploadSaved(len(saved), batch.TotalSize)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func cleanupBatchTemps(batch *pendingBatch) {
	for _, f := range batch.TempFiles {
		_ = os.Remove(f.TempPath)
	}
	if len(batch.TempFiles) > 0 {
		_ = os.Remove(filepath.Dir(batch.TempFiles[0].TempPath))
	}
}

// AcceptPending saves a pending batch (M3.3). Returns saved basenames.
func (s *Server) AcceptPending(id string) ([]string, error) {
	st := s.uploads()
	st.mu.Lock()
	batch, ok := st.pending[id]
	if ok {
		delete(st.pending, id)
	}
	st.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no pending upload %s", id)
	}
	saved, err := s.commitBatch(batch)
	cleanupBatchTemps(batch)
	if err != nil {
		s.settle(id, UploadResult{Status: "unknown", Message: "Save failed"})
		return nil, err
	}
	msg := "Saved on PC"
	if batch.Text != "" && len(saved) == 0 {
		msg = "Copied to PC clipboard"
	}
	s.settle(id, UploadResult{
		Status:  "saved",
		Saved:   saved,
		Message: msg,
	})
	s.notifyCommit(saved, batch)
	return saved, nil
}

// DeclinePending discards a pending batch.
func (s *Server) DeclinePending(id string) error {
	st := s.uploads()
	st.mu.Lock()
	batch, ok := st.pending[id]
	if ok {
		delete(st.pending, id)
	}
	st.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending upload %s", id)
	}
	cleanupBatchTemps(batch)
	s.settle(id, UploadResult{Status: "declined", Message: "Declined on PC"})
	return nil
}
