package server

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) serveTextDownload(w http.ResponseWriter, r *http.Request) {
	p := s.payload()
	body := p.Text
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", contentDisposition(p.DownloadName))
	http.ServeContent(w, r, p.DownloadName, time.Now(), strings.NewReader(body))
}
