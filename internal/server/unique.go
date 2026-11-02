package server

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SanitizeUploadName keeps a safe basename (no paths, no empty).
func SanitizeUploadName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "upload.bin"
	}
	return name
}

// UniquePath returns dir/name, or dir/name (1).ext, dir/name (2).ext, …
// exists reports whether a path is already taken.
func UniquePath(dir, name string, exists func(string) bool) string {
	name = SanitizeUploadName(name)
	candidate := filepath.Join(dir, name)
	if !exists(candidate) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 10_000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if !exists(candidate) {
			return candidate
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, 9999, ext))
}
