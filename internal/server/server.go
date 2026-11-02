package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"airlink/internal/pair"
)

// Config for a send session.
type Config struct {
	BindIP       string
	Token        string
	Payload      Payload
	Hooks        Hooks
	UploadDir    string // default: ~/Downloads/Air-Link
	AutoAccept   bool   // skip PC approval (M3.3); required to save until approval UI exists
	ConvertToJPG bool   // optional: save still images as JPEG (HEIC stays HEIC if convert fails)
	PIN          string // optional 4-digit PIN; empty = off
	Pairs        PairStore
	MaxUpload    int64 // 0 = default maxUploadPart
	Port         int   // 0 = ephemeral; receive mode prefers DefaultReceivePort
}

// DefaultReceivePort is the stable listen port for -receive / Home Screen.
const DefaultReceivePort = 17612

// PairStore is the paired-device registry (hashes only).
type PairStore interface {
	Add(name, token string) (pair.Record, error)
	Valid(token string) bool
	Remove(id string) error
	Rename(id, name string) error
	List() []pair.Record
}

// Server is a short-lived LAN file server.
type Server struct {
	cfg        Config
	payloadMu  sync.RWMutex
	listener   net.Listener
	httpSrv    *http.Server
	act        activity
	uploadOnce sync.Once
	upload     *uploadState
	pin        pinState
	rate       *rateLimiter
	access     *accessState
}

// payload is a snapshot of the current send payload.
func (s *Server) payload() Payload {
	s.payloadMu.RLock()
	defer s.payloadMu.RUnlock()
	return s.cfg.Payload
}

// Start listens on BindIP with a random free port.
// BindIP must be a concrete unicast address — not 0.0.0.0 or ::.
func Start(cfg Config) (*Server, error) {
	if err := validateBindIP(cfg.BindIP); err != nil {
		return nil, err
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}
	if !cfg.Payload.ReceiveOnly && !cfg.Payload.Room && !cfg.Payload.HasDownload() {
		return nil, fmt.Errorf("payload is required")
	}

	port := "0"
	if cfg.Port > 0 && cfg.Port < 65536 {
		port = fmt.Sprintf("%d", cfg.Port)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(cfg.BindIP, port))
	if err != nil && cfg.Port > 0 {
		ln, err = net.Listen("tcp", net.JoinHostPort(cfg.BindIP, "0"))
	}
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", cfg.BindIP, err)
	}

	s := &Server{cfg: cfg, listener: ln, rate: newRateLimiter(), access: newAccess()}
	if err := s.RotateJoin(); err != nil {
		_ = ln.Close()
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", s.handleAPIUpload)
	mux.HandleFunc("/rejoin", s.handleRejoin)
	mux.HandleFunc("/home", s.serveHome)
	mux.HandleFunc("/home/", s.serveHome)
	mux.HandleFunc("/manifest.json", s.serveManifest)
	mux.HandleFunc("/app.js", s.serveAppJS)
	mux.HandleFunc("/favicon-32.png", s.serveWebPNG("favicon-32.png"))
	mux.HandleFunc("/favicon.ico", s.serveWebPNG("favicon-32.png"))
	mux.HandleFunc("/apple-touch-icon.png", s.serveWebPNG("apple-touch-icon.png"))
	mux.HandleFunc("/icon-192.png", s.serveWebPNG("icon-192.png"))
	mux.HandleFunc("/icon-512.png", s.serveWebPNG("icon-512.png"))
	mux.HandleFunc("/logomark-transparent.png", s.serveWebPNG("logomark-transparent.png"))
	mux.HandleFunc("/", s.handle)
	s.httpSrv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		_ = s.httpSrv.Serve(ln)
	}()
	return s, nil
}

// Addr returns the bound host:port (no scheme, no token).
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) urlPrefix() string {
	host := s.Addr()
	if ip := net.ParseIP(s.cfg.BindIP); ip != nil && ip.To4() == nil {
		_, port, _ := net.SplitHostPort(host)
		host = net.JoinHostPort(s.cfg.BindIP, port)
	}
	return "http://" + host
}

