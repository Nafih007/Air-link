package tray

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// DefaultHotkey is Ctrl+Shift+P (screenshot). Off unless -hotkey is set.
const DefaultHotkey = "Ctrl+Shift+P"

const (
	modAlt      = 0x0001
	modControl  = 0x0002
	modShift    = 0x0004
	modWin      = 0x0008
	modNoRepeat = 0x4000
)

// ParseCombo parses "Ctrl+Shift+P" into RegisterHotKey modifiers and a virtual-key code.
func ParseCombo(s string) (mods uint, vk uint, label string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		s = DefaultHotkey
	}
	parts := strings.Split(s, "+")
	if len(parts) == 0 {
		return 0, 0, "", fmt.Errorf("empty hotkey")
	}
	var keys []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		low := strings.ToLower(p)
		switch low {
		case "ctrl", "control":
			mods |= modControl
			keys = append(keys, "Ctrl")
		case "shift":
			mods |= modShift
			keys = append(keys, "Shift")
		case "alt":
			mods |= modAlt
			keys = append(keys, "Alt")
		case "win", "windows", "meta":
			mods |= modWin
			keys = append(keys, "Win")
		default:
			v, ok := vkFromToken(p)
			if !ok {
				return 0, 0, "", fmt.Errorf("unknown hotkey key %q", p)
			}
			if vk != 0 {
				return 0, 0, "", fmt.Errorf("hotkey needs a single key, got extra %q", p)
			}
			vk = v
			keys = append(keys, strings.ToUpper(p))
		}
	}
	if vk == 0 {
		return 0, 0, "", fmt.Errorf("hotkey %q has no key", s)
	}
	return mods | uint(modNoRepeat), vk, strings.Join(keys, "+"), nil
}

func vkFromToken(p string) (uint, bool) {
	if utf8.RuneCountInString(p) == 1 {
		r := []rune(strings.ToUpper(p))[0]
		if r >= 'A' && r <= 'Z' {
			return uint(r), true
		}
		if r >= '0' && r <= '9' {
			return uint(r), true
		}
	}
	low := strings.ToLower(p)
	if strings.HasPrefix(low, "f") && len(low) <= 3 {
		var n int
		if _, err := fmt.Sscanf(low, "f%d", &n); err == nil && n >= 1 && n <= 12 {
			return uint(0x70 + n - 1), true
		}
	}
	return 0, false
}
