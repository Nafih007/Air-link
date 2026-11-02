package server

import (
	"net/http"
	"strings"
)

// Device is a phone/browser that hit this session.
type Device struct {
	ID      string `json:"id"`
	IP      string `json:"ip"`
	Browser string `json:"browser"`
	Kicked  bool   `json:"kicked"`
	Pending bool   `json:"pending"`
	Denied  bool   `json:"denied"`
	Paired  bool   `json:"paired"`
}

func shortBrowser(ua string) string {
	ua = strings.TrimSpace(ua)
	switch {
	case strings.Contains(ua, "CriOS"), strings.Contains(ua, "Chrome"):
		return "Chrome"
	case strings.Contains(ua, "FxiOS"), strings.Contains(ua, "Firefox"):
		return "Firefox"
	case strings.Contains(ua, "EdgiOS"), strings.Contains(ua, "Edg"):
		return "Edge"
	case strings.Contains(ua, "Safari"):
		return "Safari"
	case ua == "":
		return "Browser"
	default:
		if len(ua) > 40 {
			return ua[:40]
		}
		return ua
	}
}

func (s *Server) noteDevice(r *http.Request) {
	_ = r
}

// KickDevice revokes one device token. Unknown IDs are a no-op.
func (s *Server) KickDevice(id string) {
	a := s.acc()
	a.mu.Lock()
	if m, ok := a.members[id]; ok {
		m.Status = devKicked
	}
	a.mu.Unlock()
	s.emitDevices()
}

// AllowDevice accepts a pending join.
func (s *Server) AllowDevice(id string) {
	a := s.acc()
	a.mu.Lock()
	if m, ok := a.members[id]; ok && m.Status == devPending {
		m.Status = devAllowed
	}
	a.mu.Unlock()
	s.emitDevices()
}

// DenyDevice rejects a pending join.
func (s *Server) DenyDevice(id string) {
	a := s.acc()
	a.mu.Lock()
	if m, ok := a.members[id]; ok && m.Status == devPending {
		m.Status = devDenied
	}
	a.mu.Unlock()
	s.emitDevices()
}

// DeviceList is a snapshot for the PC window.
func (s *Server) DeviceList() []Device {
	a := s.acc()
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Device, 0, len(a.members))
	for _, m := range a.members {
		out = append(out, Device{
			ID:      m.ID,
			IP:      m.IP,
			Browser: m.Browser,
			Kicked:  m.Status == devKicked,
			Pending: m.Status == devPending,
			Denied:  m.Status == devDenied,
			Paired:  m.Paired,
		})
	}
	return out
}
