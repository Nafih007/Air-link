package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"sendtophone/internal/pair"
)

const joinTTL = 2 * time.Minute

const (
	devPending = "pending"
	devAllowed = "allowed"
	devDenied  = "denied"
	devKicked  = "kicked"
)

type joinSlot struct {
	plain   string
	expires time.Time
}

type retiredCode struct {
	plain  string
	reason string // used | expired
	until  time.Time
}

type member struct {
	ID        string
	TokenHash string
	IP        string
	Browser   string
	Status    string
	Paired    bool
}

type accessState struct {
	mu        sync.Mutex
	join      joinSlot
	retired   []retiredCode
	autoAllow bool
	members   map[string]*member // by ID
	byHash    map[string]*member
}

func newAccess() *accessState {
	return &accessState{
		members: make(map[string]*member),
		byHash:  make(map[string]*member),
	}
}

func (s *Server) acc() *accessState {
	if s.access == nil {
		s.access = newAccess()
	}
	return s.access
}

// JoinTTL is how long an unused QR stays valid.
func JoinTTL() time.Duration { return joinTTL }

// RotateJoin invalidates the current unused join code and issues a new one.
// Connected device tokens stay valid.
func (s *Server) RotateJoin() error {
	code, err := NewToken()
	if err != nil {
		return fmt.Errorf("join code: %w", err)
	}
	now := time.Now()
	a := s.acc()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.join.plain != "" {
		reason := "used"
		if now.After(a.join.expires) {
			reason = "expired"
		}
		a.retired = append(a.retired, retiredCode{plain: a.join.plain, reason: reason, until: now.Add(joinTTL)})
	}
	a.join = joinSlot{plain: code, expires: now.Add(joinTTL)}
	a.pruneRetiredLocked(now)
	return nil
}

func (a *accessState) pruneRetiredLocked(now time.Time) {
	kept := a.retired[:0]
	for _, r := range a.retired {
		if r.until.After(now) {
			kept = append(kept, r)
		}
	}
	a.retired = kept
}

// JoinExpires is when the unused QR dies. Zero if none.
func (s *Server) JoinExpires() time.Time {
	a := s.acc()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.join.expires
}

// JoinURL is the current unused QR URL. Must not be logged.
func (s *Server) JoinURL() string {
	a := s.acc()
	a.mu.Lock()
	code := a.join.plain
	a.mu.Unlock()
	if code == "" {
		_ = s.RotateJoin()
		a.mu.Lock()
		code = a.join.plain
		a.mu.Unlock()
	}
	return s.urlPrefix() + "/j/" + code + "/"
}

// SetAutoAllow skips the Allow prompt for new joins (off by default).
func (s *Server) SetAutoAllow(on bool) {
	a := s.acc()
	a.mu.Lock()
	a.autoAllow = on
	if on {
		for _, m := range a.members {
			if m.Status == devPending {
				m.Status = devAllowed
			}
		}
	}
	a.mu.Unlock()
	s.emitDevices()
}

// AutoAllow reports the current setting.
func (s *Server) AutoAllow() bool {
	a := s.acc()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.autoAllow
}

func (s *Server) claimJoin(code, ip, ua string) (deviceToken, status string, err error) {
	if !isHex(code, 32) {
		return "", "invalid", nil
	}
	now := time.Now()
	a := s.acc()
	a.mu.Lock()
	a.pruneRetiredLocked(now)

	if reason, ok := retiredReasonLocked(a, code); ok {
		a.mu.Unlock()
		return "", reason, nil
	}

	cur := a.join.plain
	exp := a.join.expires
	if cur == "" || subtle.ConstantTimeCompare([]byte(cur), []byte(code)) != 1 {
		a.mu.Unlock()
		return "", "invalid", nil
	}
	if now.After(exp) {
		a.retired = append(a.retired, retiredCode{plain: cur, reason: "expired", until: now.Add(joinTTL)})
		a.join.plain = ""
		a.mu.Unlock()
		_ = s.RotateJoin()
		return "", "expired", nil
	}

	tok, err := pair.NewToken()
	if err != nil {
		a.mu.Unlock()
		return "", "", err
	}
	status = devPending
	if a.autoAllow {
		status = devAllowed
	}
	id := memberID(tok)
	m := &member{
		ID:        id,
		TokenHash: pair.HashToken(tok),
		IP:        ip,
		Browser:   shortBrowser(ua),
		Status:    status,
	}
	a.members[id] = m
	a.byHash[m.TokenHash] = m
	a.retired = append(a.retired, retiredCode{plain: cur, reason: "used", until: now.Add(joinTTL)})
	a.join.plain = ""
	a.mu.Unlock()

	if err := s.RotateJoin(); err != nil {
		return "", "", err
	}
	if s.cfg.Hooks.OnJoinConsumed != nil {
		s.cfg.Hooks.OnJoinConsumed()
	}
	s.emitDevices()
	out := "pending"
	if status == devAllowed {
		out = "ok"
	}
	return tok, out, nil
}

func retiredReasonLocked(a *accessState, code string) (string, bool) {
	for _, r := range a.retired {
		if subtle.ConstantTimeCompare([]byte(r.plain), []byte(code)) == 1 {
			return r.reason, true
		}
	}
	return "", false
}

func memberID(deviceToken string) string {
	sum := sha256.Sum256([]byte(deviceToken))
	return hex.EncodeToString(sum[:8])
}

func (s *Server) lookupMember(deviceToken string) *member {
	if deviceToken == "" {
		return nil
	}
	h := pair.HashToken(deviceToken)
	a := s.acc()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.byHash[h]
}

func (s *Server) emitDevices() {
	if s.cfg.Hooks.OnDevicesChanged != nil {
		s.cfg.Hooks.OnDevicesChanged(s.DeviceList())
	}
}

func isHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (s *Server) grantForTest() (string, error) {
	s.SetAutoAllow(true)
	if err := s.RotateJoin(); err != nil {
		return "", err
	}
	a := s.acc()
	a.mu.Lock()
	code := a.join.plain
	a.mu.Unlock()
	tok, st, err := s.claimJoin(code, "127.0.0.1", "GoTest")
	if err != nil {
		return "", err
	}
	if st != "ok" {
		return "", fmt.Errorf("join status %s", st)
	}
	return tok, nil
}
