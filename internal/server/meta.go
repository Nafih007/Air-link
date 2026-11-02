package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FileMeta is one item shown on the phone page (preview / gallery).
type FileMeta struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MIME  string `json:"mime"`
	Kind  string `json:"kind"`
	URL   string `json:"url"`
}

// SessionMeta is the JSON returned by /t/{token}/meta.
type SessionMeta struct {
	Mode         string     `json:"mode"` // "file", "text", "receive", or "room"
	DisplayName  string     `json:"displayName"`
	DownloadName string     `json:"downloadName"`
	Size         int64      `json:"size"`
	Text         string     `json:"text,omitempty"`
	Files        []FileMeta `json:"files"`
	CanPair      bool       `json:"canPair,omitempty"`
	APIURL       string     `json:"apiUrl,omitempty"`
	HomeURL      string     `json:"homeUrl,omitempty"`
	Room         bool       `json:"room,omitempty"`
}

func (s *Server) buildMeta() SessionMeta {
	p := s.payload()
	meta := SessionMeta{
		Mode:         "file",
		DisplayName:  p.DisplayName,
		DownloadName: p.DownloadName,
		Size:         p.Size,
		Room:         p.Room,
	}
	if s.cfg.Pairs != nil {
		meta.CanPair = true
		meta.APIURL = "http://" + s.Addr() + "/api/upload"
		meta.HomeURL = s.HomeURL()
	}
	if p.ReceiveOnly {
		meta.Mode = "receive"
		meta.DisplayName = "Receive from phone"
		return meta
	}
	if p.Room && !p.HasDownload() {
		meta.Mode = "room"
		meta.DisplayName = "Waiting for phone…"
		return meta
	}
	if p.Text != "" {
		meta.Mode = "text"
		meta.Text = p.Text
		return meta
	}
	if p.Single != nil {
		f := p.Single
		meta.Files = []FileMeta{{
			Index: 0,
			Name:  f.Name,
			Size:  f.Size,
			MIME:  MIMEByFilename(f.Name),
			Kind:  FileKind(f.Name),
			URL:   "file/0",
		}}
		return meta
	}
	for i, e := range p.ZipEntries {
		name := filepath.Base(e.NameInZip)
		meta.Files = append(meta.Files, FileMeta{
			Index: i,
			Name:  name,
			Size:  0, // unknown cheaply without stat; filled below
			MIME:  MIMEByFilename(name),
			Kind:  FileKind(name),
			URL:   "file/" + strconv.Itoa(i),
		})
		if fi, err := os.Stat(e.AbsPath); err == nil {
			meta.Files[i].Size = fi.Size()
		}
	}
	return meta
}

func (s *Server) serveMeta(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.buildMeta())
}

func (s *Server) serveAppJS(w http.ResponseWriter, r *http.Request) {
	data, err := readWebFile("app.js")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func (s *Server) serveIndexedFile(w http.ResponseWriter, r *http.Request, index int) {
	p := s.payload()
	var abs, name string
	if p.Single != nil {
		if index != 0 {
			http.NotFound(w, r)
			return
		}
		abs = p.Single.AbsPath
		name = p.Single.Name
	} else {
		if index < 0 || index >= len(p.ZipEntries) {
			http.NotFound(w, r)
			return
		}
		e := p.ZipEntries[index]
		abs = e.AbsPath
		name = filepath.Base(e.NameInZip)
	}

	f, err := os.Open(abs)
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

	mime := MIMEByFilename(name)
	w.Header().Set("Content-Type", mime)
	// Inline for preview; attachment still available via /download.
	w.Header().Set("Content-Disposition", "inline; filename=\""+sanitizeASCIIFilename(name)+"\"")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

func sanitizeASCIIFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 0x20 && r <= 0x7e && r != '"' && r != '\\' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "file"
	}
	return b.String()
}
