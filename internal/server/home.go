package server

import (
	"net/http"
)

func (s *Server) serveHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.Pairs == nil {
		http.NotFound(w, r)
		return
	}
	s.servePage(w, r)
}

func (s *Server) serveManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := readWebFile("manifest.json")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

// HomeURL is the stable Add-to-Home-Screen page (no session token in the path).
func (s *Server) HomeURL() string {
	return "http://" + s.Addr() + "/home"
}
