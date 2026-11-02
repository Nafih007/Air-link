package server

import (
	"net/http"
	"strings"

	"airlink/internal/pair"
)

func extractJoinCode(path string) (string, bool) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "j" || !isHex(parts[1], 32) {
		return "", false
	}
	return parts[1], true
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if s.rate != nil && !s.rate.allow(ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	code, ok := extractJoinCode(r.URL.Path)
	if !ok {
		if s.rate != nil {
			s.rate.fail(ip)
		}
		s.servePublicNotFound(w, r)
		return
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		s.servePage(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tok, status, err := s.claimJoin(code, ip, r.UserAgent())
	if err != nil {
		http.Error(w, "join error", http.StatusInternalServerError)
		return
	}
	switch status {
	case "ok", "pending":
		writeJSON(w, http.StatusOK, map[string]string{
			"status":      status,
			"deviceToken": tok,
			"page":        "/t/" + tok + "/",
		})
	case "used":
		if s.rate != nil {
			s.rate.fail(ip)
		}
		writeJSON(w, http.StatusGone, map[string]string{
			"status":  "used",
			"message": "This code was already used. Scan the new one on the PC",
		})
	case "expired":
		if s.rate != nil {
			s.rate.fail(ip)
		}
		writeJSON(w, http.StatusGone, map[string]string{
			"status":  "expired",
			"message": "This code expired. Scan the new one on the PC",
		})
	default:
		if s.rate != nil {
			s.rate.fail(ip)
		}
		writeJSON(w, http.StatusNotFound, map[string]string{
			"status":  "invalid",
			"message": "This code is not valid. Scan the QR on the PC",
		})
	}
}

func (s *Server) handleRejoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := clientIP(r)
	if s.rate != nil && !s.rate.allow(ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	tok := bearerToken(r)
	if s.cfg.Pairs == nil || tok == "" || !s.cfg.Pairs.Valid(tok) {
		if s.rate != nil {
			s.rate.fail(ip)
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	dev, err := s.admitPaired(ip, r.UserAgent())
	if err != nil {
		http.Error(w, "rejoin error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":      "ok",
		"deviceToken": dev,
		"page":        "/t/" + dev + "/",
	})
}

func (s *Server) admitPaired(ip, ua string) (string, error) {
	tok, err := pair.NewToken()
	if err != nil {
		return "", err
	}
	a := s.acc()
	a.mu.Lock()
	id := memberID(tok)
	m := &member{
		ID:        id,
		TokenHash: pair.HashToken(tok),
		IP:        ip,
		Browser:   shortBrowser(ua),
		Status:    devAllowed,
		Paired:    true,
	}
	a.members[id] = m
	a.byHash[m.TokenHash] = m
	a.mu.Unlock()
	s.emitDevices()
	return tok, nil
}

func (s *Server) rejectMember(w http.ResponseWriter, r *http.Request, m *member) bool {
	if m == nil {
		s.servePublicNotFound(w, r)
		return true
	}
	switch m.Status {
	case devAllowed:
		return false
	case devPending:
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":   "pending",
			"message": "Waiting for the PC to Allow this device",
		})
		return true
	case devDenied:
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":   "denied",
			"message": "The PC denied this device",
		})
		return true
	case devKicked:
		http.Error(w, "kicked", http.StatusForbidden)
		return true
	default:
		s.servePublicNotFound(w, r)
		return true
	}
}
