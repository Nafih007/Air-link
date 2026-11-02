package server

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"sendtophone/web"
)

func readWebFile(name string) ([]byte, error) {
	return web.FS.ReadFile(name)
}

func (s *Server) serveWebPNG(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := readWebFile(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	}
}

// servePublicNotFound is the phone 404. It never writes the request path or token.
func (s *Server) servePublicNotFound(w http.ResponseWriter, r *http.Request) {
	data, err := readWebFile("404.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}

func (s *Server) servePage(w http.ResponseWriter, r *http.Request) {
	data, err := readWebFile("index.html")
	if err != nil {
		http.Error(w, "page error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func (s *Server) serveDownload(w http.ResponseWriter, r *http.Request) {
	p := s.payload()
	file := p.Single
	if file == nil {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(file.AbsPath)
	if err != nil {
		http.Error(w, "file unavailable", http.StatusServiceUnavailable)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "file unavailable", http.StatusServiceUnavailable)
		return
	}

	mime := MIMEByFilename(p.DownloadName)
	w.Header().Set("Content-Disposition", contentDisposition(p.DownloadName))
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, p.DownloadName, fi.ModTime(), f)
}

// routeAfterToken returns route kind and optional file index.
// kinds: page, download, meta, appjs, file, unknown
func routeAfterToken(path string) (kind string, fileIndex int) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	// ["t", token, ...]
	if len(parts) < 2 {
		return "unknown", -1
	}
	rest := ""
	if len(parts) >= 3 {
		rest = parts[2]
	}
	switch rest {
	case "", "index.html":
		return "page", -1
	case "download":
		return "download", -1
	case "meta":
		return "meta", -1
	case "app.js":
		return "appjs", -1
	case "upload":
		return "upload", -1
	case "upload-init":
		return "upload-init", -1
	case "upload-chunk":
		return "upload-chunk", -1
	case "upload-finish":
		return "upload-finish", -1
	case "upload-status":
		return "upload-status", -1
	case "pin":
		return "pin", -1
	case "pair":
		return "pair", -1
	case "file":
		if len(parts) < 4 {
			return "unknown", -1
		}
		n, err := strconv.Atoi(parts[3])
		if err != nil || n < 0 {
			return "unknown", -1
		}
		return "file", n
	default:
		return "unknown", -1
	}
}
