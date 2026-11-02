package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const maxPINAttempts = 5
const pinCookie = "stp_unlock"

// NewPIN returns a cryptographically random 4-digit PIN (0000–9999).
func NewPIN() (string, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate pin: %w", err)
	}
	n := int(b[0])<<8 | int(b[1])
	return fmt.Sprintf("%04d", n%10000), nil
}

type pinState struct {
	mu     sync.Mutex
	fails  int
	locked bool
}

func (s *Server) pinEnabled() bool {
	return s.cfg.PIN != ""
}

func (s *Server) pinCookieValue() string {
	mac := hmac.New(sha256.New, []byte(s.cfg.Token))
	_, _ = mac.Write([]byte("unlock"))
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

func (s *Server) pinCookieOK(r *http.Request) bool {
	c, err := r.Cookie(pinCookie)
	if err != nil {
		return false
	}
	want := s.pinCookieValue()
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(want)) == 1
}

func (s *Server) pinLocked() bool {
	s.pin.mu.Lock()
	defer s.pin.mu.Unlock()
	return s.pin.locked
}

func (s *Server) servePIN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.pinEnabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "needed": false})
		return
	}
	if s.pinLocked() {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "locked", "left": 0})
		return
	}
	var body struct {
		PIN string `json:"pin"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 64)).Decode(&body)
	ip := clientIP(r)
	if s.rate != nil && !s.rate.allow(ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	if len(body.PIN) != 4 || !TokenValid(s.cfg.PIN, body.PIN) {
		if s.rate != nil {
			s.rate.fail(ip)
		}
		s.pin.mu.Lock()
		s.pin.fails++
		left := maxPINAttempts - s.pin.fails
		if s.pin.fails >= maxPINAttempts {
			s.pin.locked = true
			left = 0
		}
		s.pin.mu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"ok":    false,
			"error": "pin",
			"left":  left,
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     pinCookie,
		Value:    s.pinCookieValue(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
