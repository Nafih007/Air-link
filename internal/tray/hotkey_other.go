//go:build !windows

package tray

func listenScreenshotHotkey(mods, vk uint, label string, onFire func(), onErr func(error)) (stop func()) {
	return func() {}
}

func warnHotkey(err error) {}
