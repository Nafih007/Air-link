package applog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

var (
	mu   sync.Mutex
	file *os.File
	tok  = regexp.MustCompile(`/(j|t)/[0-9a-f]{32,64}`)
)

// Dir is %LOCALAPPDATA%\Air-Link\logs on Windows, or ~/.local/share/Air-Link/logs.
func Dir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	// UserCacheDir is LocalAppData on Windows.
	return filepath.Join(base, "Air-Link", "logs")
}

// Open appends to today's log file.
func Open() error {
	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return fmt.Errorf("log dir: %w", err)
	}
	name := filepath.Join(Dir(), time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	if file != nil {
		_ = file.Close()
	}
	file = f
	return nil
}

// Close the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Close()
		file = nil
	}
}

// Sanitize strips session tokens from a line.
func Sanitize(s string) string {
	return tok.ReplaceAllString(s, "/$1/…")
}

// Info writes a sanitized line.
func Info(format string, args ...any) {
	line := Sanitize(fmt.Sprintf(format, args...))
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	_, _ = fmt.Fprintf(file, "%s %s\n", time.Now().Format(time.RFC3339), line)
}

// Path of the current log file (empty if not open).
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return ""
	}
	return file.Name()
}
