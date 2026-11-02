//go:build !windows

package shell

import "time"

// GatherPaths is a no-op outside Windows.
func GatherPaths(own []string) (paths []string, leader bool, err error) {
	return own, true, nil
}

// CollectIncoming returns own paths off Windows.
func CollectIncoming(own []string, wait time.Duration) []string {
	_ = wait
	return uniquePreserve(own)
}
