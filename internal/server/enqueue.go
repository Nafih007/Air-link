package server

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnqueuePaths adds files/folders to a live session (room or send).
// Existing URLs keep working; new indexes are appended. Streams from disk.
func (s *Server) EnqueuePaths(paths []string) ([]FileMeta, error) {
	extra, err := ResolvePaths(paths)
	if err != nil {
		return nil, err
	}
	s.payloadMu.Lock()
	defer s.payloadMu.Unlock()
	cur := s.cfg.Payload
	merged, added, err := mergePayloads(cur, extra)
	if err != nil {
		return nil, err
	}
	s.cfg.Payload = merged
	return added, nil
}

func mergePayloads(cur, extra Payload) (Payload, []FileMeta, error) {
	if extra.ReceiveOnly {
		return cur, nil, fmt.Errorf("cannot enqueue a receive-only payload")
	}
	if extra.Text != "" && !extra.HasDownload() {
		return cur, nil, fmt.Errorf("cannot enqueue text-only into a file room")
	}
	base := payloadEntries(cur)
	add := payloadEntries(extra)
	if len(add) == 0 {
		return cur, nil, fmt.Errorf("nothing to enqueue")
	}
	used := map[string]int{}
	for _, e := range base {
		used[e.NameInZip] = 1
	}
	var added []FileMeta
	start := len(base)
	for _, e := range add {
		e.NameInZip = uniqueZipName(e.NameInZip, used)
		base = append(base, e)
		st, err := os.Lstat(e.AbsPath)
		size := int64(0)
		if err == nil {
			size = st.Size()
		}
		name := filepath.Base(e.NameInZip)
		idx := start + len(added)
		added = append(added, FileMeta{
			Index: idx,
			Name:  name,
			Size:  size,
			MIME:  MIMEByFilename(name),
			Kind:  FileKind(name),
			URL:   fmt.Sprintf("file/%d", idx),
		})
	}
	out := cur
	out.Room = cur.Room
	out.Text = ""
	out.Size = 0
	for _, e := range base {
		if st, err := os.Lstat(e.AbsPath); err == nil {
			out.Size += st.Size()
		}
	}
	if len(base) == 1 {
		name := filepath.Base(base[0].NameInZip)
		st, _ := os.Lstat(base[0].AbsPath)
		sz := int64(0)
		if st != nil {
			sz = st.Size()
		}
		out.Single = &ResolvedFile{AbsPath: base[0].AbsPath, Name: name, Size: sz}
		out.ZipEntries = nil
		out.DownloadName = name
		out.DisplayName = name
		added[0].Index = 0
		added[0].URL = "file/0"
	} else {
		out.Single = nil
		out.ZipEntries = base
		out.DownloadName = "airlink.zip"
		out.DisplayName = fmt.Sprintf("%d items", len(base))
	}
	return out, added, nil
}

func payloadEntries(p Payload) []ZipEntry {
	if p.Single != nil {
		return []ZipEntry{{AbsPath: p.Single.AbsPath, NameInZip: p.Single.Name}}
	}
	out := make([]ZipEntry, len(p.ZipEntries))
	copy(out, p.ZipEntries)
	return out
}
