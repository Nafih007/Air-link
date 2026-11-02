package server

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Payload is what one session will send to the phone.
type Payload struct {
	// DownloadName is the suggested filename (file or .zip).
	DownloadName string
	// DisplayName is shown on the phone/PC UI.
	DisplayName string
	// Size is bytes for a single file, or sum of regular-file sizes for a zip
	// (uncompressed estimate for the UI).
	Size int64
	// Single is set for one-file mode (streamed with Range support).
	Single *ResolvedFile
	// ZipEntries is set for multi-file / folder mode (streamed zip, no temp file).
	ZipEntries []ZipEntry
	// Text is set for clipboard/long-text sessions (UTF-8).
	Text string
	// ReceiveOnly is an upload-only session (Feature 3.2).
	ReceiveOnly bool
	// Room is an empty connected session (Feature 5): QR with no files yet.
	Room bool
}

// TextPayload builds a text-only session payload.
func TextPayload(text string) Payload {
	return Payload{
		DisplayName:  "Text",
		DownloadName: "text.txt",
		Size:         int64(len(text)),
		Text:         text,
	}
}

// ReceivePayload builds an upload-only session (no files for the phone to download).
func ReceivePayload() Payload {
	return Payload{
		DisplayName:  "Receive from phone",
		DownloadName: "",
		Size:         0,
		ReceiveOnly:  true,
	}
}

// RoomPayload is the no-args connected room (scan once, send later).
func RoomPayload() Payload {
	return Payload{
		DisplayName:  "Waiting for phone…",
		DownloadName: "",
		Size:         0,
		Room:         true,
	}
}

// HasDownload reports whether the phone can fetch a file or text from this session.
func (p Payload) HasDownload() bool {
	return p.Text != "" || p.Single != nil || len(p.ZipEntries) > 0
}

// ZipEntry is one file inside an on-the-fly zip.
type ZipEntry struct {
	AbsPath   string
	NameInZip string // forward slashes, no ".."
}

// ResolvePaths validates user-selected paths into a single-file or zip payload.
func ResolvePaths(paths []string) (Payload, error) {
	if len(paths) == 0 {
		return Payload{}, fmt.Errorf("no paths")
	}
	cleaned := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			return Payload{}, fmt.Errorf("empty path")
		}
		cleaned = append(cleaned, p)
	}

	if len(cleaned) == 1 {
		abs, err := absClean(cleaned[0])
		if err != nil {
			return Payload{}, err
		}
		fi, err := os.Lstat(abs)
		if err != nil {
			return Payload{}, fmt.Errorf("stat %s: %w", abs, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return Payload{}, fmt.Errorf("refusing symlink: %s", abs)
		}
		if fi.Mode().IsRegular() {
			rf := ResolvedFile{AbsPath: abs, Name: filepath.Base(abs), Size: fi.Size()}
			return Payload{
				DownloadName: rf.Name,
				DisplayName:  rf.Name,
				Size:         rf.Size,
				Single:       &rf,
			}, nil
		}
		if fi.IsDir() {
			entries, total, err := collectDir(abs)
			if err != nil {
				return Payload{}, err
			}
			if len(entries) == 0 {
				return Payload{}, fmt.Errorf("folder is empty: %s", abs)
			}
			name := filepath.Base(abs) + ".zip"
			return Payload{
				DownloadName: name,
				DisplayName:  filepath.Base(abs) + " (folder)",
				Size:         total,
				ZipEntries:   entries,
			}, nil
		}
		return Payload{}, fmt.Errorf("unsupported path type: %s", abs)
	}

	var entries []ZipEntry
	var total int64
	usedNames := map[string]int{}
	for _, p := range cleaned {
		abs, err := absClean(p)
		if err != nil {
			return Payload{}, err
		}
		fi, err := os.Lstat(abs)
		if err != nil {
			return Payload{}, fmt.Errorf("stat %s: %w", abs, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return Payload{}, fmt.Errorf("refusing symlink: %s", abs)
		}
		if fi.Mode().IsRegular() {
			name := uniqueZipName(filepath.Base(abs), usedNames)
			entries = append(entries, ZipEntry{AbsPath: abs, NameInZip: name})
			total += fi.Size()
			continue
		}
		if fi.IsDir() {
			sub, sum, err := collectDir(abs)
			if err != nil {
				return Payload{}, err
			}
			for _, e := range sub {
				e.NameInZip = uniqueZipName(e.NameInZip, usedNames)
				entries = append(entries, e)
			}
			total += sum
			continue
		}
		return Payload{}, fmt.Errorf("unsupported path type: %s", abs)
	}
	if len(entries) == 0 {
		return Payload{}, fmt.Errorf("nothing to send")
	}
	return Payload{
		DownloadName: "airlink.zip",
		DisplayName:  fmt.Sprintf("%d items", len(cleaned)),
		Size:         total,
		ZipEntries:   entries,
	}, nil
}

// ResolveFile keeps the single-file helper used by older tests.
func ResolveFile(userPath string) (ResolvedFile, error) {
	p, err := ResolvePaths([]string{userPath})
	if err != nil {
		return ResolvedFile{}, err
	}
	if p.Single == nil {
		return ResolvedFile{}, fmt.Errorf("not a regular file: %s", userPath)
	}
	return *p.Single, nil
}

func absClean(userPath string) (string, error) {
	abs, err := filepath.Abs(userPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return filepath.Clean(abs), nil
}

func collectDir(root string) ([]ZipEntry, int64, error) {
	var entries []ZipEntry
	var total int64
	base := filepath.Base(root)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		// Never follow symlinks (files or dirs).
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !safeZipRel(rel) {
			return fmt.Errorf("unsafe path in folder: %s", rel)
		}
		name := filepath.ToSlash(filepath.Join(base, rel))
		info, err := d.Info()
		if err != nil {
			return err
		}
		entries = append(entries, ZipEntry{AbsPath: path, NameInZip: name})
		total += info.Size()
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("walk %s: %w", root, err)
	}
	return entries, total, nil
}

// safeZipRel rejects absolute / parent segments in a relative path.
func safeZipRel(rel string) bool {
	if rel == "" || filepath.IsAbs(rel) {
		return false
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func uniqueZipName(name string, used map[string]int) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if !strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	if used[name] == 0 {
		used[name] = 1
		return name
	}
	used[name]++
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s (%d)%s", stem, used[name]-1, ext)
}
