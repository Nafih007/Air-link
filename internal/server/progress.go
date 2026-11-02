package server

import (
	"net/http"
)

type progressWriter struct {
	http.ResponseWriter
	written    int64
	total      int64
	onProgress func(written, total int64)
	lastReport int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.ResponseWriter.Write(b)
	p.written += int64(n)
	// Throttle UI updates (~64 KiB) so we don't flood the WebView.
	if p.onProgress != nil && (p.written-p.lastReport >= 64*1024 || err != nil || (p.total > 0 && p.written >= p.total)) {
		p.lastReport = p.written
		p.onProgress(p.written, p.total)
	}
	return n, err
}

func (p *progressWriter) Flush() {
	if f, ok := p.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) wrapDownload(inner http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.noteHit()
		total := int64(-1)
		if p := s.payload(); p.Single != nil {
			total = p.Single.Size
		}
		pw := &progressWriter{
			ResponseWriter: w,
			total:          total,
			onProgress:     s.cfg.Hooks.OnProgress,
		}
		if s.cfg.Hooks.OnProgress != nil {
			s.cfg.Hooks.OnProgress(0, total)
		}
		inner(pw, r)
		if s.cfg.Hooks.OnProgress != nil && pw.written > 0 {
			s.cfg.Hooks.OnProgress(pw.written, total)
		}
		s.noteTransferComplete()
	}
}