// SessionURL is the current unused join QR. Must not be logged.
func (s *Server) SessionURL() string {
	return s.JoinURL()
}

// Close stops the HTTP server and drops pending uploads.
func (s *Server) Close() error {
	s.discardPending()
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Close()
}

func (s *Server) discardPending() {
	st := s.uploads()
	st.mu.Lock()
	pending := st.pending
	st.pending = make(map[string]*pendingBatch)
	chunks := st.chunks
	st.chunks = make(map[string]*chunkSession)
	st.mu.Unlock()
	for _, batch := range pending {
		cleanupBatchTemps(batch)
	}
	for _, sess := range chunks {
		_ = os.RemoveAll(sess.Dir)
	}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if s.rate != nil && !s.rate.allow(ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/j/") {
		s.handleJoin(w, r)
		return
	}

	token, ok := extractToken(r.URL.Path)
	mem := s.lookupMember(token)
	if !ok || mem == nil {
		if s.rate != nil {
			s.rate.fail(ip)
		}
		s.servePublicNotFound(w, r)
		return
	}

	kind, fileIndex := routeAfterToken(r.URL.Path)
	if kind == "pin" {
		s.servePIN(w, r)
		return
	}
	if mem.Status == devKicked {
		http.Error(w, "kicked", http.StatusForbidden)
		return
	}
	if mem.Status != devAllowed && kind != "page" && kind != "appjs" {
		s.rejectMember(w, r, mem)
		return
	}
	if s.pinEnabled() && kind != "page" && kind != "appjs" {
		if s.pinLocked() {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "locked"})
			return
		}
		if !s.pinCookieOK(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "pin"})
			return
		}
	}
	if kind == "pair" {
		s.servePair(w, r)
		return
	}

	switch kind {
	case "page":
		s.noteHit()
		s.noteDevice(r)
		s.servePage(w, r)
	case "appjs":
		s.serveAppJS(w, r)
	case "meta":
		s.noteHit()
		s.serveMeta(w, r)
	case "file":
		p := s.payload()
		if !p.HasDownload() {
			http.NotFound(w, r)
			return
		}
		s.noteHit()
		s.serveIndexedFile(w, r, fileIndex)
	case "download":
		p := s.payload()
		if !p.HasDownload() {
			http.NotFound(w, r)
			return
		}
		if p.Text != "" {
			s.wrapDownload(s.serveTextDownload)(w, r)
			return
		}
		if p.Single != nil {
			s.wrapDownload(s.serveDownload)(w, r)
			return
		}
		s.wrapDownload(s.serveZip)(w, r)
	case "upload":
		s.serveUpload(w, r)
	case "upload-init":
		s.serveUploadInit(w, r)
	case "upload-chunk":
		s.serveUploadChunk(w, r)
	case "upload-finish":
		s.serveUploadFinish(w, r)
	case "upload-status":
		s.serveUploadStatus(w, r)
	default:
		http.NotFound(w, r)
	}
}

// extractToken parses /t/{token}/... — anything else is invalid.
func extractToken(path string) (string, bool) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "t" || parts[1] == "" {
		return "", false
	}
	tok := parts[1]
	if !isHex(tok, 64) {
		return "", false
	}
	return tok, true
}

func contentDisposition(name string) string {
	fallback := "download"
	var b strings.Builder
	for _, r := range name {
		if r >= 0x20 && r <= 0x7e && r != '"' && r != '\\' {
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		fallback = b.String()
	}
	encoded := url.PathEscape(name)
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fallback, encoded)
}

func validateBindIP(ipStr string) error {
	if ipStr == "" {
		return fmt.Errorf("bind IP is required")
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid bind IP %q", ipStr)
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("refusing to bind to %s (use a concrete LAN IP)", ipStr)
	}
	if ip.IsLoopback() {
		return fmt.Errorf("refusing to bind to loopback %s (phone cannot reach it)", ipStr)
	}
	return nil
}
