package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"sendtophone/internal/pair"
)

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if !strings.HasPrefix(h, p) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, p))
}

func (s *Server) handleAPIUpload(w http.ResponseWriter, r *http.Request) {
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
	s.noteHit()
	limit := s.uploadLimit()
	if limit > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, limit+32<<20)
	}
	s.serveUpload(w, r)
}

func (s *Server) servePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.Pairs == nil {
		http.Error(w, "pairing disabled", http.StatusNotFound)
		return
	}
	s.noteHit()
	tok, err := pair.NewToken()
	if err != nil {
		http.Error(w, "pair error", http.StatusInternalServerError)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "Phone"
	}
	rec, err := s.cfg.Pairs.Add(name, tok)
	if err != nil {
		http.Error(w, "save pair", http.StatusInternalServerError)
		return
	}
	if s.cfg.Hooks.OnPairsChanged != nil {
		s.cfg.Hooks.OnPairsChanged(s.cfg.Pairs.List())
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":          rec.ID,
		"name":        rec.Name,
		"deviceToken": tok,
		"home":        "http://" + s.Addr() + "/home",
	})
}
