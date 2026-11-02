//go:build windows

package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GatherPaths consolidates multi-select Explorer launches into one process.
// Uses an exclusive lock file under the user temp dir (not a TCP port) so a
// local listener cannot hijack selected paths.
// The leader receives all paths. Followers return leader=false and should exit.
func GatherPaths(own []string) (paths []string, leader bool, err error) {
	dir, err := coalesceDir()
	if err != nil {
		return own, true, nil
	}
	lockPath := filepath.Join(dir, "leader.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Another instance is leader — drop our paths for it to pick up.
		if writeErr := writePathDrop(dir, own); writeErr != nil {
			return nil, false, fmt.Errorf("hand off paths: %w", writeErr)
		}
		return nil, false, nil
	}
	defer func() {
		_ = lock.Close()
		_ = os.Remove(lockPath)
	}()

	collected := CollectIncoming(own, MultiLaunchWindow)
	return collected, true, nil
}

// CollectIncoming merges this process's paths with JSON drops from other
// Explorer launches that arrive within wait (typically 500ms).
func CollectIncoming(own []string, wait time.Duration) []string {
	collected := append([]string{}, own...)
	if wait <= 0 {
		return uniquePreserve(collected)
	}
	collected = append(collected, PollOfferedPaths()...)
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		time.Sleep(40 * time.Millisecond)
		collected = append(collected, PollOfferedPaths()...)
	}
	collected = append(collected, PollOfferedPaths()...)
	return uniquePreserve(collected)
}

func coalesceDir() (string, error) {
	dir := filepath.Join(os.TempDir(), "airlink-coalesce")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func writePathDrop(dir string, paths []string) error {
	name := filepath.Join(dir, fmt.Sprintf("p-%d-%d.json", os.Getpid(), time.Now().UnixNano()))
	raw, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	tmp := name + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, name)
}

func readAndRemoveDrops(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var all []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "p-") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		_ = os.Remove(path)
		if err != nil {
			continue
		}
		var paths []string
		if json.Unmarshal(raw, &paths) != nil {
			continue
		}
		all = append(all, paths...)
	}
	return all, nil
}
