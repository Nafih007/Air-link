//go:build !windows

package shell

// ClaimLiveInstance is a no-op off Windows.
func ClaimLiveInstance() (bool, error) { return true, nil }

// LiveInstanceExists is false off Windows.
func LiveInstanceExists() bool { return false }

// ReleaseLiveInstance is a no-op off Windows.
func ReleaseLiveInstance() {}

// OfferPaths is a no-op off Windows.
func OfferPaths([]string) error { return nil }

// PollOfferedPaths is a no-op off Windows.
func PollOfferedPaths() []string { return nil }

// FocusMainWindow is a no-op off Windows.
func FocusMainWindow() {}
