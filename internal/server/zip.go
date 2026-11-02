package server

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func (s *Server) serveZip(w http.ResponseWriter, r *http.Request) {
	// Streaming zip: no Content-Length, no Range (cannot seek mid-stream).
	if r.Header.Get("Range") != "" {
		http.Error(w, "range not supported for zip", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	p := s.payload()
	entries := p.ZipEntries
	if idx := parseFileIndices(r, len(entries)); len(idx) > 0 {
		filtered := make([]ZipEntry, 0, len(idx))
		for _, i := range idx {
			filtered = append(filtered, entries[i])
		}
		entries = filtered
	}
	if len(entries) == 0 {
		http.Error(w, "no files selected", http.StatusBadRequest)
		return
	}

	name := p.DownloadName
	if len(entries) != len(p.ZipEntries) {
		name = "selected.zip"
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", contentDisposition(name))
	w.Header().Set("Accept-Ranges", "none")
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, e := range entries {
		if err := writeZipEntry(zw, e); err != nil {
			// Client may have disconnected; nothing useful to send after headers.
			return
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func writeZipEntry(zw *zip.Writer, e ZipEntry) error {
	f, err := os.Open(e.AbsPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", e.AbsPath, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	hdr.Name = e.NameInZip
	hdr.Method = zip.Deflate
	hdr.Modified = fi.ModTime()
	if hdr.Modified.IsZero() {
		hdr.Modified = time.Now()
	}

	out, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, f)
	return err
}
